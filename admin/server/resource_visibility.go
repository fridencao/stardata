package server

import (
	"context"

	admin "github.com/fridencao/stardata/admin"
	"github.com/fridencao/stardata/admin/database"
	"github.com/fridencao/stardata/admin/server/auth"
	adminv1 "github.com/fridencao/stardata/proto/gen/stardata/admin/v1"
	"github.com/fridencao/stardata/runtime/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Resource visibility replaces publish.yaml: it decides which resources business
// users can see. Editing a resource does not affect its visibility, and vice
// versa. This split is the point of Q13-B — a governor can save a new version
// atomically and still stage which resources are released.

func resourceVisibilityToPB(r *database.ResourceVisibility) *adminv1.ResourceVisibilityInfo {
	if r == nil {
		return nil
	}
	out := &adminv1.ResourceVisibilityInfo{
		ResourceKind: r.ResourceKind,
		ResourceName: r.ResourceName,
		Visible:      r.Visible,
		UpdatedOn:    timestamppb.New(r.UpdatedOn),
	}
	if r.UpdatedByUserID != nil {
		out.UpdatedByUserId = *r.UpdatedByUserID
	}
	return out
}

func (s *Server) ListResourceVisibility(ctx context.Context, req *adminv1.ListResourceVisibilityRequest) (*adminv1.ListResourceVisibilityResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
	)

	proj, err := s.requireProjectEditor(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	rows, err := s.admin.DB.ListResourceVisibility(ctx, proj.ID)
	if err != nil {
		return nil, err
	}

	out := make([]*adminv1.ResourceVisibilityInfo, 0, len(rows))
	for _, r := range rows {
		out = append(out, resourceVisibilityToPB(r))
	}
	return &adminv1.ListResourceVisibilityResponse{Visibility: out}, nil
}

// SetResourceVisibility flips one resource on or off. Unlike SaveSemanticResource
// this does not require the editing lock: visibility is a release-management
// action, not a definition change, and holding a lock to stage or unstage would
// prevent the person who published from finishing the release themselves.
func (s *Server) SetResourceVisibility(ctx context.Context, req *adminv1.SetResourceVisibilityRequest) (*adminv1.SetResourceVisibilityResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
		attribute.String("args.resource_kind", req.ResourceKind),
		attribute.String("args.resource_name", req.ResourceName),
		attribute.Bool("args.visible", req.Visible),
	)

	proj, err := s.requireProjectEditor(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}
	if !validSemanticResourceKinds[req.ResourceKind] {
		return nil, status.Errorf(codes.InvalidArgument, "unknown resource_kind %q", req.ResourceKind)
	}

	claims := auth.GetClaims(ctx)
	actor := auditActor(claims)

	row, err := s.admin.DB.UpsertResourceVisibility(ctx, &database.UpsertResourceVisibilityOptions{
		ProjectID:       proj.ID,
		ResourceKind:    req.ResourceKind,
		ResourceName:    req.ResourceName,
		Visible:         req.Visible,
		UpdatedByUserID: actor,
	})
	if err != nil {
		return nil, err
	}

	s.admin.RecordAudit(ctx, &admin.AuditEventOptions{
		OrgID:       proj.OrganizationID,
		ProjectID:   &proj.ID,
		ActorUserID: actor,
		EventType:   admin.AuditEventResourceVisibilitySet,
		TargetID:    row.ID,
		Payload: map[string]any{
			"kind":    row.ResourceKind,
			"name":    row.ResourceName,
			"visible": row.Visible,
		},
	})

	return &adminv1.SetResourceVisibilityResponse{Visibility: resourceVisibilityToPB(row)}, nil
}
