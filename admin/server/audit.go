package server

import (
	"context"
	"encoding/json"

	"github.com/fridencao/stardata/admin/database"
	adminv1 "github.com/fridencao/stardata/proto/gen/stardata/admin/v1"
	"github.com/fridencao/stardata/admin/server/auth"
	"github.com/fridencao/stardata/runtime/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Server) ListAuditEvents(ctx context.Context, req *adminv1.ListAuditEventsRequest) (*adminv1.ListAuditEventsResponse, error) {
	observability.AddRequestAttributes(ctx, attribute.String("args.org", req.Org))

	org, err := s.admin.DB.FindOrganizationByName(ctx, req.Org)
	if err != nil {
		return nil, err
	}
	claims := auth.GetClaims(ctx)
	if !claims.OrganizationPermissions(ctx, org.ID).ManageOrg {
		return nil, status.Error(codes.PermissionDenied, "not allowed to view audit events")
	}

	limit := int(req.Limit)
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var filter *database.AuditEventFilter
	if (req.Project != nil && *req.Project != "") || req.EventType != "" {
		filter = &database.AuditEventFilter{}
		if req.Project != nil && *req.Project != "" {
			proj, err := s.admin.DB.FindProjectByName(ctx, org.Name, *req.Project)
			if err != nil {
				return nil, err
			}
			filter.ProjectID = &proj.ID
		}
		if req.EventType != "" {
			filter.EventType = req.EventType
		}
	}

	rows, err := s.admin.DB.ListAuditEventsForOrg(ctx, org.ID, filter, limit)
	if err != nil {
		return nil, err
	}

	// Build lookup caches for user/project denormalization.
	userIDs := map[string]bool{}
	projectIDs := map[string]bool{}
	for _, r := range rows {
		if r.ActorUserID != nil {
			userIDs[*r.ActorUserID] = true
		}
		if r.ProjectID != nil {
			projectIDs[*r.ProjectID] = true
		}
	}

	userMap := map[string]*database.User{}
	for uid := range userIDs {
		u, err := s.admin.DB.FindUser(ctx, uid)
		if err == nil {
			userMap[uid] = u
		}
	}
	projMap := map[string]*database.Project{}
	for pid := range projectIDs {
		p, err := s.admin.DB.FindProject(ctx, pid)
		if err == nil {
			projMap[pid] = p
		}
	}

	events := make([]*adminv1.AuditEvent, 0, len(rows))
	for _, r := range rows {
		ev := &adminv1.AuditEvent{
			Id:        r.ID,
			OrgId:     r.OrgID,
			EventType: r.EventType,
			TargetId:  r.TargetID,
			CreatedOn: timestamppb.New(r.CreatedOn),
		}
		if r.ProjectID != nil {
			ev.ProjectId = r.ProjectID
			if p, ok := projMap[*r.ProjectID]; ok {
				ev.ProjectName = p.Name
			}
		}
		if r.ActorUserID != nil {
			ev.ActorUserId = r.ActorUserID
			if u, ok := userMap[*r.ActorUserID]; ok {
				ev.ActorUserEmail = u.Email
				ev.ActorUserName = u.DisplayName
			}
		}
		if len(r.Payload) > 0 && string(r.Payload) != "{}" {
			var m map[string]any
			if json.Unmarshal(r.Payload, &m) == nil {
				if s, err := structpb.NewStruct(m); err == nil {
					ev.Payload = s
				}
			}
		}
		events = append(events, ev)
	}

	return &adminv1.ListAuditEventsResponse{Events: events}, nil
}
