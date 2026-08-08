package server

import (
	"context"

	adminv1 "github.com/fridencao/stardata/proto/gen/stardata/admin/v1"
	admin "github.com/fridencao/stardata/admin"
	"github.com/fridencao/stardata/admin/server/auth"
	"github.com/fridencao/stardata/runtime/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// auditActor extracts a user ID pointer from claims suitable for audit logging.
func auditActor(claims auth.Claims) *string {
	if claims.OwnerType() != auth.OwnerTypeUser {
		return nil
	}
	id := claims.OwnerID()
	return &id
}

// featureKeys are the features controlled by feature access.
var featureKeys = []string{"chat", "dashboards", "reports", "alerts", "studio", "admin"}

func validFeatureKey(k string) bool {
	for _, fk := range featureKeys {
		if fk == k {
			return true
		}
	}
	return false
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (s *Server) resolveProjectID(ctx context.Context, orgName string, project *string) (*string, error) {
	if project == nil || *project == "" {
		return nil, nil
	}
	proj, err := s.admin.DB.FindProjectByName(ctx, orgName, *project)
	if err != nil {
		return nil, err
	}
	return &proj.ID, nil
}

// SetFeatureAccess sets feature access overrides for a user or user group.
func (s *Server) SetFeatureAccess(ctx context.Context, req *adminv1.SetFeatureAccessRequest) (*adminv1.SetFeatureAccessResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.subject_type", req.SubjectType),
	)
	if req.SubjectType != "user" && req.SubjectType != "group" {
		return nil, status.Error(codes.InvalidArgument, "subject_type must be 'user' or 'group'")
	}

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.Org)
	if err != nil {
		return nil, err
	}
	claims := auth.GetClaims(ctx)
	if !claims.OrganizationPermissions(ctx, org.ID).ManageOrgMembers {
		return nil, status.Error(codes.PermissionDenied, "not allowed to manage feature access")
	}

	projectID, err := s.resolveProjectID(ctx, org.Name, req.Project)
	if err != nil {
		return nil, err
	}

	createdBy := claims.OwnerID()
	changed := make(map[string]any, len(req.Features))
	for _, f := range req.Features {
		if !validFeatureKey(f.FeatureKey) {
			return nil, status.Errorf(codes.InvalidArgument, "unknown feature_key %q", f.FeatureKey)
		}
		if err := s.admin.DB.UpsertFeatureAccess(ctx, org.ID, projectID, req.SubjectType, req.SubjectId, f.FeatureKey, f.Granted, &createdBy); err != nil {
			return nil, err
		}
		changed[f.FeatureKey] = f.Granted
	}

	s.admin.RecordAudit(ctx, &admin.AuditEventOptions{
		OrgID:       org.ID,
		ProjectID:   projectID,
		ActorUserID: auditActor(claims),
		EventType:   admin.AuditEventFeatureAccessSet,
		TargetID:    req.SubjectId,
		Payload: map[string]any{
			"subject_type": req.SubjectType,
			"features":     changed,
		},
	})

	return &adminv1.SetFeatureAccessResponse{}, nil
}

// ListFeatureAccess lists feature access for an org (optionally scoped to a project),
// including org defaults and each subject's effective access.
func (s *Server) ListFeatureAccess(ctx context.Context, req *adminv1.ListFeatureAccessRequest) (*adminv1.ListFeatureAccessResponse, error) {
	observability.AddRequestAttributes(ctx, attribute.String("args.org", req.Org))

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.Org)
	if err != nil {
		return nil, err
	}
	claims := auth.GetClaims(ctx)
	if !claims.OrganizationPermissions(ctx, org.ID).ManageOrgMembers {
		return nil, status.Error(codes.PermissionDenied, "not allowed to view feature access")
	}

	projectID, err := s.resolveProjectID(ctx, org.Name, req.Project)
	if err != nil {
		return nil, err
	}

	defaults, err := s.admin.DB.GetOrgFeatureDefaults(ctx, org.ID)
	if err != nil {
		return nil, err
	}
	orgDefaults := make([]*adminv1.FeatureAccessEntry, 0, len(featureKeys))
	for _, k := range featureKeys {
		g := true
		if v, ok := defaults[k]; ok {
			g = v
		}
		orgDefaults = append(orgDefaults, &adminv1.FeatureAccessEntry{FeatureKey: k, Granted: g})
	}

	resp := &adminv1.ListFeatureAccessResponse{OrgDefaults: orgDefaults}
	pid := deref(projectID)

	groups, err := s.admin.DB.FindOrganizationMemberUsergroups(ctx, org.ID, "", false, "", 1000)
	if err != nil {
		return nil, err
	}
	for _, g := range groups {
		eff, err := s.admin.DB.ResolveSubjectFeatureAccess(ctx, org.ID, pid, "group", g.ID, featureKeys)
		if err != nil {
			return nil, err
		}
		resp.Subjects = append(resp.Subjects, &adminv1.FeatureAccessSubject{
			SubjectType: "group",
			SubjectId:   g.ID,
			SubjectName: g.Name,
			Features:    eff,
		})
	}

	users, err := s.admin.DB.FindOrganizationMemberUsers(ctx, org.ID, "", false, "", 1000, "")
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		eff, err := s.admin.DB.ResolveSubjectFeatureAccess(ctx, org.ID, pid, "user", u.ID, featureKeys)
		if err != nil {
			return nil, err
		}
		name := u.DisplayName
		if name == "" {
			name = u.Email
		}
		resp.Subjects = append(resp.Subjects, &adminv1.FeatureAccessSubject{
			SubjectType: "user",
			SubjectId:   u.ID,
			SubjectName: name,
			Features:    eff,
		})
	}

	return resp, nil
}

// SetOrgFeatureDefaults sets the org-level default feature grants.
func (s *Server) SetOrgFeatureDefaults(ctx context.Context, req *adminv1.SetOrgFeatureDefaultsRequest) (*adminv1.SetOrgFeatureDefaultsResponse, error) {
	observability.AddRequestAttributes(ctx, attribute.String("args.org", req.Org))

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.Org)
	if err != nil {
		return nil, err
	}
	claims := auth.GetClaims(ctx)
	if !claims.OrganizationPermissions(ctx, org.ID).ManageOrgMembers {
		return nil, status.Error(codes.PermissionDenied, "not allowed to manage feature access")
	}

	changed := make(map[string]any, len(req.Features))
	for _, f := range req.Features {
		if !validFeatureKey(f.FeatureKey) {
			return nil, status.Errorf(codes.InvalidArgument, "unknown feature_key %q", f.FeatureKey)
		}
		if err := s.admin.DB.SetOrgFeatureDefault(ctx, org.ID, f.FeatureKey, f.Granted); err != nil {
			return nil, err
		}
		changed[f.FeatureKey] = f.Granted
	}

	s.admin.RecordAudit(ctx, &admin.AuditEventOptions{
		OrgID:       org.ID,
		ActorUserID: auditActor(claims),
		EventType:   admin.AuditEventOrgFeatureDefaults,
		Payload:     map[string]any{"features": changed},
	})

	return &adminv1.SetOrgFeatureDefaultsResponse{}, nil
}
