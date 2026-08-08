package server

import (
	"context"
	"errors"
	"time"

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

// StarData Phase 5: with named branches gone there is a single draft per project,
// so two governors editing at once would silently overwrite each other. The lock
// below arbitrates that. It is advisory in the sense that it guards the editing
// experience, but SaveSemanticResource enforces it, so it is not bypassable by
// calling the API directly.
//
// The TTL exists because a governor who closes their laptop must not wedge the
// project forever. It is deliberately long enough (hours, not minutes) that a
// meeting or a lunch break does not cost someone their editing session.
const (
	defaultEditLockTTL = 2 * time.Hour
)

// editLockTTL returns the configured lock TTL. Kept as a function so the value can
// later be sourced from org-level settings without touching call sites.
func (s *Server) editLockTTL() time.Duration {
	return defaultEditLockTTL
}

// requireProjectEditor resolves the project and asserts the caller may edit its
// semantic layer. Editing is a governor capability, hence ManageProject.
func (s *Server) requireProjectEditor(ctx context.Context, orgName, projectName string) (*database.Project, error) {
	proj, err := s.admin.DB.FindProjectByName(ctx, orgName, projectName)
	if err != nil {
		return nil, err
	}
	claims := auth.GetClaims(ctx)
	if !claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID).ManageProject {
		return nil, status.Error(codes.PermissionDenied, "not allowed to edit this project")
	}
	return proj, nil
}

// editLockToPB decorates the lock with the holder's name and email so the UI can
// say who has the project rather than showing a bare UUID.
func (s *Server) editLockToPB(ctx context.Context, lock *database.EditingLock) *adminv1.EditLockInfo {
	if lock == nil {
		return nil
	}
	out := &adminv1.EditLockInfo{
		ProjectId:      lock.ProjectID,
		LockedByUserId: lock.LockedByUserID,
		LockedAt:       timestamppb.New(lock.LockedAt),
		ExpiresAt:      timestamppb.New(lock.ExpiresAt),
	}
	if u, err := s.admin.DB.FindUser(ctx, lock.LockedByUserID); err == nil {
		out.LockedByUserEmail = u.Email
		out.LockedByUserName = u.DisplayName
	}
	return out
}

func (s *Server) AcquireEditLock(ctx context.Context, req *adminv1.AcquireEditLockRequest) (*adminv1.AcquireEditLockResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
	)

	proj, err := s.requireProjectEditor(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	if claims.OwnerType() != auth.OwnerTypeUser {
		return nil, status.Error(codes.PermissionDenied, "only users can hold an editing lock")
	}

	lock, err := s.admin.DB.AcquireEditingLock(ctx, proj.ID, claims.OwnerID(), s.editLockTTL())
	if err != nil {
		// ErrNotUnique means someone else holds a live lock. That is an expected
		// outcome, not a failure: return who holds it so the UI can go read-only.
		if errors.Is(err, database.ErrNotUnique) {
			return &adminv1.AcquireEditLockResponse{
				Lock:     s.editLockToPB(ctx, lock),
				Acquired: false,
			}, nil
		}
		return nil, err
	}

	return &adminv1.AcquireEditLockResponse{
		Lock:     s.editLockToPB(ctx, lock),
		Acquired: true,
	}, nil
}

func (s *Server) HeartbeatEditLock(ctx context.Context, req *adminv1.HeartbeatEditLockRequest) (*adminv1.HeartbeatEditLockResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
	)

	proj, err := s.requireProjectEditor(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	lock, err := s.admin.DB.HeartbeatEditingLock(ctx, proj.ID, claims.OwnerID(), s.editLockTTL())
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			// The lock lapsed or was taken over. Say so explicitly rather than
			// silently succeeding, so the client stops assuming it can still write.
			return nil, status.Error(codes.FailedPrecondition, "editing lock is no longer held; re-acquire it")
		}
		return nil, err
	}

	return &adminv1.HeartbeatEditLockResponse{Lock: s.editLockToPB(ctx, lock)}, nil
}

func (s *Server) ReleaseEditLock(ctx context.Context, req *adminv1.ReleaseEditLockRequest) (*adminv1.ReleaseEditLockResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
	)

	proj, err := s.requireProjectEditor(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	if err := s.admin.DB.ReleaseEditingLock(ctx, proj.ID, claims.OwnerID()); err != nil {
		return nil, err
	}
	return &adminv1.ReleaseEditLockResponse{}, nil
}

func (s *Server) GetEditLock(ctx context.Context, req *adminv1.GetEditLockRequest) (*adminv1.GetEditLockResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
	)

	proj, err := s.requireProjectEditor(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	lock, err := s.admin.DB.FindEditingLock(ctx, proj.ID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			// Unlocked is a normal state, not an error.
			return &adminv1.GetEditLockResponse{}, nil
		}
		return nil, err
	}
	return &adminv1.GetEditLockResponse{Lock: s.editLockToPB(ctx, lock)}, nil
}

// ForceReleaseEditLock is the recovery path for a lock whose holder is unreachable.
// It requires org-level authority rather than project-level, because taking someone
// else's session away is an administrative act, and it is audited.
func (s *Server) ForceReleaseEditLock(ctx context.Context, req *adminv1.ForceReleaseEditLockRequest) (*adminv1.ForceReleaseEditLockResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
	)

	proj, err := s.admin.DB.FindProjectByName(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	if !claims.OrganizationPermissions(ctx, proj.OrganizationID).ManageOrg {
		return nil, status.Error(codes.PermissionDenied, "not allowed to force-release editing locks")
	}

	// Capture the displaced holder before deleting, so the audit trail records whose
	// session was taken.
	var displaced string
	if lock, err := s.admin.DB.FindEditingLock(ctx, proj.ID); err == nil {
		displaced = lock.LockedByUserID
	}

	if err := s.admin.DB.ForceReleaseEditingLock(ctx, proj.ID); err != nil {
		return nil, err
	}

	s.admin.RecordAudit(ctx, &admin.AuditEventOptions{
		OrgID:       proj.OrganizationID,
		ProjectID:   &proj.ID,
		ActorUserID: auditActor(claims),
		EventType:   admin.AuditEventEditLockForceRelease,
		TargetID:    proj.ID,
		Payload:     map[string]any{"displaced_user_id": displaced},
	})

	return &adminv1.ForceReleaseEditLockResponse{}, nil
}
