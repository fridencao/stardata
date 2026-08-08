package server

import (
	"context"
	"time"

	admin "github.com/fridencao/stardata/admin"
	"github.com/fridencao/stardata/admin/database"
	"github.com/fridencao/stardata/admin/server/auth"
	adminv1 "github.com/fridencao/stardata/proto/gen/stardata/admin/v1"
	"github.com/fridencao/stardata/runtime/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Server) GetRepoMeta(ctx context.Context, req *adminv1.GetRepoMetaRequest) (*adminv1.GetRepoMetaResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.project_id", req.ProjectId),
	)

	proj, err := s.admin.DB.FindProject(ctx, req.ProjectId)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	perms := claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID)
	if !perms.ReadProdStatus && !perms.ReadDevStatus {
		return nil, status.Error(codes.PermissionDenied, "does not have permission to read project repo")
	}

	// If the caller is a deployment, resolve it to determine editability (used by both the archive and git branches below).
	var depl *database.Deployment
	if claims.OwnerType() == auth.OwnerTypeDeployment {
		var err error
		depl, err = s.admin.DB.FindDeployment(ctx, claims.OwnerID())
		if err != nil {
			return nil, err
		}
	}

	if proj.ArchiveAssetID != nil {
		asset, err := s.admin.DB.FindAsset(ctx, *proj.ArchiveAssetID)
		if err != nil {
			return nil, err
		}

		downloadURL, err := s.generateSignedDownloadURL(asset)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		return &adminv1.GetRepoMetaResponse{
			ExpiresOn:          timestamppb.New(time.Now().Add(time.Hour * 24 * 365)), // Setting to a year because it doesn't need to be refreshed
			LastUpdatedOn:      timestamppb.New(proj.UpdatedOn),
			ArchiveId:          asset.ID,
			ArchiveDownloadUrl: downloadURL,
			ArchiveCreatedOn:   timestamppb.New(asset.CreatedOn),
			// Editable enables the runtime to maintain a writable draft copy of the archive (dev deployments).
			Editable: depl != nil && depl.Editable,
		}, nil
	}

	return nil, status.Error(codes.FailedPrecondition, "project does not have an uploaded archive")
}

func (s *Server) PullVirtualRepo(ctx context.Context, req *adminv1.PullVirtualRepoRequest) (*adminv1.PullVirtualRepoResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.project_id", req.ProjectId),
		attribute.Int("args.page_size", int(req.PageSize)),
		attribute.String("args.page_token", req.PageToken),
	)

	proj, err := s.admin.DB.FindProject(ctx, req.ProjectId)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	permissions := claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID)
	forceAccess := claims.Superuser(ctx) && req.SuperuserForceAccess
	if !permissions.ReadProdStatus && !permissions.ReadDevStatus && !forceAccess {
		return nil, status.Error(codes.PermissionDenied, "does not have permission to read project repo")
	}

	var depl *database.Deployment
	if claims.OwnerType() == auth.OwnerTypeDeployment {
		var err error
		depl, err = s.admin.DB.FindDeployment(ctx, claims.OwnerID())
		if err != nil {
			return nil, err
		}
	}

	environment := "prod"
	if depl != nil {
		environment = depl.Environment
	}

	pageToken, err := unmarshalStringTimestampPageToken(req.PageToken)
	if err != nil {
		return nil, err
	}
	pageSize := validPageSize(req.PageSize)

	// StarData Phase 5: DB-mode projects keep their semantic definitions as rows in
	// semantic_resources rather than files in an archive. Rather than teach the
	// runtime a second repo driver, admin renders those rows back into the files the
	// parser already knows how to read and ships them over the existing virtual-file
	// transport. The runtime side needs no changes.
	if proj.SemanticLayerMode == "db" {
		return s.pullSemanticResourcesAsVirtualFiles(ctx, proj.ID, pageToken.Str)
	}

	vfs, err := s.admin.DB.FindVirtualFiles(ctx, proj.ID, environment, pageToken.Ts.AsTime(), pageToken.Str, pageSize)
	if err != nil {
		return nil, err
	}

	// If no files were found, we return the same page token as the next page token.
	// This enables the client to poll for new changes continuously. (The client is responsible for pausing when an empty page is returned.)
	nextToken := req.PageToken
	if len(vfs) > 0 {
		f := vfs[len(vfs)-1]
		nextToken = marshalStringTimestampPageToken(f.Path, f.UpdatedOn)
	}

	dtos := make([]*adminv1.VirtualFile, len(vfs))
	for i, vf := range vfs {
		dtos[i] = virtualFileToDTO(vf)
	}

	return &adminv1.PullVirtualRepoResponse{
		Files:         dtos,
		NextPageToken: nextToken,
	}, nil
}

func (s *Server) GetVirtualFile(ctx context.Context, req *adminv1.GetVirtualFileRequest) (*adminv1.GetVirtualFileResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.project_id", req.ProjectId),
		attribute.String("args.path", req.Path),
	)

	proj, err := s.admin.DB.FindProject(ctx, req.ProjectId)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	permissions := claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID)
	forceAccess := claims.Superuser(ctx) && req.SuperuserForceAccess
	if !permissions.ReadProdStatus && !permissions.ReadDevStatus && !forceAccess {
		return nil, status.Error(codes.PermissionDenied, "does not have permission to read project repo")
	}

	environment := req.Environment
	if environment == "" {
		if claims.OwnerType() == auth.OwnerTypeDeployment {
			depl, err := s.admin.DB.FindDeployment(ctx, claims.OwnerID())
			if err != nil {
				return nil, err
			}
			environment = depl.Environment
		} else {
			environment = "prod"
		}
	}

	vf, err := s.admin.DB.FindVirtualFile(ctx, proj.ID, environment, req.Path)
	if err != nil {
		return nil, err
	}

	return &adminv1.GetVirtualFileResponse{
		File: virtualFileToDTO(vf),
	}, nil
}

func (s *Server) DeleteVirtualFile(ctx context.Context, req *adminv1.DeleteVirtualFileRequest) (*adminv1.DeleteVirtualFileResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.project_id", req.ProjectId),
		attribute.String("args.path", req.Path),
	)

	proj, err := s.admin.DB.FindProject(ctx, req.ProjectId)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	permissions := claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID)
	forceAccess := claims.Superuser(ctx) && req.SuperuserForceAccess
	if !permissions.ManageProd && !forceAccess {
		return nil, status.Error(codes.PermissionDenied, "does not have permission to delete virtual files")
	}

	environment := req.Environment
	if environment == "" {
		if claims.OwnerType() == auth.OwnerTypeDeployment {
			depl, err := s.admin.DB.FindDeployment(ctx, claims.OwnerID())
			if err != nil {
				return nil, err
			}
			environment = depl.Environment
		} else {
			environment = "prod"
		}
	}

	// Directly mark the virtual file as deleted without parsing
	err = s.admin.DB.UpdateVirtualFileDeleted(ctx, proj.ID, environment, req.Path)
	if err != nil {
		return nil, err
	}

	return &adminv1.DeleteVirtualFileResponse{}, nil
}

func virtualFileToDTO(vf *database.VirtualFile) *adminv1.VirtualFile {
	return &adminv1.VirtualFile{
		Path:      vf.Path,
		Data:      vf.Data,
		Deleted:   vf.Deleted,
		UpdatedOn: timestamppb.New(vf.UpdatedOn),
	}
}

// pullSemanticResourcesAsVirtualFiles renders a DB-mode project's semantic resources
// into files and returns them over the virtual-file transport.
//
// Sync protocol: the caller's page token carries the fingerprint of the resource set
// it last received. When the fingerprint still matches, nothing has changed and we
// return an empty page (which is how the runtime's watcher decides to stop polling
// and idle). When it differs, we return the full set again. Full resend rather than
// incremental diffing is deliberate — a project has tens of resources, not
// thousands, so the simplicity is worth more than the bytes saved.
//
// Phase 5.1 scope: draft resources are served to every environment because the
// published/draft split arrives with the publish pipeline in 5.2. Deleting a
// resource also does not yet retract the already-materialized file on the runtime;
// both are tracked as 5.2 work.
func (s *Server) pullSemanticResourcesAsVirtualFiles(ctx context.Context, projectID, lastFingerprint string) (*adminv1.PullVirtualRepoResponse, error) {
	fingerprint, err := s.admin.DB.FindSemanticResourceFingerprint(ctx, projectID, database.SemanticResourceStatusDraft)
	if err != nil {
		return nil, err
	}

	// Unchanged since the caller's last pull: return an empty page, same token.
	if lastFingerprint != "" && lastFingerprint == fingerprint {
		return &adminv1.PullVirtualRepoResponse{
			NextPageToken: marshalStringTimestampPageToken(fingerprint, time.Time{}),
		}, nil
	}

	resources, err := s.admin.DB.FindSemanticResources(ctx, projectID, database.SemanticResourceStatusDraft)
	if err != nil {
		return nil, err
	}

	files := make([]*adminv1.VirtualFile, 0, len(resources))
	for _, r := range resources {
		p, data, err := admin.RenderSemanticResource(r)
		if err != nil {
			// A single malformed row must not blank out the whole project. Skip it and
			// let the resource simply be absent; the save path validates on write, so
			// reaching here means the row was written before validation existed.
			s.logger.Warn("skipping unrenderable semantic resource",
				zap.String("project_id", projectID),
				zap.String("kind", r.ResourceKind),
				zap.String("name", r.ResourceName),
				zap.Error(err),
			)
			continue
		}
		files = append(files, &adminv1.VirtualFile{
			Path:      p,
			Data:      data,
			UpdatedOn: timestamppb.New(r.UpdatedOn),
		})
	}

	return &adminv1.PullVirtualRepoResponse{
		Files:         files,
		NextPageToken: marshalStringTimestampPageToken(fingerprint, time.Time{}),
	}, nil
}
