package server

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/fridencao/stardata/admin"
	"github.com/fridencao/stardata/admin/database"
	"github.com/fridencao/stardata/admin/pkg/publicemail"
	"github.com/fridencao/stardata/admin/server/auth"
	adminv1 "github.com/fridencao/stardata/proto/gen/stardata/admin/v1"
	runtimev1 "github.com/fridencao/stardata/proto/gen/stardata/runtime/v1"
	"github.com/fridencao/stardata/runtime/pkg/email"
	"github.com/fridencao/stardata/runtime/pkg/env"
	"github.com/fridencao/stardata/runtime/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const devDeplTTL = time.Hour

const prodDeplTTL = 14 * 24 * time.Hour

// defaultProdSlots and defaultDevSlots are the slot counts applied when a CreateProject
// request omits them (e.g. the UI, or older CLIs that don't pass these fields).
const defaultProdSlots = 4

const defaultDevSlots = 4

// runtimeAccessTokenTTL is the validity duration of JWTs issued for runtime access when calling GetProject.
// This TTL is not used for tokens created for internal communication between the admin and runtime services.
const runtimeAccessTokenDefaultTTL = 30 * time.Minute

// runtimeAccessTokenEmbedTTL is the validation duration of JWTs issued for embedding.
// Since low-risk embed users might not implement refresh, it defaults to a high value of 24 hours.
// It can be overridden to a lower value when issued for high-risk embed users.
const runtimeAccessTokenEmbedTTL = 24 * time.Hour

func (s *Server) ListProjectsForOrganization(ctx context.Context, req *adminv1.ListProjectsForOrganizationRequest) (*adminv1.ListProjectsForOrganizationResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
	)

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.Org)
	if err != nil {
		return nil, err
	}

	token, err := unmarshalPageToken(req.PageToken)
	if err != nil {
		return nil, err
	}
	pageSize := validPageSize(req.PageSize)

	// If user has ManageProjects, return all projects
	claims := auth.GetClaims(ctx)
	var projs []*database.Project
	if claims.OrganizationPermissions(ctx, org.ID).ManageProjects {
		projs, err = s.admin.DB.FindProjectsForOrganization(ctx, org.ID, token.Val, pageSize)
	} else if claims.OwnerType() == auth.OwnerTypeUser {
		// Get projects the user is a (direct or group) member of, plus all public projects.
		projs, err = s.admin.DB.FindProjectsForOrgAndUser(ctx, org.ID, claims.OwnerID(), true, true, token.Val, pageSize)
	} else {
		projs, err = s.admin.DB.FindPublicProjectsInOrganization(ctx, org.ID, token.Val, pageSize)
	}
	if err != nil {
		return nil, err
	}

	// If no projects are public, and user is not an outside member of any projects, the projsMap is empty.
	// If additionally, the user is not an org member, return permission denied (instead of an empty slice).
	if len(projs) == 0 && !claims.OrganizationPermissions(ctx, org.ID).ReadProjects {
		return nil, status.Error(codes.PermissionDenied, "does not have permission to read projects")
	}

	nextToken := ""
	if len(projs) >= pageSize {
		nextToken = marshalPageToken(projs[len(projs)-1].Name)
	}

	dtos := make([]*adminv1.Project, len(projs))
	for i, p := range projs {
		dtos[i] = s.projToDTO(p, org.Name)
	}

	return &adminv1.ListProjectsForOrganizationResponse{
		Projects:      dtos,
		NextPageToken: nextToken,
	}, nil
}

func (s *Server) ListProjectsForOrganizationAndUser(ctx context.Context, req *adminv1.ListProjectsForOrganizationAndUserRequest) (*adminv1.ListProjectsForOrganizationAndUserResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.organization", req.Org),
		attribute.String("args.user_id", req.UserId),
	)

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.Org)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	if !claims.OrganizationPermissions(ctx, org.ID).ReadOrgMembers {
		return nil, status.Error(codes.PermissionDenied, "does not have permission to read org members")
	}

	pageToken, err := unmarshalPageToken(req.PageToken)
	if err != nil {
		return nil, err
	}
	pageSize := validPageSize(req.PageSize)

	projects, err := s.admin.DB.FindProjectsForOrgAndUser(ctx, org.ID, req.UserId, false, !req.Direct, pageToken.Val, pageSize)
	if err != nil {
		return nil, err
	}

	nextToken := ""
	if len(projects) >= pageSize {
		nextToken = marshalPageToken(projects[len(projects)-1].Name)
	}

	dtos := make([]*adminv1.Project, len(projects))
	for i, p := range projects {
		dtos[i] = s.projToDTO(p, org.Name)
	}

	var projectRoles map[string]*adminv1.ProjectMemberUser
	if req.IncludeRoles && len(projects) > 0 {
		ids := make([]string, len(projects))
		for i, p := range projects {
			ids[i] = p.ID
		}
		members, err := s.admin.DB.FindProjectMemberUsersForUserAndProjects(ctx, req.UserId, ids)
		if err != nil {
			return nil, err
		}
		projectRoles = make(map[string]*adminv1.ProjectMemberUser, len(members))
		for projectID, m := range members {
			projectRoles[projectID] = projMemberUserToPB(m)
		}
	}

	return &adminv1.ListProjectsForOrganizationAndUserResponse{
		Projects:      dtos,
		NextPageToken: nextToken,
		ProjectRoles:  projectRoles,
	}, nil
}

func (s *Server) ListProjectsForFingerprint(ctx context.Context, req *adminv1.ListProjectsForFingerprintRequest) (*adminv1.ListProjectsForFingerprintResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.directory_name", req.DirectoryName),
		attribute.String("args.git_remote", req.GitRemote),
		attribute.String("args.sub_path", req.SubPath),
		attribute.String("args.rill_mgd_git_remote", req.RillMgdGitRemote),
	)

	claims := auth.GetClaims(ctx)
	if claims.OwnerType() != auth.OwnerTypeUser {
		return nil, status.Error(codes.PermissionDenied, "only users can list projects by fingerprint")
	}
	userID := claims.OwnerID()

	projects, err := s.admin.DB.FindProjectsForUserAndFingerprint(ctx, userID, req.DirectoryName, normalizeGitRemote(req.GitRemote), req.SubPath, req.RillMgdGitRemote)
	if err != nil {
		return nil, err
	}

	if len(projects) == 0 && req.GitRemote != "" {
		// if no project is found check if there is project user doesn't have access to
		projects, err = s.admin.DB.FindProjectsByGitRemote(ctx, normalizeGitRemote(req.GitRemote))
		if err != nil {
			return nil, err
		}
		for _, p := range projects {
			if p.Subpath != req.SubPath {
				continue
			}
			org, err := s.admin.DB.FindOrganization(ctx, p.OrganizationID)
			if err != nil {
				return nil, err
			}
			return &adminv1.ListProjectsForFingerprintResponse{
				UnauthorizedProject: fmt.Sprintf("%s/%s", org.Name, p.Name),
			}, nil
		}
		return &adminv1.ListProjectsForFingerprintResponse{}, nil
	}

	dtos := make([]*adminv1.Project, len(projects))
	orgNames := make(map[string]string)
	for i, p := range projects {
		orgName := orgNames[p.OrganizationID]
		if orgName == "" {
			org, err := s.admin.DB.FindOrganization(ctx, p.OrganizationID)
			if err != nil {
				return nil, err
			}
			orgName = org.Name
			orgNames[p.OrganizationID] = orgName
		}

		dtos[i] = s.projToDTO(p, orgName)
	}

	return &adminv1.ListProjectsForFingerprintResponse{
		Projects: dtos,
	}, nil
}

func (s *Server) ListProjectsForUserByName(ctx context.Context, req *adminv1.ListProjectsForUserByNameRequest) (*adminv1.ListProjectsForUserByNameResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.project", req.Name),
	)

	claims := auth.GetClaims(ctx)
	userID := claims.OwnerID()

	projects, err := s.admin.DB.FindProjectsByNameAndUser(ctx, req.Name, userID)
	if err != nil {
		return nil, err
	}

	orgsByID := make(map[string]*database.Organization)

	dtos := make([]*adminv1.Project, len(projects))
	for i, p := range projects {
		org, hasOrg := orgsByID[p.OrganizationID]
		if !hasOrg {
			org, err = s.admin.DB.FindOrganization(ctx, p.OrganizationID)
			if err != nil {
				return nil, err
			}
			orgsByID[p.OrganizationID] = org
		}

		dtos[i] = s.projToDTO(p, org.Name)
	}

	return &adminv1.ListProjectsForUserByNameResponse{
		Projects: dtos,
	}, nil
}

func (s *Server) GetProject(ctx context.Context, req *adminv1.GetProjectRequest) (*adminv1.GetProjectResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
	)
	if req.Branch != "" {
		observability.AddRequestAttributes(ctx, attribute.String("args.branch", req.Branch))
	}

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.Org)
	if err != nil {
		return nil, err
	}

	proj, err := s.admin.DB.FindProjectByName(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	permissions := claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID)
	forceAccess := claims.Superuser(ctx) && req.SuperuserForceAccess
	if proj.Public {
		permissions.ReadProject = true
		permissions.ReadProd = true
	}
	if forceAccess {
		permissions.ReadProject = true
		permissions.ReadProd = true
		permissions.ReadProdStatus = true
		permissions.ReadDev = true
		permissions.ReadDevStatus = true
		permissions.ReadProvisionerResources = true
		permissions.ReadProjectMembers = true
	}

	if !permissions.ReadProject {
		return nil, status.Error(codes.PermissionDenied, "does not have permission to read project")
	}

	var depl *database.Deployment
	if req.Branch != "" {
		depls, err := s.admin.DB.FindDeploymentsForProject(ctx, proj.ID, "", req.Branch)
		if err != nil {
			return nil, err
		}
		if len(depls) == 0 {
			return nil, status.Errorf(codes.NotFound, "no deployment found for branch %q", req.Branch)
		} else if len(depls) > 1 {
			return nil, status.Errorf(codes.FailedPrecondition, "multiple deployments found for branch %q. Recreate deployments to resolve", req.Branch)
		}
		depl = depls[0]
	} else {
		if proj.PrimaryDeploymentID == nil {
			return &adminv1.GetProjectResponse{
				Project:            s.projToDTO(proj, org.Name),
				ProjectPermissions: permissions,
			}, nil
		}

		depl, err = s.admin.DB.FindDeployment(ctx, *proj.PrimaryDeploymentID)
		if err != nil {
			return nil, err
		}
	}

	if depl.Environment == "dev" {
		if !permissions.ReadDev {
			return nil, status.Error(codes.PermissionDenied, "does not have permission to read dev deployment")
		}
		if !permissions.ReadDevStatus {
			depl.StatusMessage = ""
		}
	} else {
		if !permissions.ReadProd {
			return nil, status.Error(codes.PermissionDenied, "does not have permission to read prod deployment")
		}
		if !permissions.ReadProdStatus {
			depl.StatusMessage = ""
		}
	}

	ttlDuration := runtimeAccessTokenDefaultTTL
	if req.AccessTokenTtlSeconds != 0 {
		ttlDuration = time.Duration(req.AccessTokenTtlSeconds) * time.Second
	}

	jwt, err := s.issueRuntimeToken(ctx, &issueRuntimeTokenOptions{
		project:            proj,
		deployment:         depl,
		projectPermissions: permissions,
		forOwner:           true,
		grantManageAll:     req.IssueSuperuserToken,
		ttl:                ttlDuration,
	})
	if err != nil {
		return nil, err
	}

	s.admin.Used.Deployment(depl.ID)

	return &adminv1.GetProjectResponse{
		Project:            s.projToDTO(proj, org.Name),
		Deployment:         deploymentToDTO(depl),
		Jwt:                jwt,
		ProjectPermissions: permissions,
	}, nil
}

func (s *Server) GetProjectByID(ctx context.Context, req *adminv1.GetProjectByIDRequest) (*adminv1.GetProjectByIDResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.project_id", req.Id),
	)

	proj, err := s.admin.DB.FindProject(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	org, err := s.admin.DB.FindOrganization(ctx, proj.OrganizationID)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	permissions := claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID)
	if !permissions.ReadProject && !proj.Public && !claims.Superuser(ctx) {
		return nil, status.Error(codes.PermissionDenied, "does not have permission to read project")
	}

	return &adminv1.GetProjectByIDResponse{
		Project: s.projToDTO(proj, org.Name),
	}, nil
}

func (s *Server) SearchProjectNames(ctx context.Context, req *adminv1.SearchProjectNamesRequest) (*adminv1.SearchProjectNamesResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.pattern", req.NamePattern),
		attribute.Int("args.annotations", len(req.Annotations)),
	)

	claims := auth.GetClaims(ctx)
	if !claims.Superuser(ctx) {
		return nil, status.Error(codes.PermissionDenied, "only superusers can search projects")
	}

	token, err := unmarshalPageToken(req.PageToken)
	if err != nil {
		return nil, err
	}
	pageSize := validPageSize(req.PageSize)

	var projectNames []string
	if len(req.Annotations) > 0 {
		// If an annotation is set to "*", we just check for key presence (instead of exact key-value match)
		var annotationKeys []string
		for k, v := range req.Annotations {
			if v == "*" {
				annotationKeys = append(annotationKeys, k)
				delete(req.Annotations, k)
			}
		}

		projectNames, err = s.admin.DB.FindProjectPathsByPatternAndAnnotations(ctx, req.NamePattern, token.Val, annotationKeys, req.Annotations, pageSize)
	} else {
		projectNames, err = s.admin.DB.FindProjectPathsByPattern(ctx, req.NamePattern, token.Val, pageSize)
	}
	if err != nil {
		return nil, err
	}

	nextToken := ""
	if len(projectNames) >= pageSize {
		nextToken = marshalPageToken(projectNames[len(projectNames)-1])
	}

	return &adminv1.SearchProjectNamesResponse{
		Names:         projectNames,
		NextPageToken: nextToken,
	}, nil
}

func (s *Server) CreateProject(ctx context.Context, req *adminv1.CreateProjectRequest) (*adminv1.CreateProjectResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
		attribute.String("args.description", req.Description),
		attribute.Bool("args.public", req.Public),
		attribute.String("args.directory_name", req.DirectoryName),
		attribute.String("args.provisioner", req.Provisioner),
		attribute.String("args.prod_version", req.ProdVersion),
		attribute.Int64("args.prod_slots", req.ProdSlots),
		attribute.Int64("args.dev_slots", req.DevSlots),
		attribute.String("args.sub_path", req.Subpath),
		attribute.String("args.primary_branch", req.PrimaryBranch),
		attribute.String("args.git_remote", req.GitRemote),
		attribute.String("args.archive_asset_id", req.ArchiveAssetId),
		attribute.Bool("args.skip_deploy", req.SkipDeploy),
	)

	// Backwards compatibility
	req.GitRemote = normalizeGitRemote(req.GitRemote)

	// Find parent org
	org, err := s.admin.DB.FindOrganizationByName(ctx, req.Org)
	if err != nil {
		return nil, err
	}

	// Check permissions
	claims := auth.GetClaims(ctx)
	if !claims.OrganizationPermissions(ctx, org.ID).CreateProjects {
		return nil, status.Error(codes.PermissionDenied, "does not have permission to create projects")
	}

	// check if org has any blocking billing errors
	err = s.admin.CheckBlockingBillingErrors(ctx, org.ID)
	if err != nil {
		if errors.Is(err, admin.ErrBlockingBillingIssue) {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, err
	}

	// Apply defaults when slots are omitted (e.g. the UI, or older CLIs that don't pass these fields).
	prodSlots := int(req.ProdSlots)
	if prodSlots == 0 {
		prodSlots = defaultProdSlots
	}
	devSlots := int(req.DevSlots)
	if devSlots == 0 {
		devSlots = defaultDevSlots
	}

	// Check projects quota
	usage, err := s.admin.DB.CountProjectsQuotaUsage(ctx, org.ID)
	if err != nil {
		return nil, err
	}
	if org.QuotaProjects >= 0 && usage.Projects >= org.QuotaProjects {
		return nil, status.Errorf(codes.FailedPrecondition, "quota exceeded: org %q is limited to %d projects", org.Name, org.QuotaProjects)
	}
	if org.QuotaSlotsPerDeployment >= 0 && prodSlots > org.QuotaSlotsPerDeployment {
		return nil, status.Errorf(codes.FailedPrecondition, "quota exceeded: org can't provision more than %d slots per deployment; contact support for larger deployments", org.QuotaSlotsPerDeployment)
	}
	if org.QuotaSlotsPerDeployment >= 0 && devSlots > org.QuotaSlotsPerDeployment {
		return nil, status.Errorf(codes.FailedPrecondition, "quota exceeded: org can't provision more than %d slots per deployment; contact support for larger deployments", org.QuotaSlotsPerDeployment)
	}
	if org.QuotaSlotsTotal >= 0 && usage.Slots+prodSlots > org.QuotaSlotsTotal {
		return nil, status.Errorf(codes.FailedPrecondition, "quota exceeded: org %q is limited to %d total slots", org.Name, org.QuotaSlotsTotal)
	}
	if org.QuotaDeployments >= 0 && usage.Deployments >= org.QuotaDeployments {
		return nil, status.Errorf(codes.FailedPrecondition, "quota exceeded: org %q is limited to %d deployments", org.Name, org.QuotaDeployments)
	}

	// Add prod TTL as 14 days if not a public project else infinite
	var prodTTL *int64
	if !req.Public {
		tmp := int64(prodDeplTTL.Seconds())
		prodTTL = &tmp
	}

	// Add dev TTL as 6 hours
	devTTL := int64(devDeplTTL.Seconds())

	// Backwards compatibility: if prod version is not set, default to "latest"
	if req.ProdVersion == "" {
		req.ProdVersion = "latest"
	}

	// Capture creating user (can be nil if created with a service token)
	var userID *string
	if claims.OwnerType() == auth.OwnerTypeUser {
		tmp := claims.OwnerID()
		userID = &tmp
	}

	// Prepare the project options
	opts := &database.InsertProjectOptions{
		OrganizationID:       org.ID,
		Name:                 req.Project,
		Description:          req.Description,
		Public:               req.Public,
		CreatedByUserID:      userID,
		DirectoryName:        req.DirectoryName,
		Provisioner:          req.Provisioner,
		ArchiveAssetID:       nil,         // Populated below
		GitRemote:            nil,         // Populated below
		GithubInstallationID: nil,         // Populated below
		GithubRepoID:         nil,         // Populated below
		ManagedGitRepoID:     nil,         // Populated below
		PrimaryBranch:        "",          // Populated below
		Subpath:              req.Subpath, // Populated below
		ProdVersion:          req.ProdVersion,
		ProdSlots:            prodSlots,
		ProdTTLSeconds:       prodTTL,
		DevSlots:             devSlots,
		DevTTLSeconds:        devTTL,
		OverrideDiskGB:       nil, // default to no override; can be set later by superusers
	}

	// Check and validate the project file source.
	// NOTE: It is allowed to create a project without a source. It will then error later when creating the deployment (which can be skipped by passing skip_deploy).
	if req.GitRemote != "" {
		return nil, status.Error(codes.InvalidArgument, "git deployments are not supported; use archive uploads instead")
	}
	if req.ArchiveAssetId != "" {
		// Check access to the archive asset
		if !s.hasAssetUsagePermission(ctx, req.ArchiveAssetId, org.ID, claims.OwnerID()) {
			return nil, status.Error(codes.PermissionDenied, "archive_asset_id is not accessible to this org")
		}
		opts.ArchiveAssetID = &req.ArchiveAssetId
	}

	// if there is no subscription for the org, submit a job to start a trial
	bi, err := s.admin.DB.FindBillingIssueByTypeForOrg(ctx, org.ID, database.BillingIssueTypeNeverSubscribed)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return nil, err
	}
	if bi != nil {
		// check against trial orgs quota but skip if the user is a superuser
		if org.CreatedByUserID != nil && !claims.Superuser(ctx) {
			u, err := s.admin.DB.FindUser(ctx, *org.CreatedByUserID)
			if err != nil {
				return nil, fmt.Errorf("failed to find user: %w", err)
			}
			if u.QuotaTrialOrgs >= 0 && u.CurrentTrialOrgsCount >= u.QuotaTrialOrgs {
				return nil, status.Errorf(codes.FailedPrecondition, "trial orgs quota exceeded for user %s", u.Email)
			}
		}
		if _, err = s.admin.Jobs.StartOrgCreditTrial(ctx, org.ID); err != nil {
			s.logger.Named("billing").Error("failed to submit job to start credit trial for org, please do it manually", zap.String("org_id", org.ID), zap.Error(err))
			// continue creating the project
		}
	}

	// Create the project
	proj, err := s.admin.CreateProject(ctx, org, opts, !req.SkipDeploy)
	if err != nil {
		return nil, err
	}

	return &adminv1.CreateProjectResponse{
		Project: s.projToDTO(proj, org.Name),
	}, nil
}

func (s *Server) DeleteProject(ctx context.Context, req *adminv1.DeleteProjectRequest) (*adminv1.DeleteProjectResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
	)

	proj, err := s.admin.DB.FindProjectByName(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	if !claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProject {
		return nil, status.Error(codes.PermissionDenied, "does not have permission to delete project")
	}

	err = s.admin.TeardownProject(ctx, proj)
	if err != nil {
		return nil, err
	}

	return &adminv1.DeleteProjectResponse{
		Id: proj.ID,
	}, nil
}

func (s *Server) UpdateProject(ctx context.Context, req *adminv1.UpdateProjectRequest) (*adminv1.UpdateProjectResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
	)
	if req.NewName != nil {
		observability.AddRequestAttributes(ctx, attribute.String("args.new_name", *req.NewName))
	}
	if req.Description != nil {
		observability.AddRequestAttributes(ctx, attribute.String("args.description", *req.Description))
	}
	if req.Public != nil {
		observability.AddRequestAttributes(ctx, attribute.Bool("args.public", *req.Public))
	}
	if req.DirectoryName != nil {
		observability.AddRequestAttributes(ctx, attribute.String("args.directory_name", *req.DirectoryName))
	}
	if req.Provisioner != nil {
		observability.AddRequestAttributes(ctx, attribute.String("args.provisioner", *req.Provisioner))
	}
	if req.ProdVersion != nil {
		observability.AddRequestAttributes(ctx, attribute.String("args.prod_version", *req.ProdVersion))
	}
	if req.PrimaryBranch != nil {
		observability.AddRequestAttributes(ctx, attribute.String("args.primary_branch", *req.PrimaryBranch))
	}
	if req.GitRemote != nil {
		observability.AddRequestAttributes(ctx, attribute.String("args.git_remote", *req.GitRemote))
	}
	if req.Subpath != nil {
		observability.AddRequestAttributes(ctx, attribute.String("args.subpath", *req.Subpath))
	}
	if req.ArchiveAssetId != nil {
		observability.AddRequestAttributes(ctx, attribute.String("args.archive_asset_id", *req.ArchiveAssetId))
	}
	if req.Public != nil {
		observability.AddRequestAttributes(ctx, attribute.Bool("args.public", *req.Public))
	}
	if req.ProdSlots != nil {
		observability.AddRequestAttributes(ctx, attribute.Int64("args.prod_slots", *req.ProdSlots))
	}
	if req.DevSlots != nil {
		observability.AddRequestAttributes(ctx, attribute.Int64("args.dev_slots", *req.DevSlots))
	}
	if req.ProdTtlSeconds != nil {
		observability.AddRequestAttributes(ctx, attribute.Int64("args.prod_ttl_seconds", *req.ProdTtlSeconds))
	}
	if req.DevTtlSeconds != nil {
		observability.AddRequestAttributes(ctx, attribute.Int64("args.dev_ttl_seconds", *req.DevTtlSeconds))
	}
	if req.OverrideDiskGb != nil {
		observability.AddRequestAttributes(ctx, attribute.Int64("args.override_disk_gb", *req.OverrideDiskGb))
	}
	if req.NewName != nil {
		observability.AddRequestAttributes(ctx, attribute.String("args.new_name", *req.NewName))
	}

	// Backwards compatibility
	if req.GitRemote != nil {
		*req.GitRemote = normalizeGitRemote(*req.GitRemote)
	}

	// Find project
	proj, err := s.admin.DB.FindProjectByName(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	forceAccess := claims.Superuser(ctx) && req.SuperuserForceAccess
	if !claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProject && !forceAccess {
		return nil, status.Error(codes.PermissionDenied, "does not have permission to manage project")
	}

	// Enforce slot quotas when ProdSlots or DevSlots is being changed (superusers bypass)
	if (req.ProdSlots != nil || req.DevSlots != nil) && !forceAccess {
		org, err := s.admin.DB.FindOrganization(ctx, proj.OrganizationID)
		if err != nil {
			return nil, err
		}
		if req.ProdSlots != nil {
			if org.QuotaSlotsPerDeployment >= 0 && int(*req.ProdSlots) > org.QuotaSlotsPerDeployment {
				return nil, status.Errorf(codes.FailedPrecondition, "quota exceeded: org can't provision more than %d slots per deployment; contact support for larger deployments", org.QuotaSlotsPerDeployment)
			}
		}
		if req.DevSlots != nil {
			if org.QuotaSlotsPerDeployment >= 0 && int(*req.DevSlots) > org.QuotaSlotsPerDeployment {
				return nil, status.Errorf(codes.FailedPrecondition, "quota exceeded: org can't provision more than %d slots per deployment; contact support for larger deployments", org.QuotaSlotsPerDeployment)
			}
		}
		if org.QuotaSlotsTotal >= 0 {
			usage, err := s.admin.DB.CountProjectsQuotaUsage(ctx, org.ID)
			if err != nil {
				return nil, err
			}
			// Calculate the delta: new slots minus current slots
			delta := 0
			if req.ProdSlots != nil {
				delta += int(*req.ProdSlots) - proj.ProdSlots
			}
			if req.DevSlots != nil {
				delta += int(*req.DevSlots) - proj.DevSlots
			}
			if delta > 0 && usage.Slots+delta > org.QuotaSlotsTotal {
				return nil, status.Errorf(codes.FailedPrecondition, "quota exceeded: org %q is limited to %d total slots; contact support for larger deployments", org.Name, org.QuotaSlotsTotal)
			}
		}
	}

	if req.GitRemote != nil && safeStr(proj.GitRemote) != *req.GitRemote {
		return nil, status.Error(codes.InvalidArgument, "git deployments are not supported; use archive uploads instead")
	}
	gitRemote := proj.GitRemote
	githubInstID := proj.GithubInstallationID
	githubRepoID := proj.GithubRepoID
	managedGitRepoID := proj.ManagedGitRepoID
	subpath := valOrDefault(req.Subpath, proj.Subpath)
	primaryBranch := valOrDefault(req.PrimaryBranch, proj.PrimaryBranch)
	archiveAssetID := proj.ArchiveAssetID

	if req.ArchiveAssetId != nil {
		archiveAssetID = req.ArchiveAssetId
		org, err := s.admin.DB.FindOrganizationByName(ctx, req.Org)
		if err != nil {
			return nil, err
		}
		if !s.hasAssetUsagePermission(ctx, *archiveAssetID, org.ID, claims.OwnerID()) {
			return nil, status.Error(codes.PermissionDenied, "archive_asset_id is not accessible to this org")
		}
		gitRemote = nil
		githubInstID = nil
		subpath = ""
		primaryBranch = ""
	}

	prodTTLSeconds := proj.ProdTTLSeconds
	if req.ProdTtlSeconds != nil {
		if *req.ProdTtlSeconds == 0 {
			prodTTLSeconds = nil
		} else {
			prodTTLSeconds = req.ProdTtlSeconds
		}
	}

	devTTLSeconds := proj.DevTTLSeconds
	if req.DevTtlSeconds != nil {
		if *req.DevTtlSeconds <= 0 {
			return nil, status.Error(codes.InvalidArgument, "dev_ttl_seconds must be greater than 0")
		}
		devTTLSeconds = *req.DevTtlSeconds
	}

	// override_disk_gb is a sudo-only field. Only allow changes when the caller is a superuser using force access.
	overrideDiskGB := proj.OverrideDiskGB
	if req.OverrideDiskGb != nil {
		if !forceAccess {
			return nil, status.Error(codes.PermissionDenied, "only superusers can set override_disk_gb")
		}
		if *req.OverrideDiskGb == 0 {
			overrideDiskGB = nil
		} else if *req.OverrideDiskGb < 0 {
			return nil, status.Error(codes.InvalidArgument, "override_disk_gb must be >= 0")
		} else {
			v := *req.OverrideDiskGb
			overrideDiskGB = &v
		}
	}

	opts := &database.UpdateProjectOptions{
		Name:                 valOrDefault(req.NewName, proj.Name),
		Description:          valOrDefault(req.Description, proj.Description),
		Public:               valOrDefault(req.Public, proj.Public),
		DirectoryName:        valOrDefault(req.DirectoryName, proj.DirectoryName),
		ArchiveAssetID:       archiveAssetID,
		GitRemote:            gitRemote,
		GithubInstallationID: githubInstID,
		GithubRepoID:         githubRepoID,
		ManagedGitRepoID:     managedGitRepoID,
		Subpath:              subpath,
		ProdVersion:          valOrDefault(req.ProdVersion, proj.ProdVersion),
		PrimaryBranch:        primaryBranch,
		PrimaryDeploymentID:  proj.PrimaryDeploymentID,
		ProdSlots:            int(valOrDefault(req.ProdSlots, int64(proj.ProdSlots))),
		ProdTTLSeconds:       prodTTLSeconds,
		DevSlots:             int(valOrDefault(req.DevSlots, int64(proj.DevSlots))),
		DevTTLSeconds:        devTTLSeconds,
		OverrideDiskGB:       overrideDiskGB,
		Provisioner:          valOrDefault(req.Provisioner, proj.Provisioner),
		Annotations:          proj.Annotations,
	}
	proj, err = s.admin.UpdateProject(ctx, proj, opts)
	if err != nil {
		return nil, err
	}

	return &adminv1.UpdateProjectResponse{
		Project: s.projToDTO(proj, req.Org),
	}, nil
}

func (s *Server) GetProjectVariables(ctx context.Context, req *adminv1.GetProjectVariablesRequest) (*adminv1.GetProjectVariablesResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
		attribute.String("args.environment", req.Environment),
		attribute.Bool("args.for_all_environments", req.ForAllEnvironments),
	)

	proj, err := s.admin.DB.FindProjectByName(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	perms := claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID)
	excludeProd := false
	if !perms.ReadDevStatus && !perms.ReadProdStatus {
		return nil, status.Error(codes.PermissionDenied, "does not have permission to read variables")
	}
	if !perms.ReadProdStatus {
		if req.Environment == "prod" {
			return nil, status.Error(codes.PermissionDenied, "does not have permission to read variables for the prod environment")
		}
		excludeProd = true
	}
	// NOTE: Not explicitly checking ReadDevStatus for non-prod environments for simplicity; if you have ReadProdStatus, you're good to read variables for non-prod envs as well.

	var vars []*database.ProjectVariable
	if req.ForAllEnvironments {
		vars, err = s.admin.DB.FindProjectVariables(ctx, proj.ID, nil)
	} else {
		vars, err = s.admin.DB.FindProjectVariables(ctx, proj.ID, &req.Environment)
	}
	if err != nil {
		return nil, err
	}

	resp := &adminv1.GetProjectVariablesResponse{
		Variables:    make([]*adminv1.ProjectVariable, 0, len(vars)),
		VariablesMap: make(map[string]string, len(vars)),
	}
	for _, v := range vars {
		if excludeProd && v.Environment == "prod" {
			continue
		}
		resp.Variables = append(resp.Variables, projectVariableToDTO(v))
		// nolint:staticcheck // We still need to set it
		resp.VariablesMap[v.Name] = v.Value
	}
	return resp, nil
}

func (s *Server) UpdateProjectVariables(ctx context.Context, req *adminv1.UpdateProjectVariablesRequest) (*adminv1.UpdateProjectVariablesResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
		attribute.String("args.environment", req.Environment),
		attribute.StringSlice("args.variables", maps.Keys(req.Variables)),
		attribute.StringSlice("args.unset_variables", req.UnsetVariables),
	)
	proj, err := s.admin.DB.FindProjectByName(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	perms := claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID)
	if !perms.ManageDev && !perms.ManageProd {
		return nil, status.Error(codes.PermissionDenied, "does not have permission to update variables")
	}
	if req.Environment == "prod" && !perms.ManageProd {
		return nil, status.Error(codes.PermissionDenied, "does not have permission to update variables for the prod environment")
	}
	// NOTE: Not explicitly checking ManageDev for non-prod environments for simplicity; if you have ManageProd, you're good to manage variables for non-prod envs as well.

	var validationErr error
	for k := range req.Variables {
		if err := env.ValidateName(k); err != nil {
			validationErr = errors.Join(validationErr, err)
		}
	}
	if validationErr != nil {
		return nil, status.Error(codes.InvalidArgument, validationErr.Error())
	}

	var userID string
	if claims.OwnerType() == auth.OwnerTypeUser {
		userID = claims.OwnerID()
	}

	err = s.admin.UpdateProjectVariables(ctx, proj, req.Environment, req.Variables, req.UnsetVariables, userID)
	if err != nil {
		return nil, fmt.Errorf("variables updated failed with error %w", err)
	}

	vars, err := s.admin.DB.FindProjectVariables(ctx, proj.ID, nil)
	if err != nil {
		return nil, err
	}
	resp := &adminv1.UpdateProjectVariablesResponse{}
	for _, v := range vars {
		resp.Variables = append(resp.Variables, projectVariableToDTO(v))
	}
	return resp, nil
}

func (s *Server) GetProjectMemberUser(ctx context.Context, req *adminv1.GetProjectMemberUserRequest) (*adminv1.GetProjectMemberUserResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
		attribute.String("args.email", req.Email),
	)

	proj, err := s.admin.DB.FindProjectByName(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	if !auth.GetClaims(ctx).ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ReadProjectMembers {
		return nil, status.Error(codes.PermissionDenied, "not allowed to read project members")
	}

	user, err := s.admin.DB.FindUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, err
	}

	member, err := s.admin.DB.FindProjectMemberUser(ctx, proj.ID, user.ID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "user is not a member of the project")
		}
		return nil, err
	}

	return &adminv1.GetProjectMemberUserResponse{Member: projMemberUserToPB(member)}, nil
}

func (s *Server) ListProjectMemberUsers(ctx context.Context, req *adminv1.ListProjectMemberUsersRequest) (*adminv1.ListProjectMemberUsersResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
	)

	proj, err := s.admin.DB.FindProjectByName(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	forceAccess := claims.Superuser(ctx) && req.SuperuserForceAccess
	if !claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ReadProjectMembers && !forceAccess {
		return nil, status.Error(codes.PermissionDenied, "not authorized to read project members")
	}

	token, err := unmarshalPageToken(req.PageToken)
	if err != nil {
		return nil, err
	}
	pageSize := validPageSize(req.PageSize)

	var roleID string
	if req.Role != "" {
		role, err := s.admin.DB.FindProjectRole(ctx, req.Role)
		if err != nil {
			return nil, err
		}
		roleID = role.ID
	}

	members, err := s.admin.DB.FindProjectMemberUsers(ctx, proj.OrganizationID, proj.ID, roleID, token.Val, pageSize)
	if err != nil {
		return nil, err
	}

	nextToken := ""
	if len(members) >= pageSize {
		nextToken = marshalPageToken(members[len(members)-1].Email)
	}

	dtos := make([]*adminv1.ProjectMemberUser, len(members))
	for i, member := range members {
		dtos[i] = projMemberUserToPB(member)
	}

	return &adminv1.ListProjectMemberUsersResponse{
		Members:       dtos,
		NextPageToken: nextToken,
	}, nil
}

func (s *Server) ListProjectInvites(ctx context.Context, req *adminv1.ListProjectInvitesRequest) (*adminv1.ListProjectInvitesResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
	)

	proj, err := s.admin.DB.FindProjectByName(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	if !claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ReadProjectMembers {
		return nil, status.Error(codes.PermissionDenied, "not authorized to read project members")
	}

	token, err := unmarshalPageToken(req.PageToken)
	if err != nil {
		return nil, err
	}
	pageSize := validPageSize(req.PageSize)

	// get pending user invites for this project
	userInvites, err := s.admin.DB.FindProjectInvites(ctx, proj.ID, token.Val, pageSize)
	if err != nil {
		return nil, err
	}

	nextToken := ""
	if len(userInvites) >= pageSize {
		nextToken = marshalPageToken(userInvites[len(userInvites)-1].Email)
	}

	invitesDtos := make([]*adminv1.ProjectInvite, len(userInvites))
	for i, invite := range userInvites {
		invitesDtos[i] = projInviteToPB(invite)
	}

	return &adminv1.ListProjectInvitesResponse{
		Invites:       invitesDtos,
		NextPageToken: nextToken,
	}, nil
}

func (s *Server) AddProjectMemberUser(ctx context.Context, req *adminv1.AddProjectMemberUserRequest) (*adminv1.AddProjectMemberUserResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.email", req.Email),
		attribute.String("args.project", req.Project),
		attribute.String("args.role", req.Role),
	)
	if req.RestrictResources != nil {
		observability.AddRequestAttributes(ctx, attribute.Bool("args.restrict_resources", *req.RestrictResources))
	}
	if len(req.Resources) > 0 {
		observability.AddRequestAttributes(ctx, attribute.StringSlice("args.resources", resourcesString(req.Resources)))
	}

	proj, err := s.admin.DB.FindProjectByName(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	if !claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProjectMembers {
		return nil, status.Error(codes.PermissionDenied, "not allowed to add project members")
	}

	// Check outstanding invites quota
	count, err := s.admin.DB.CountInvitesForOrganization(ctx, proj.OrganizationID)
	if err != nil {
		return nil, err
	}
	org, err := s.admin.DB.FindOrganization(ctx, proj.OrganizationID)
	if err != nil {
		return nil, err
	}
	if org.QuotaOutstandingInvites >= 0 && count >= org.QuotaOutstandingInvites {
		return nil, status.Errorf(codes.FailedPrecondition, "quota exceeded: org %q can at most have %d outstanding invitations", org.Name, org.QuotaOutstandingInvites)
	}

	role, err := s.admin.DB.FindProjectRole(ctx, req.Role)
	if err != nil {
		return nil, err
	}
	if role.Admin && !claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProjectAdmins {
		return nil, status.Error(codes.PermissionDenied, "as a non-admin you are not allowed to assign an admin role")
	}

	var invitedByUserID, invitedByName string
	if claims.OwnerType() == auth.OwnerTypeUser {
		user, err := s.admin.DB.FindUser(ctx, claims.OwnerID())
		if err != nil && !errors.Is(err, database.ErrNotFound) {
			return nil, err
		}
		if user != nil {
			invitedByUserID = user.ID
			invitedByName = user.DisplayName
		}
	}

	keepExistingRestrictions := req.RestrictResources == nil && len(req.Resources) == 0
	restrictResources := valOrDefault(req.RestrictResources, false) || len(req.Resources) > 0
	resources := resourceNamesFromProto(req.Resources)

	user, err := s.admin.DB.FindUserByEmail(ctx, req.Email)
	if err != nil {
		if !errors.Is(err, database.ErrNotFound) {
			return nil, err
		}
		// Find the guest role
		guestRole, err := s.admin.DB.FindOrganizationRole(ctx, database.OrganizationRoleNameGuest)
		if err != nil {
			return nil, err
		}

		// Insert an organization guest invite (will fail with a constraint error if an org-level invite already exists).
		// NOTE: Not using a transaction here for simplicity. The operation is idempotent and worst-case the user becomes a guest member with no access.
		err = s.admin.DB.InsertOrganizationInvite(ctx, &database.InsertOrganizationInviteOptions{
			Email:     req.Email,
			OrgID:     proj.OrganizationID,
			RoleID:    guestRole.ID,
			InviterID: invitedByUserID,
		})
		if err != nil && !errors.Is(err, database.ErrNotUnique) {
			return nil, err
		}

		// Find the organization invite
		orgInvite, err := s.admin.DB.FindOrganizationInvite(ctx, proj.OrganizationID, req.Email)
		if err != nil {
			if errors.Is(err, ctx.Err()) {
				return nil, err
			}
			return nil, fmt.Errorf("expected but failed to find organization invite: %w", err)
		}

		// Invite user to join the project
		err = s.admin.DB.InsertProjectInvite(ctx, &database.InsertProjectInviteOptions{
			Email:             req.Email,
			OrgInviteID:       orgInvite.ID,
			ProjectID:         proj.ID,
			RoleID:            role.ID,
			InviterID:         invitedByUserID,
			RestrictResources: restrictResources,
			Resources:         resources,
		})
		// continue sending an email if an invitation entry already exists
		if err != nil {
			if !errors.Is(err, database.ErrNotUnique) {
				return nil, err
			}
			invite, err := s.admin.DB.FindProjectInvite(ctx, proj.ID, req.Email)
			if err != nil {
				return nil, err
			}
			if keepExistingRestrictions {
				restrictResources = invite.RestrictResources
				resources = invite.Resources
			}
			if err := s.admin.DB.UpdateProjectInviteRole(ctx, invite.ID, role.ID, restrictResources, resources); err != nil {
				return nil, err
			}
		}

		// Send invitation email
		err = s.admin.Email.SendProjectInvite(&email.ProjectInvite{
			ToEmail:       req.Email,
			ToName:        "",
			AcceptURL:     s.admin.URLs.WithCustomDomain(org.CustomDomain).ProjectInviteAccept(org.Name, proj.Name),
			OrgName:       org.Name,
			ProjectName:   proj.Name,
			RoleName:      role.Name,
			InvitedByName: invitedByName,
		})
		if err != nil {
			return nil, err
		}

		return &adminv1.AddProjectMemberUserResponse{
			PendingSignup: true,
		}, nil
	}

	// Add or update the user to the project with the requested role and resource scope.
	err = s.admin.InsertProjectMemberUser(ctx, proj.OrganizationID, proj.ID, user.ID, role.ID, nil, restrictResources, resources)
	if err != nil {
		if !errors.Is(err, database.ErrNotUnique) {
			return nil, err
		}
		if keepExistingRestrictions {
			member, err := s.admin.DB.FindProjectMemberUser(ctx, proj.ID, user.ID)
			if err != nil {
				return nil, err
			}
			restrictResources = member.RestrictResources
			resources = member.Resources
		}
		if err := s.admin.DB.UpdateProjectMemberUserRole(ctx, proj.ID, user.ID, role.ID, restrictResources, resources); err != nil {
			return nil, err
		}
	}

	err = s.admin.Email.SendProjectAddition(&email.ProjectAddition{
		ToEmail:       req.Email,
		ToName:        "",
		OpenURL:       s.admin.URLs.WithCustomDomain(org.CustomDomain).Project(org.Name, proj.Name),
		OrgName:       org.Name,
		ProjectName:   proj.Name,
		RoleName:      role.Name,
		InvitedByName: invitedByName,
	})
	if err != nil {
		return nil, err
	}

	return &adminv1.AddProjectMemberUserResponse{
		PendingSignup: false,
	}, nil
}

func (s *Server) RemoveProjectMemberUser(ctx context.Context, req *adminv1.RemoveProjectMemberUserRequest) (*adminv1.RemoveProjectMemberUserResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
		attribute.String("args.email", req.Email),
	)

	proj, err := s.admin.DB.FindProjectByName(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	user, err := s.admin.DB.FindUserByEmail(ctx, req.Email)
	if err != nil {
		if !errors.Is(err, database.ErrNotFound) {
			return nil, err
		}

		// Only admins can remove pending invites.
		// NOTE: If we change invites to accept/decline (instead of auto-accept on signup), we need to revisit this.
		claims := auth.GetClaims(ctx)
		if !claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProjectMembers {
			return nil, status.Error(codes.PermissionDenied, "not allowed to remove project members")
		}

		// Check if there is a pending invite
		invite, err := s.admin.DB.FindProjectInvite(ctx, proj.ID, req.Email)
		if err != nil {
			return nil, err
		}

		err = s.admin.DB.DeleteProjectInvite(ctx, invite.ID)
		if err != nil {
			return nil, err
		}
		return &adminv1.RemoveProjectMemberUserResponse{}, nil
	}

	// The caller must either have ManageProjectMembers permission or be the user being removed.
	claims := auth.GetClaims(ctx)
	isManager := claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProjectMembers
	isSelf := claims.OwnerType() == auth.OwnerTypeUser && claims.OwnerID() == user.ID
	if !isManager && !isSelf {
		return nil, status.Error(codes.PermissionDenied, "not allowed to remove project members")
	}
	if !isSelf {
		currentRole, err := s.admin.DB.FindProjectMemberUserRole(ctx, proj.ID, user.ID)
		if err != nil {
			return nil, err
		}
		if currentRole.Admin && !claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProjectAdmins {
			return nil, status.Error(codes.PermissionDenied, "as a non-admin you are not allowed to remove an admin")
		}
	}

	err = s.admin.DB.DeleteProjectMemberUser(ctx, proj.ID, user.ID)
	if err != nil {
		return nil, err
	}

	s.admin.RecordAudit(ctx, &admin.AuditEventOptions{
		OrgID:       proj.OrganizationID,
		ProjectID:   &proj.ID,
		ActorUserID: auditActor(claims),
		EventType:   admin.AuditEventMemberRemove,
		TargetID:    user.ID,
		Payload:     map[string]any{"scope": "project", "email": user.Email},
	})
	return &adminv1.RemoveProjectMemberUserResponse{}, nil
}

func (s *Server) SetProjectMemberUserRole(ctx context.Context, req *adminv1.SetProjectMemberUserRoleRequest) (*adminv1.SetProjectMemberUserRoleResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.email", req.Email),
		attribute.String("args.project", req.Project),
	)
	if req.Role != nil {
		observability.AddRequestAttributes(ctx, attribute.String("args.role", *req.Role))
	}
	if req.RestrictResources != nil {
		observability.AddRequestAttributes(ctx, attribute.Bool("args.restrict_resources", *req.RestrictResources))
	}
	if len(req.Resources) > 0 {
		observability.AddRequestAttributes(ctx, attribute.StringSlice("args.resources", resourcesString(req.Resources)))
	}

	if req.Role == nil && req.RestrictResources == nil && len(req.Resources) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one of role, restrict_resources, or resources must be set")
	}

	proj, err := s.admin.DB.FindProjectByName(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	if !claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProjectMembers {
		return nil, status.Error(codes.PermissionDenied, "not allowed to set project member roles")
	}

	user, err := s.admin.DB.FindUserByEmail(ctx, req.Email)
	if err != nil {
		if !errors.Is(err, database.ErrNotFound) {
			return nil, err
		}
		// Check if there is a pending invite for this user
		invite, err := s.admin.DB.FindProjectInvite(ctx, proj.ID, req.Email)
		if err != nil {
			return nil, err
		}
		var role *database.ProjectRole
		if req.Role == nil {
			// keep existing role
			role, err = s.admin.DB.FindProjectRoleByID(ctx, invite.ProjectRoleID)
			if err != nil {
				return nil, err
			}
		} else {
			role, err = s.admin.DB.FindProjectRole(ctx, *req.Role)
			if err != nil {
				return nil, err
			}
			if role.Admin && !claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProjectAdmins {
				return nil, status.Error(codes.PermissionDenied, "as a non-admin you are not allowed to assign an admin role")
			}
		}

		var restrictResources bool
		var resources []database.ResourceName
		keepExistingRestrictions := req.RestrictResources == nil && len(req.Resources) == 0
		if keepExistingRestrictions {
			restrictResources = invite.RestrictResources
			resources = invite.Resources
		} else {
			restrictResources = valOrDefault(req.RestrictResources, false) || len(req.Resources) > 0
			resources = resourceNamesFromProto(req.Resources)
		}
		err = s.admin.DB.UpdateProjectInviteRole(ctx, invite.ID, role.ID, restrictResources, resources)
		if err != nil {
			return nil, err
		}
		return &adminv1.SetProjectMemberUserRoleResponse{}, nil
	}

	currentRole, err := s.admin.DB.FindProjectMemberUserRole(ctx, proj.ID, user.ID)
	if err != nil {
		return nil, err
	}
	if currentRole.Admin && !claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProjectAdmins {
		return nil, status.Error(codes.PermissionDenied, "as a non-admin you are not allowed to remove an admin")
	}

	var role *database.ProjectRole
	if req.Role == nil {
		// keep existing role
		role = currentRole
	} else {
		role, err = s.admin.DB.FindProjectRole(ctx, *req.Role)
		if err != nil {
			return nil, err
		}
		if role.Admin && !claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProjectAdmins {
			return nil, status.Error(codes.PermissionDenied, "as a non-admin you are not allowed to assign an admin role")
		}
	}

	keepExistingRestrictions := req.RestrictResources == nil && len(req.Resources) == 0
	var restrictResources bool
	var resources []database.ResourceName
	if keepExistingRestrictions {
		member, err := s.admin.DB.FindProjectMemberUser(ctx, proj.ID, user.ID)
		if err != nil {
			return nil, err
		}
		restrictResources = member.RestrictResources
		resources = member.Resources
	} else {
		restrictResources = valOrDefault(req.RestrictResources, false) || len(req.Resources) > 0
		resources = resourceNamesFromProto(req.Resources)
	}

	err = s.admin.DB.UpdateProjectMemberUserRole(ctx, proj.ID, user.ID, role.ID, restrictResources, resources)
	if err != nil {
		return nil, err
	}

	s.admin.RecordAudit(ctx, &admin.AuditEventOptions{
		OrgID:       proj.OrganizationID,
		ProjectID:   &proj.ID,
		ActorUserID: auditActor(claims),
		EventType:   admin.AuditEventMemberRoleChange,
		TargetID:    user.ID,
		Payload: map[string]any{
			"scope":              "project",
			"email":              user.Email,
			"role":               role.Name,
			"restrict_resources": restrictResources,
		},
	})

	return &adminv1.SetProjectMemberUserRoleResponse{}, nil
}

func (s *Server) GetCloneCredentials(ctx context.Context, req *adminv1.GetCloneCredentialsRequest) (*adminv1.GetCloneCredentialsResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
	)

	proj, err := s.admin.DB.FindProjectByName(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	forceAccess := claims.Superuser(ctx) && req.SuperuserForceAccess
	if !claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProject && !forceAccess {
		// neither a superuser nor can manage the project
		return nil, status.Error(codes.PermissionDenied, "does not have permission to get clone credentials")
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
		return &adminv1.GetCloneCredentialsResponse{ArchiveDownloadUrl: downloadURL}, nil
	}

	return nil, status.Error(codes.FailedPrecondition, "project does not have an uploaded archive")
}

func (s *Server) RequestProjectAccess(ctx context.Context, req *adminv1.RequestProjectAccessRequest) (*adminv1.RequestProjectAccessResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
	)

	proj, err := s.admin.DB.FindProjectByName(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	projectPermissions := claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID)
	if (req.Role != database.ProjectRoleNameAdmin && projectPermissions.ReadProject) ||
		(req.Role == database.ProjectRoleNameAdmin && projectPermissions.ManageProject) {
		return nil, status.Error(codes.FailedPrecondition, "already have access to project")
	}

	if claims.OwnerType() != auth.OwnerTypeUser {
		return nil, status.Error(codes.PermissionDenied, "only users can request access")
	}

	user, err := s.admin.DB.FindUser(ctx, claims.OwnerID())
	if err != nil {
		return nil, err
	}

	org, err := s.admin.DB.FindOrganization(ctx, proj.OrganizationID)
	if err != nil {
		return nil, err
	}

	existing, err := s.admin.DB.FindProjectAccessRequest(ctx, proj.ID, user.ID)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, status.Error(codes.AlreadyExists, "have already requested access to project")
	}

	accessReq, err := s.admin.DB.InsertProjectAccessRequest(ctx, &database.InsertProjectAccessRequestOptions{
		UserID:    user.ID,
		ProjectID: proj.ID,
	})
	if err != nil {
		return nil, err
	}

	admins, err := s.admin.DB.FindOrganizationMembersWithManageUsersRole(ctx, proj.OrganizationID)
	if err != nil {
		return nil, err
	}

	for _, u := range admins {
		err = s.admin.Email.SendProjectAccessRequest(&email.ProjectAccessRequest{
			ToEmail:     u.Email,
			ToName:      u.DisplayName,
			Email:       user.Email,
			OrgName:     org.Name,
			ProjectName: proj.Name,
			Role:        req.Role,
			ApproveLink: s.admin.URLs.WithCustomDomain(org.CustomDomain).ApproveProjectAccess(org.Name, proj.Name, accessReq.ID, req.Role),
			DenyLink:    s.admin.URLs.WithCustomDomain(org.CustomDomain).DenyProjectAccess(org.Name, proj.Name, accessReq.ID),
		})
		if err != nil {
			return nil, err
		}
	}

	return &adminv1.RequestProjectAccessResponse{}, nil
}

func (s *Server) GetProjectAccessRequest(ctx context.Context, req *adminv1.GetProjectAccessRequestRequest) (*adminv1.GetProjectAccessRequestResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.id", req.Id),
	)

	accessReq, err := s.admin.DB.FindProjectAccessRequestByID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	proj, err := s.admin.DB.FindProject(ctx, accessReq.ProjectID)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	// for now only admins can view these.
	if !claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProjectMembers {
		return nil, status.Error(codes.PermissionDenied, "not allowed to view project access request")
	}

	user, err := s.admin.DB.FindUser(ctx, accessReq.UserID)
	if err != nil {
		return nil, err
	}

	return &adminv1.GetProjectAccessRequestResponse{Email: user.Email}, nil
}

func (s *Server) ApproveProjectAccess(ctx context.Context, req *adminv1.ApproveProjectAccessRequest) (*adminv1.ApproveProjectAccessResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.id", req.Id),
	)

	accessReq, err := s.admin.DB.FindProjectAccessRequestByID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	proj, err := s.admin.DB.FindProject(ctx, accessReq.ProjectID)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	if !claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProjectMembers {
		return nil, status.Error(codes.PermissionDenied, "not allowed to set project member roles")
	}

	user, err := s.admin.DB.FindUser(ctx, accessReq.UserID)
	if err != nil {
		return nil, err
	}

	org, err := s.admin.DB.FindOrganization(ctx, proj.OrganizationID)
	if err != nil {
		return nil, err
	}

	role, err := s.admin.DB.FindProjectRole(ctx, req.Role)
	if err != nil {
		return nil, err
	}
	if role.Admin && !claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProjectAdmins {
		return nil, status.Error(codes.PermissionDenied, "as a non-admin you are not allowed to assign an admin role")
	}

	ok, err := s.admin.DB.CheckUserIsAProjectMember(ctx, user.ID, proj.ID)
	if err != nil {
		return nil, err
	}

	if ok {
		// User is already a project member, update the role, keep existing resource restrictions.
		member, err := s.admin.DB.FindProjectMemberUser(ctx, proj.ID, user.ID)
		if err != nil {
			return nil, err
		}

		err = s.admin.DB.UpdateProjectMemberUserRole(ctx, proj.ID, user.ID, role.ID, member.RestrictResources, member.Resources)
		if err != nil {
			return nil, err
		}
	} else {
		// Add the user as a project member.
		err = s.admin.InsertProjectMemberUser(ctx, proj.OrganizationID, proj.ID, user.ID, role.ID, nil, false, nil)
		if err != nil {
			return nil, err
		}
	}

	// Remove the access request.
	err = s.admin.DB.DeleteProjectAccessRequest(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	err = s.admin.Email.SendProjectAccessGranted(&email.ProjectAccessGranted{
		ToEmail:     user.Email,
		ToName:      user.DisplayName,
		OpenURL:     s.admin.URLs.WithCustomDomain(org.CustomDomain).Project(org.Name, proj.Name),
		OrgName:     org.Name,
		ProjectName: proj.Name,
	})
	if err != nil {
		return nil, err
	}

	return &adminv1.ApproveProjectAccessResponse{}, nil
}

func (s *Server) DenyProjectAccess(ctx context.Context, req *adminv1.DenyProjectAccessRequest) (*adminv1.DenyProjectAccessResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.id", req.Id),
	)

	accessReq, err := s.admin.DB.FindProjectAccessRequestByID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	proj, err := s.admin.DB.FindProject(ctx, accessReq.ProjectID)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	if !claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProjectMembers {
		return nil, status.Error(codes.PermissionDenied, "not allowed to set project member roles")
	}

	user, err := s.admin.DB.FindUser(ctx, accessReq.UserID)
	if err != nil {
		return nil, err
	}
	org, err := s.admin.DB.FindOrganization(ctx, proj.OrganizationID)
	if err != nil {
		return nil, err
	}

	// remove the invitation
	err = s.admin.DB.DeleteProjectAccessRequest(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	err = s.admin.Email.SendProjectAccessRejected(&email.ProjectAccessRejected{
		ToEmail:     user.Email,
		ToName:      user.DisplayName,
		OrgName:     org.Name,
		ProjectName: proj.Name,
	})
	if err != nil {
		return nil, err
	}

	return &adminv1.DenyProjectAccessResponse{}, nil
}

// SudoUpdateTags updates the tags for a project in organization for superusers
func (s *Server) SudoUpdateAnnotations(ctx context.Context, req *adminv1.SudoUpdateAnnotationsRequest) (*adminv1.SudoUpdateAnnotationsResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
		attribute.Int("args.annotations", len(req.Annotations)),
	)

	// Check the request is made by a superuser
	claims := auth.GetClaims(ctx)
	if !claims.Superuser(ctx) {
		return nil, status.Error(codes.PermissionDenied, "not authorized to update annotations")
	}

	proj, err := s.admin.DB.FindProjectByName(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	proj, err = s.admin.UpdateProject(ctx, proj, &database.UpdateProjectOptions{
		Name:                 proj.Name,
		Description:          proj.Description,
		Public:               proj.Public,
		DirectoryName:        proj.DirectoryName,
		ArchiveAssetID:       proj.ArchiveAssetID,
		GitRemote:            proj.GitRemote,
		GithubInstallationID: proj.GithubInstallationID,
		GithubRepoID:         proj.GithubRepoID,
		ManagedGitRepoID:     proj.ManagedGitRepoID,
		ProdVersion:          proj.ProdVersion,
		PrimaryBranch:        proj.PrimaryBranch,
		Subpath:              proj.Subpath,
		PrimaryDeploymentID:  proj.PrimaryDeploymentID,
		ProdSlots:            proj.ProdSlots,
		ProdTTLSeconds:       proj.ProdTTLSeconds,
		DevSlots:             proj.DevSlots,
		DevTTLSeconds:        proj.DevTTLSeconds,
		OverrideDiskGB:       proj.OverrideDiskGB,
		Provisioner:          proj.Provisioner,
		Annotations:          req.Annotations,
	})
	if err != nil {
		return nil, err
	}

	return &adminv1.SudoUpdateAnnotationsResponse{
		Project: s.projToDTO(proj, req.Org),
	}, nil
}

func (s *Server) CreateProjectWhitelistedDomain(ctx context.Context, req *adminv1.CreateProjectWhitelistedDomainRequest) (*adminv1.CreateProjectWhitelistedDomainResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.organization", req.Org),
		attribute.String("args.project", req.Project),
		attribute.String("args.domain", req.Domain),
		attribute.String("args.role", req.Role),
	)

	claims := auth.GetClaims(ctx)
	if claims.OwnerType() != auth.OwnerTypeUser {
		return nil, status.Error(codes.Unauthenticated, "not authenticated as a user")
	}

	proj, err := s.admin.DB.FindProjectByName(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	if !claims.Superuser(ctx) {
		if !claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProject {
			return nil, status.Error(codes.PermissionDenied, "only proj admins can add whitelisted domain")
		}
		// check if the user's domain matches the whitelist domain
		user, err := s.admin.DB.FindUser(ctx, claims.OwnerID())
		if err != nil {
			return nil, err
		}
		if !strings.HasSuffix(user.Email, "@"+req.Domain) {
			return nil, status.Error(codes.PermissionDenied, "Domain name doesn’t match verified email domain. Please contact Rill support.")
		}

		if publicemail.IsPublic(req.Domain) {
			return nil, status.Errorf(codes.InvalidArgument, "Public Domain %s cannot be whitelisted", req.Domain)
		}
	}

	role, err := s.admin.DB.FindProjectRole(ctx, req.Role)
	if err != nil {
		return nil, err
	}
	if role.Admin && !claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProjectAdmins {
		return nil, status.Error(codes.PermissionDenied, "as a non-admin you are not allowed to assign an admin role")
	}

	// find existing users belonging to the whitelisted domain to the project
	users, err := s.admin.DB.FindUsersByEmailPattern(ctx, "%@"+req.Domain, "", math.MaxInt)
	if err != nil {
		return nil, err
	}

	// filter out users who are already members of the project
	newUsers := make([]*database.User, 0)
	for _, user := range users {
		// check if user is already a member of the project
		exists, err := s.admin.DB.CheckUserIsAProjectMember(ctx, user.ID, proj.ID)
		if err != nil {
			return nil, err
		}
		if !exists {
			newUsers = append(newUsers, user)
		}
	}

	ctx, tx, err := s.admin.DB.NewTx(ctx, false)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = s.admin.DB.InsertProjectWhitelistedDomain(ctx, &database.InsertProjectWhitelistedDomainOptions{
		ProjectID:     proj.ID,
		ProjectRoleID: role.ID,
		Domain:        req.Domain,
	})
	if err != nil {
		return nil, err
	}

	for _, user := range newUsers {
		// Add the user to the project.
		err = s.admin.InsertProjectMemberUser(ctx, proj.OrganizationID, proj.ID, user.ID, role.ID, nil, false, nil)
		if err != nil {
			return nil, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &adminv1.CreateProjectWhitelistedDomainResponse{}, nil
}

func (s *Server) RemoveProjectWhitelistedDomain(ctx context.Context, req *adminv1.RemoveProjectWhitelistedDomainRequest) (*adminv1.RemoveProjectWhitelistedDomainResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.organization", req.Org),
		attribute.String("args.project", req.Project),
		attribute.String("args.domain", req.Domain),
	)

	claims := auth.GetClaims(ctx)

	proj, err := s.admin.DB.FindProjectByName(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	if !(claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProject || claims.Superuser(ctx)) {
		return nil, status.Error(codes.PermissionDenied, "only project admins can remove whitelisted domain")
	}

	invite, err := s.admin.DB.FindProjectWhitelistedDomain(ctx, proj.ID, req.Domain)
	if err != nil {
		return nil, err
	}

	err = s.admin.DB.DeleteProjectWhitelistedDomain(ctx, invite.ID)
	if err != nil {
		return nil, err
	}

	return &adminv1.RemoveProjectWhitelistedDomainResponse{}, nil
}

func (s *Server) ListProjectWhitelistedDomains(ctx context.Context, req *adminv1.ListProjectWhitelistedDomainsRequest) (*adminv1.ListProjectWhitelistedDomainsResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.organization", req.Org),
		attribute.String("args.project", req.Project),
	)

	proj, err := s.admin.DB.FindProjectByName(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	if !(claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProject || claims.Superuser(ctx)) {
		return nil, status.Error(codes.PermissionDenied, "only project admins can list whitelisted domains")
	}

	domains, err := s.admin.DB.FindProjectWhitelistedDomainForProjectWithJoinedRoleNames(ctx, proj.ID)
	if err != nil {
		return nil, err
	}

	dtos := make([]*adminv1.WhitelistedDomain, len(domains))
	for i, domain := range domains {
		dtos[i] = &adminv1.WhitelistedDomain{
			Domain: domain.Domain,
			Role:   domain.RoleName,
		}
	}

	return &adminv1.ListProjectWhitelistedDomainsResponse{Domains: dtos}, nil
}

func (s *Server) RedeployProject(ctx context.Context, req *adminv1.RedeployProjectRequest) (*adminv1.RedeployProjectResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.organization", req.Org),
		attribute.String("args.project", req.Project),
	)

	proj, err := s.admin.DB.FindProjectByName(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	forceAccess := claims.Superuser(ctx) && req.SuperuserForceAccess
	if !claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProd && !forceAccess {
		return nil, status.Error(codes.PermissionDenied, "does not have permission to manage deployment")
	}

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.Org)
	if err != nil {
		return nil, err
	}

	// check if org has blocking billing errors
	err = s.admin.CheckBlockingBillingErrors(ctx, org.ID)
	if err != nil {
		if errors.Is(err, admin.ErrBlockingBillingIssue) {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, err
	}

	var depl *database.Deployment
	if proj.PrimaryDeploymentID != nil {
		depl, err = s.admin.DB.FindDeployment(ctx, *proj.PrimaryDeploymentID)
		if err != nil {
			return nil, err
		}
	}

	_, err = s.admin.RedeployProject(ctx, proj, depl)
	if err != nil {
		return nil, err
	}

	return &adminv1.RedeployProjectResponse{}, nil
}

func (s *Server) HibernateProject(ctx context.Context, req *adminv1.HibernateProjectRequest) (*adminv1.HibernateProjectResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.organization", req.Org),
		attribute.String("args.project", req.Project),
	)

	proj, err := s.admin.DB.FindProjectByName(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	forceAccess := claims.Superuser(ctx) && req.SuperuserForceAccess
	if !claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProject && !forceAccess {
		return nil, status.Error(codes.PermissionDenied, "not allowed to manage project")
	}

	_, err = s.admin.HibernateProject(ctx, proj)
	if err != nil {
		return nil, fmt.Errorf("failed to hibernate project: %w", err)
	}

	return &adminv1.HibernateProjectResponse{}, nil
}

// Deprecated: See api.proto for details.
func (s *Server) TriggerRedeploy(ctx context.Context, req *adminv1.TriggerRedeployRequest) (*adminv1.TriggerRedeployResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.organization", req.Org),
		attribute.String("args.project", req.Project),
		attribute.String("args.deployment_id", req.DeploymentId),
	)

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.Org)
	if err != nil {
		return nil, err
	}

	// check if org has blocking billing errors
	err = s.admin.CheckBlockingBillingErrors(ctx, org.ID)
	if err != nil {
		if errors.Is(err, admin.ErrBlockingBillingIssue) {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, err
	}

	// For backwards compatibility, this RPC supports passing either DeploymentId or Organization+Project names
	var proj *database.Project
	var depl *database.Deployment
	if req.DeploymentId != "" {
		var err error
		depl, err = s.admin.DB.FindDeployment(ctx, req.DeploymentId)
		if err != nil {
			return nil, err
		}

		proj, err = s.admin.DB.FindProject(ctx, depl.ProjectID)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		proj, err = s.admin.DB.FindProjectByName(ctx, req.Org, req.Project)
		if err != nil {
			return nil, err
		}

		if proj.PrimaryDeploymentID != nil {
			depl, err = s.admin.DB.FindDeployment(ctx, *proj.PrimaryDeploymentID)
			if err != nil {
				return nil, err
			}
		}
	}

	claims := auth.GetClaims(ctx)
	if !claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProd {
		return nil, status.Error(codes.PermissionDenied, "does not have permission to manage deployment")
	}

	_, err = s.admin.RedeployProject(ctx, proj, depl)
	if err != nil {
		return nil, err
	}

	return &adminv1.TriggerRedeployResponse{}, nil
}

func (s *Server) projToDTO(p *database.Project, orgName string) *adminv1.Project {
	return &adminv1.Project{
		Id:                  p.ID,
		Name:                p.Name,
		OrgId:               p.OrganizationID,
		OrgName:             orgName,
		Description:         p.Description,
		Public:              p.Public,
		CreatedByUserId:     safeStr(p.CreatedByUserID),
		DirectoryName:       p.DirectoryName,
		Provisioner:         p.Provisioner,
		ProdVersion:         p.ProdVersion,
		ProdSlots:           int64(p.ProdSlots),
		DevSlots:            int64(p.DevSlots),
		PrimaryBranch:       p.PrimaryBranch,
		Subpath:             p.Subpath,
		GitRemote:           safeStr(p.GitRemote),
		ManagedGitId:        safeStr(p.ManagedGitRepoID),
		ArchiveAssetId:      safeStr(p.ArchiveAssetID),
		PrimaryDeploymentId: safeStr(p.PrimaryDeploymentID),
		ProdTtlSeconds:      safeInt64(p.ProdTTLSeconds),
		DevTtlSeconds:       p.DevTTLSeconds,
		OverrideDiskGb:      safeInt64(p.OverrideDiskGB),
		FrontendUrl:         s.admin.URLs.Project(orgName, p.Name),
		Annotations:         p.Annotations,
		SemanticLayerMode:   p.SemanticLayerMode,
		CreatedOn:           timestamppb.New(p.CreatedOn),
		UpdatedOn:           timestamppb.New(p.UpdatedOn),
	}
}

func (s *Server) hasAssetUsagePermission(ctx context.Context, id, orgID, ownerID string) bool {
	asset, err := s.admin.DB.FindAsset(ctx, id)
	if err != nil {
		return false
	}
	return asset.OrganizationID != nil && *asset.OrganizationID == orgID && asset.OwnerID == ownerID
}

// normalizeGitRemote adds a .git suffix to the Git remote URL if it doesn't already have one.
// If it's not a Github URL, it returns the string as is.
// This is for backwards compatibility with old CLIs that sent Github HTML URLs instead of Github remote URLs.
func normalizeGitRemote(remote string) string {
	if !strings.HasPrefix(remote, "https://github.com") {
		return remote // Not a Github remote, return as is
	}
	if strings.HasSuffix(remote, ".git") {
		return remote
	}
	return remote + ".git"
}

func deploymentToDTO(d *database.Deployment) *adminv1.Deployment {
	var s adminv1.DeploymentStatus
	switch d.Status {
	case database.DeploymentStatusUnspecified:
		s = adminv1.DeploymentStatus_DEPLOYMENT_STATUS_UNSPECIFIED
	case database.DeploymentStatusPending:
		s = adminv1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING
	case database.DeploymentStatusUpdating:
		s = adminv1.DeploymentStatus_DEPLOYMENT_STATUS_UPDATING
	case database.DeploymentStatusRunning:
		s = adminv1.DeploymentStatus_DEPLOYMENT_STATUS_RUNNING
	case database.DeploymentStatusErrored:
		s = adminv1.DeploymentStatus_DEPLOYMENT_STATUS_ERRORED
	case database.DeploymentStatusStopping:
		s = adminv1.DeploymentStatus_DEPLOYMENT_STATUS_STOPPING
	case database.DeploymentStatusStopped:
		s = adminv1.DeploymentStatus_DEPLOYMENT_STATUS_STOPPED
	case database.DeploymentStatusDeleting:
		s = adminv1.DeploymentStatus_DEPLOYMENT_STATUS_DELETING
	case database.DeploymentStatusDeleted:
		s = adminv1.DeploymentStatus_DEPLOYMENT_STATUS_DELETED
	default:
		panic(fmt.Errorf("unhandled deployment status %d", d.Status))
	}

	return &adminv1.Deployment{
		Id:                d.ID,
		ProjectId:         d.ProjectID,
		OwnerUserId:       safeStr(d.OwnerUserID),
		Environment:       d.Environment,
		Branch:            d.Branch,
		Editable:          d.Editable,
		RuntimeHost:       d.RuntimeHost,
		RuntimeInstanceId: d.RuntimeInstanceID,
		Status:            s,
		StatusMessage:     d.StatusMessage,
		CreatedOn:         timestamppb.New(d.CreatedOn),
		UpdatedOn:         timestamppb.New(d.UpdatedOn),
		UsedOn:            timestamppb.New(d.UsedOn),
	}
}

func projectVariableToDTO(v *database.ProjectVariable) *adminv1.ProjectVariable {
	return &adminv1.ProjectVariable{
		Id:              v.ID,
		Name:            v.Name,
		Value:           v.Value,
		Environment:     v.Environment,
		UpdatedByUserId: safeStr(v.UpdatedByUserID),
		CreatedOn:       timestamppb.New(v.CreatedOn),
		UpdatedOn:       timestamppb.New(v.UpdatedOn),
	}
}

func securityRulesFromResources(restricted bool, resources []database.ResourceName) []*runtimev1.SecurityRule {
	if !restricted {
		// No resource restrictions
		return nil
	}
	if len(resources) == 0 {
		// No access to any resources
		return []*runtimev1.SecurityRule{{
			Rule: &runtimev1.SecurityRule_Access{
				Access: &runtimev1.SecurityRuleAccess{
					Allow: false,
				},
			},
		}}
	}

	rules := make([]*runtimev1.SecurityRule, 0, len(resources))
	for _, r := range resources {
		if r.Type == "" || r.Name == "" {
			continue
		}
		rules = append(rules, &runtimev1.SecurityRule{
			Rule: &runtimev1.SecurityRule_TransitiveAccess{
				TransitiveAccess: &runtimev1.SecurityRuleTransitiveAccess{
					Resource: &runtimev1.ResourceName{
						Kind: r.Type,
						Name: r.Name,
					},
				},
			},
		})
	}
	return rules
}

func resourceNamesFromProto(resources []*adminv1.ResourceName) []database.ResourceName {
	res := make([]database.ResourceName, 0, len(resources))
	for _, r := range resources {
		res = append(res, database.ResourceName{Type: r.Type, Name: r.Name})
	}
	return res
}

func resourcesString(res []*adminv1.ResourceName) []string {
	var resources []string
	for _, r := range res {
		resources = append(resources, fmt.Sprintf("%s:%s", r.Type, r.Name))
	}
	return resources
}

func safeStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func safeInt64(s *int64) int64 {
	if s == nil {
		return 0
	}
	return *s
}

func valOrDefault[T any](ptr *T, def T) T {
	if ptr != nil {
		return *ptr
	}
	return def
}
