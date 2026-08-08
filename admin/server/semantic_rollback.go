package server

import (
	"context"

	"github.com/fridencao/stardata/admin/database"
	"github.com/fridencao/stardata/admin/server/auth"
	adminv1 "github.com/fridencao/stardata/proto/gen/stardata/admin/v1"
	"github.com/fridencao/stardata/runtime/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Rollback endpoints. All rollback actions require the caller to be a User
// (not a deployment or service) — a service token can publish, but retracting a
// week's work of definitions is a decision people make.

func (s *Server) rollbackRequestToPB(ctx context.Context, r *database.RollbackRequest) *adminv1.RollbackRequestInfo {
	if r == nil {
		return nil
	}
	out := &adminv1.RollbackRequestInfo{
		Id:                r.ID,
		TargetVersion:     int32(r.TargetVersion),
		RequestedByUserId: r.RequestedByUserID,
		Status:            string(r.Status),
		Reason:            r.Reason,
		RequestedOn:       timestamppb.New(r.RequestedOn),
	}
	if r.ResolvedOn != nil {
		out.ResolvedOn = timestamppb.New(*r.ResolvedOn)
	}
	if u, err := s.admin.DB.FindUser(ctx, r.RequestedByUserID); err == nil {
		out.RequestedByUserEmail = u.Email
	}
	if r.ApprovedByUserID != nil {
		out.ApprovedByUserId = *r.ApprovedByUserID
		if u, err := s.admin.DB.FindUser(ctx, *r.ApprovedByUserID); err == nil {
			out.ApprovedByUserEmail = u.Email
		}
	}
	return out
}

func (s *Server) requireHumanEditor(ctx context.Context, orgName, projectName string) (*database.Project, string, error) {
	proj, err := s.requireProjectEditor(ctx, orgName, projectName)
	if err != nil {
		return nil, "", err
	}
	claims := auth.GetClaims(ctx)
	if claims.OwnerType() != auth.OwnerTypeUser {
		return nil, "", status.Error(codes.PermissionDenied, "rollback requires a user identity")
	}
	return proj, claims.OwnerID(), nil
}

func (s *Server) RequestSemanticRollback(ctx context.Context, req *adminv1.RequestSemanticRollbackRequest) (*adminv1.RequestSemanticRollbackResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
		attribute.Int("args.target_version", int(req.TargetVersion)),
	)

	proj, userID, err := s.requireHumanEditor(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	r, err := s.admin.RequestRollback(ctx, proj.ID, int(req.TargetVersion), userID, req.Reason)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	return &adminv1.RequestSemanticRollbackResponse{Request: s.rollbackRequestToPB(ctx, r)}, nil
}

func (s *Server) ApproveSemanticRollback(ctx context.Context, req *adminv1.ApproveSemanticRollbackRequest) (*adminv1.ApproveSemanticRollbackResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
		attribute.String("args.request_id", req.RequestId),
	)

	_, userID, err := s.requireHumanEditor(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	r, err := s.admin.ApproveAndExecuteRollback(ctx, req.RequestId, userID)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	return &adminv1.ApproveSemanticRollbackResponse{Request: s.rollbackRequestToPB(ctx, r)}, nil
}

func (s *Server) RejectSemanticRollback(ctx context.Context, req *adminv1.RejectSemanticRollbackRequest) (*adminv1.RejectSemanticRollbackResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
		attribute.String("args.request_id", req.RequestId),
	)

	_, userID, err := s.requireHumanEditor(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	r, err := s.admin.RejectRollback(ctx, req.RequestId, userID)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	return &adminv1.RejectSemanticRollbackResponse{Request: s.rollbackRequestToPB(ctx, r)}, nil
}

func (s *Server) ListSemanticRollbackRequests(ctx context.Context, req *adminv1.ListSemanticRollbackRequestsRequest) (*adminv1.ListSemanticRollbackRequestsResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
	)

	proj, err := s.requireProjectEditor(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	rows, err := s.admin.DB.ListRollbackRequests(ctx, proj.ID, int(req.Limit))
	if err != nil {
		return nil, err
	}

	out := make([]*adminv1.RollbackRequestInfo, 0, len(rows))
	for _, r := range rows {
		out = append(out, s.rollbackRequestToPB(ctx, r))
	}
	return &adminv1.ListSemanticRollbackRequestsResponse{Requests: out}, nil
}
