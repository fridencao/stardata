package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/fridencao/stardata/admin/database"
	"go.uber.org/zap"
)

// StarData Phase 5.3: rollback operates on top of Phase 5.2's version snapshots.
// It is deliberately expensive to do — dual approval, 90-day window — because the
// same click that undoes an incident can also silently retract weeks of published
// definitions if triggered by mistake.

// rollbackWindow bounds how far back a rollback target may be. Versions older than
// this fall out of the direct UI and require an archive-recovery process (out of
// scope for Phase 5). 90 days matches the Q26 decision.
const rollbackWindow = 90 * 24 * time.Hour

// RequestRollback opens a rollback request for a project to a specific version.
// It fails if the target version does not exist, is older than the window, is
// already the current version, or if there is already a pending request.
func (s *Service) RequestRollback(ctx context.Context, projectID string, targetVersion int, requestedByUserID, reason string) (*database.RollbackRequest, error) {
	proj, err := s.DB.FindProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if proj.SemanticLayerMode != "db" {
		return nil, fmt.Errorf("rollback: project is not in DB semantic layer mode")
	}

	// Verify the target version exists and is within the rollback window.
	versions, err := s.DB.ListProjectVersions(ctx, projectID, 500)
	if err != nil {
		return nil, err
	}
	var target *database.ProjectVersion
	for _, v := range versions {
		if v.Version == targetVersion {
			target = v
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("rollback: version %d does not exist", targetVersion)
	}
	if target.Status != database.ProjectVersionStatusPublished {
		return nil, fmt.Errorf("rollback: version %d is %s, only published versions can be rolled back to", targetVersion, target.Status)
	}
	if proj.CurrentPublishedVersionID != nil && *proj.CurrentPublishedVersionID == target.ID {
		return nil, fmt.Errorf("rollback: version %d is already the current published version", targetVersion)
	}
	if target.PublishedOn != nil && time.Since(*target.PublishedOn) > rollbackWindow {
		return nil, fmt.Errorf("rollback: version %d is older than the %d-day window", targetVersion, int(rollbackWindow.Hours()/24))
	}

	req, err := s.DB.InsertRollbackRequest(ctx, &database.InsertRollbackRequestOptions{
		ProjectID:         projectID,
		TargetVersion:     targetVersion,
		RequestedByUserID: requestedByUserID,
		Reason:            reason,
	})
	if err != nil {
		return nil, err
	}
	return req, nil
}

// ApproveAndExecuteRollback approves a pending rollback request and executes it.
// Approval and execution are one call because splitting them would let a request
// linger in an 'approved' but not-yet-executed limbo where the runtime state and
// the DB state disagree.
func (s *Service) ApproveAndExecuteRollback(ctx context.Context, requestID, approverUserID string) (*database.RollbackRequest, error) {
	req, err := s.DB.FindRollbackRequest(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if req.Status != database.RollbackRequestStatusPending {
		return nil, fmt.Errorf("rollback: request is %s, not pending", req.Status)
	}
	if req.RequestedByUserID == approverUserID {
		// The DB CHECK enforces this too, but returning a clean error here beats a
		// generic constraint violation surfacing to the UI.
		return nil, fmt.Errorf("rollback: the approver must be different from the requester")
	}

	proj, err := s.DB.FindProject(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}

	// Re-verify the target is still published and within the window. Something
	// could have changed between request and approval.
	versions, err := s.DB.ListProjectVersions(ctx, req.ProjectID, 500)
	if err != nil {
		return nil, err
	}
	var target *database.ProjectVersion
	for _, v := range versions {
		if v.Version == req.TargetVersion {
			target = v
			break
		}
	}
	if target == nil || target.Status != database.ProjectVersionStatusPublished {
		return nil, fmt.Errorf("rollback: target version %d is no longer eligible", req.TargetVersion)
	}
	if target.PublishedOn != nil && time.Since(*target.PublishedOn) > rollbackWindow {
		return nil, fmt.Errorf("rollback: target version is older than the %d-day window", int(rollbackWindow.Hours()/24))
	}

	// Point the project at the target version. This is the single switch that makes
	// the older definitions live again — the same call publish uses.
	if err := s.DB.SetProjectCurrentPublishedVersion(ctx, req.ProjectID, target.ID); err != nil {
		return nil, fmt.Errorf("rollback: switch version: %w", err)
	}

	// Notify the runtime so business users see the older version immediately rather
	// than waiting for the next poll. Best-effort: the DB switch is already live.
	if err := s.notifyRuntimeVersionChange(ctx, proj); err != nil {
		s.Logger.Warn("rollback: failed to notify runtime (will catch up on next poll)",
			zap.String("project_id", req.ProjectID),
			zap.Int("target_version", req.TargetVersion),
			zap.Error(err),
		)
	}

	// Mark the request executed. Doing this last means a mid-flight failure leaves
	// the request pending and easy to retry; a stuck 'approved' would be worse.
	approver := approverUserID
	if err := s.DB.ResolveRollbackRequest(ctx, requestID, database.RollbackRequestStatusExecuted, &approver); err != nil {
		return nil, fmt.Errorf("rollback: finalize request: %w", err)
	}

	s.RecordAudit(ctx, &AuditEventOptions{
		OrgID:       proj.OrganizationID,
		ProjectID:   &req.ProjectID,
		ActorUserID: &approverUserID,
		EventType:   AuditEventProjectRollback,
		TargetID:    target.ID,
		Payload: map[string]any{
			"target_version": req.TargetVersion,
			"requested_by":   req.RequestedByUserID,
			"approved_by":    approverUserID,
			"reason":         req.Reason,
		},
	})

	updated, _ := s.DB.FindRollbackRequest(ctx, requestID)
	return updated, nil
}

// RejectRollback declines a pending request. Also requires the resolver be
// different from the requester — a governor cannot silently walk their own
// request back if a peer is on the way to approving it.
func (s *Service) RejectRollback(ctx context.Context, requestID, rejecterUserID string) (*database.RollbackRequest, error) {
	req, err := s.DB.FindRollbackRequest(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if req.Status != database.RollbackRequestStatusPending {
		return nil, fmt.Errorf("rollback: request is %s, not pending", req.Status)
	}
	if req.RequestedByUserID == rejecterUserID {
		return nil, fmt.Errorf("rollback: the rejecter must be different from the requester")
	}

	rejecter := rejecterUserID
	if err := s.DB.ResolveRollbackRequest(ctx, requestID, database.RollbackRequestStatusRejected, &rejecter); err != nil {
		return nil, err
	}

	updated, _ := s.DB.FindRollbackRequest(ctx, requestID)
	return updated, nil
}
