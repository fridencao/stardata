package postgres

import (
	"context"
	"time"

	"github.com/fridencao/stardata/admin/database"
)

// Rollback is dual-approved because it is the one action that can silently retract
// weeks of published definitions. The DB carries two of the guardrails so an API
// bug cannot bypass them: a partial unique index limits a project to one pending
// request, and a CHECK constraint forbids self-approval.

func (c *connection) InsertRollbackRequest(ctx context.Context, opts *database.InsertRollbackRequestOptions) (*database.RollbackRequest, error) {
	res := &database.RollbackRequest{}
	err := c.getDB(ctx).QueryRowxContext(ctx, `
		INSERT INTO rollback_requests (project_id, target_version, requested_by_user_id, reason)
		VALUES ($1, $2, $3, $4)
		RETURNING id, project_id, target_version, requested_by_user_id, approved_by_user_id,
		          status, reason, requested_on, resolved_on
	`, opts.ProjectID, opts.TargetVersion, opts.RequestedByUserID, opts.Reason).StructScan(res)
	if err != nil {
		// A unique violation here means another request is already pending, which is
		// a meaningful conflict rather than an internal failure.
		return nil, parseErr("rollback request", err)
	}
	return res, nil
}

func (c *connection) FindRollbackRequest(ctx context.Context, id string) (*database.RollbackRequest, error) {
	res := &database.RollbackRequest{}
	err := c.getDB(ctx).QueryRowxContext(ctx, `
		SELECT id, project_id, target_version, requested_by_user_id, approved_by_user_id,
		       status, reason, requested_on, resolved_on
		FROM rollback_requests WHERE id = $1
	`, id).StructScan(res)
	if err != nil {
		return nil, parseErr("rollback request", err)
	}
	return res, nil
}

// FindPendingRollbackRequest returns the project's open request, or ErrNotFound.
func (c *connection) FindPendingRollbackRequest(ctx context.Context, projectID string) (*database.RollbackRequest, error) {
	res := &database.RollbackRequest{}
	err := c.getDB(ctx).QueryRowxContext(ctx, `
		SELECT id, project_id, target_version, requested_by_user_id, approved_by_user_id,
		       status, reason, requested_on, resolved_on
		FROM rollback_requests
		WHERE project_id = $1 AND status = 'pending'
	`, projectID).StructScan(res)
	if err != nil {
		return nil, parseErr("rollback request", err)
	}
	return res, nil
}

func (c *connection) ListRollbackRequests(ctx context.Context, projectID string, limit int) ([]*database.RollbackRequest, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var res []*database.RollbackRequest
	err := c.getDB(ctx).SelectContext(ctx, &res, `
		SELECT id, project_id, target_version, requested_by_user_id, approved_by_user_id,
		       status, reason, requested_on, resolved_on
		FROM rollback_requests
		WHERE project_id = $1
		ORDER BY requested_on DESC
		LIMIT $2
	`, projectID, limit)
	if err != nil {
		return nil, parseErr("rollback requests", err)
	}
	return res, nil
}

// ResolveRollbackRequest moves a request out of 'pending'. The WHERE clause pins it
// to the pending state so a second approval cannot re-resolve an already-handled
// request (e.g. two approvers clicking at once).
func (c *connection) ResolveRollbackRequest(ctx context.Context, id string, status database.RollbackRequestStatus, approvedByUserID *string) error {
	now := time.Now()
	res, err := c.getDB(ctx).ExecContext(ctx, `
		UPDATE rollback_requests
		SET status = $2, approved_by_user_id = COALESCE($3, approved_by_user_id), resolved_on = $4
		WHERE id = $1 AND status = 'pending'
	`, id, status, approvedByUserID, now)
	return checkUpdateRow("rollback request", res, err)
}
