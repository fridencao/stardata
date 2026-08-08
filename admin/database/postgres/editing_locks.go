package postgres

import (
	"context"
	"time"

	"github.com/fridencao/stardata/admin/database"
)

// The editing lock replaces what named branches used to provide: isolation between
// two governors editing the same project. Because there is now a single draft per
// project, contention is real and must be arbitrated explicitly.
//
// Every mutation below is a single statement. Acquire in particular relies on a
// conditional upsert rather than a read-then-write, so two governors racing for a
// free lock cannot both win.

// FindEditingLock returns the current lock, or ErrNotFound when the project is free.
// Expired locks are treated as absent so a stale row never blocks a new holder even
// if the sweeper has not run yet.
func (c *connection) FindEditingLock(ctx context.Context, projectID string) (*database.EditingLock, error) {
	res := &database.EditingLock{}
	err := c.getDB(ctx).QueryRowxContext(ctx, `
		SELECT project_id, locked_by_user_id, locked_at, last_heartbeat, expires_at
		FROM editing_locks
		WHERE project_id = $1 AND expires_at > now()
	`, projectID).StructScan(res)
	if err != nil {
		return nil, parseErr("editing lock", err)
	}
	return res, nil
}

// AcquireEditingLock takes the lock for userID, or refreshes it if userID already
// holds it. It succeeds when the project is unlocked, when the existing lock has
// expired, or when the caller is the current holder (re-entrant). Otherwise it
// returns the *existing* lock unchanged, so the caller can tell the user who holds it.
//
// The WHERE clause on the DO UPDATE branch is what makes this safe: postgres
// evaluates it against the conflicting row, so a live lock held by someone else
// leaves the row untouched and no row is returned.
func (c *connection) AcquireEditingLock(ctx context.Context, projectID, userID string, ttl time.Duration) (*database.EditingLock, error) {
	expires := time.Now().Add(ttl)

	res := &database.EditingLock{}
	err := c.getDB(ctx).QueryRowxContext(ctx, `
		INSERT INTO editing_locks (project_id, locked_by_user_id, locked_at, last_heartbeat, expires_at)
		VALUES ($1, $2, now(), now(), $3)
		ON CONFLICT (project_id) DO UPDATE SET
			locked_by_user_id = excluded.locked_by_user_id,
			locked_at = CASE
				WHEN editing_locks.locked_by_user_id = excluded.locked_by_user_id THEN editing_locks.locked_at
				ELSE now()
			END,
			last_heartbeat = now(),
			expires_at = excluded.expires_at
		WHERE editing_locks.expires_at <= now()
		   OR editing_locks.locked_by_user_id = excluded.locked_by_user_id
		RETURNING project_id, locked_by_user_id, locked_at, last_heartbeat, expires_at
	`, projectID, userID, expires).StructScan(res)
	if err == nil {
		return res, nil
	}

	// No row returned means the conditional update was skipped: someone else holds a
	// live lock. Surface that holder rather than a bare error so the UI can name them.
	existing, findErr := c.FindEditingLock(ctx, projectID)
	if findErr == nil {
		return existing, database.ErrNotUnique
	}
	return nil, parseErr("editing lock", err)
}

// HeartbeatEditingLock extends the lock's expiry. It only succeeds for the current
// holder and only while the lock is still live: once expired, the holder must
// re-acquire, because another governor may have taken over in the meantime.
func (c *connection) HeartbeatEditingLock(ctx context.Context, projectID, userID string, ttl time.Duration) (*database.EditingLock, error) {
	expires := time.Now().Add(ttl)

	res := &database.EditingLock{}
	err := c.getDB(ctx).QueryRowxContext(ctx, `
		UPDATE editing_locks
		SET last_heartbeat = now(), expires_at = $3
		WHERE project_id = $1 AND locked_by_user_id = $2 AND expires_at > now()
		RETURNING project_id, locked_by_user_id, locked_at, last_heartbeat, expires_at
	`, projectID, userID, expires).StructScan(res)
	if err != nil {
		return nil, parseErr("editing lock", err)
	}
	return res, nil
}

// ReleaseEditingLock drops the lock, but only for its holder. A no-op release
// (already expired or taken over) is not an error: the caller's intent, giving up
// the lock, is satisfied either way.
func (c *connection) ReleaseEditingLock(ctx context.Context, projectID, userID string) error {
	_, err := c.getDB(ctx).ExecContext(ctx, `
		DELETE FROM editing_locks WHERE project_id = $1 AND locked_by_user_id = $2
	`, projectID, userID)
	return parseErr("editing lock", err)
}

// ForceReleaseEditingLock drops the lock regardless of holder. Reserved for org
// admins recovering a project whose holder has gone away without releasing.
func (c *connection) ForceReleaseEditingLock(ctx context.Context, projectID string) error {
	_, err := c.getDB(ctx).ExecContext(ctx, `DELETE FROM editing_locks WHERE project_id = $1`, projectID)
	return parseErr("editing lock", err)
}

// DeleteExpiredEditingLocks removes locks whose heartbeat stopped long enough ago
// that the TTL lapsed. Reads already ignore expired rows, so this is housekeeping
// rather than correctness.
func (c *connection) DeleteExpiredEditingLocks(ctx context.Context) (int, error) {
	res, err := c.getDB(ctx).ExecContext(ctx, `DELETE FROM editing_locks WHERE expires_at <= now()`)
	if err != nil {
		return 0, parseErr("editing lock", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, parseErr("editing lock", err)
	}
	return int(n), nil
}
