package server

import (
	"context"
	"encoding/json"

	"github.com/fridencao/stardata/admin/database"
	"github.com/fridencao/stardata/admin/server/auth"
	adminv1 "github.com/fridencao/stardata/proto/gen/stardata/admin/v1"
	"github.com/fridencao/stardata/runtime/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Publishing is the moment a project's draft becomes what business users see. It is
// deliberately a distinct action from saving: a governor saves many times, then
// publishes once, and the version created at that point is immutable so it can be
// rolled back to.

func (s *Server) semanticVersionToPB(ctx context.Context, v *database.ProjectVersion, currentVersionID string) *adminv1.SemanticVersionInfo {
	if v == nil {
		return nil
	}
	out := &adminv1.SemanticVersionInfo{
		Id:        v.ID,
		Version:   int32(v.Version),
		Status:    string(v.Status),
		Note:      v.Note,
		CreatedOn: timestamppb.New(v.CreatedOn),
		IsCurrent: v.ID == currentVersionID,
	}
	if v.PublishedOn != nil {
		out.PublishedOn = timestamppb.New(*v.PublishedOn)
	}
	if v.PublishedByUserID != nil {
		out.PublishedByUserId = *v.PublishedByUserID
		// Resolve the email so history reads as names rather than UUIDs.
		if u, err := s.admin.DB.FindUser(ctx, *v.PublishedByUserID); err == nil {
			out.PublishedByUserEmail = u.Email
		}
	}
	if len(v.ValidationReport) > 0 {
		var m map[string]any
		if json.Unmarshal(v.ValidationReport, &m) == nil {
			if st, err := structpb.NewStruct(m); err == nil {
				out.ValidationReport = st
			}
		}
	}
	return out
}

func (s *Server) PublishSemanticProject(ctx context.Context, req *adminv1.PublishSemanticProjectRequest) (*adminv1.PublishSemanticProjectResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
	)

	proj, err := s.requireProjectEditor(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}
	if proj.SemanticLayerMode != "db" {
		return nil, status.Error(codes.FailedPrecondition, "this project uses the legacy archive publish flow")
	}

	claims := auditActor(auth.GetClaims(ctx))

	ver, err := s.admin.PublishProject(ctx, proj.ID, req.Note, claims)
	if err != nil {
		// A dry-run rejection returns the rejected version alongside the error. That
		// is not a server failure — it's the gate doing its job — so surface the
		// version (with its validation_report) to the UI rather than a 500. Only a
		// nil version means the publish genuinely failed.
		if ver != nil && ver.Status == database.ProjectVersionStatusRejected {
			return &adminv1.PublishSemanticProjectResponse{
				Version: s.semanticVersionToPB(ctx, ver, ""),
			}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Re-read the project so is_current reflects the freshly-set pointer.
	updated, err := s.admin.DB.FindProject(ctx, proj.ID)
	if err != nil {
		return nil, err
	}
	current := ""
	if updated.CurrentPublishedVersionID != nil {
		current = *updated.CurrentPublishedVersionID
	}

	return &adminv1.PublishSemanticProjectResponse{
		Version: s.semanticVersionToPB(ctx, ver, current),
	}, nil
}

func (s *Server) ListSemanticVersions(ctx context.Context, req *adminv1.ListSemanticVersionsRequest) (*adminv1.ListSemanticVersionsResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
	)

	proj, err := s.requireProjectEditor(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	rows, err := s.admin.DB.ListProjectVersions(ctx, proj.ID, int(req.Limit))
	if err != nil {
		return nil, err
	}

	current := ""
	if proj.CurrentPublishedVersionID != nil {
		current = *proj.CurrentPublishedVersionID
	}

	out := make([]*adminv1.SemanticVersionInfo, 0, len(rows))
	for _, v := range rows {
		out = append(out, s.semanticVersionToPB(ctx, v, current))
	}
	return &adminv1.ListSemanticVersionsResponse{Versions: out}, nil
}
