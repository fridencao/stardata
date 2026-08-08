package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/fridencao/stardata/admin/database"
)

// semantic_resources is append-only: every save inserts a new version rather than
// updating in place, so the version chain doubles as an audit trail. These helpers
// therefore never UPDATE a definition — only insert, read, and soft-delete.

// FindSemanticResources returns the latest version of every resource in the project
// for the given status. Older versions of the same (kind, name) are filtered out.
func (c *connection) FindSemanticResources(ctx context.Context, projectID string, status database.SemanticResourceStatus) ([]*database.SemanticResource, error) {
	var res []*database.SemanticResource
	// DISTINCT ON collapses the version chain to its newest row per resource, which
	// is what the parser needs: a flat current view, not the history.
	err := c.getDB(ctx).SelectContext(ctx, &res, `
		SELECT DISTINCT ON (resource_kind, lower(resource_name))
		       id, project_id, resource_kind, resource_name, definition, version,
		       status, created_by_user_id, created_on, updated_on
		FROM semantic_resources
		WHERE project_id = $1 AND status = $2
		ORDER BY resource_kind, lower(resource_name), version DESC
	`, projectID, status)
	if err != nil {
		return nil, parseErr("semantic resources", err)
	}
	return res, nil
}

// FindSemanticResource returns the latest version of one resource.
func (c *connection) FindSemanticResource(ctx context.Context, projectID, kind, name string, status database.SemanticResourceStatus) (*database.SemanticResource, error) {
	res := &database.SemanticResource{}
	err := c.getDB(ctx).QueryRowxContext(ctx, `
		SELECT id, project_id, resource_kind, resource_name, definition, version,
		       status, created_by_user_id, created_on, updated_on
		FROM semantic_resources
		WHERE project_id = $1 AND resource_kind = $2 AND lower(resource_name) = lower($3) AND status = $4
		ORDER BY version DESC
		LIMIT 1
	`, projectID, kind, name, status).StructScan(res)
	if err != nil {
		return nil, parseErr("semantic resource", err)
	}
	return res, nil
}

// InsertSemanticResource appends a new draft version of a resource. The version
// number is derived inside the statement so concurrent saves cannot collide on a
// stale read; the unique index on (project, kind, name, version) is the backstop.
func (c *connection) InsertSemanticResource(ctx context.Context, opts *database.InsertSemanticResourceOptions) (*database.SemanticResource, error) {
	res := &database.SemanticResource{}
	err := c.getDB(ctx).QueryRowxContext(ctx, `
		INSERT INTO semantic_resources
			(project_id, resource_kind, resource_name, definition, version, status, created_by_user_id)
		VALUES (
			$1, $2, $3, $4,
			COALESCE((
				SELECT MAX(version) FROM semantic_resources
				WHERE project_id = $1 AND resource_kind = $2 AND lower(resource_name) = lower($3)
			), 0) + 1,
			'draft', $5
		)
		RETURNING id, project_id, resource_kind, resource_name, definition, version,
		          status, created_by_user_id, created_on, updated_on
	`, opts.ProjectID, opts.ResourceKind, opts.ResourceName, opts.Definition, opts.CreatedByUserID).StructScan(res)
	if err != nil {
		return nil, parseErr("semantic resource", err)
	}
	return res, nil
}

// DeleteSemanticResource removes every version of a resource. Deleting a resource
// is itself a draft-level edit, so the whole chain goes; published versions remain
// reachable through the version snapshot tables (Phase 5.2).
func (c *connection) DeleteSemanticResource(ctx context.Context, projectID, kind, name string) error {
	_, err := c.getDB(ctx).ExecContext(ctx, `
		DELETE FROM semantic_resources
		WHERE project_id = $1 AND resource_kind = $2 AND lower(resource_name) = lower($3)
	`, projectID, kind, name)
	return parseErr("semantic resource", err)
}

// FindSemanticResourceFingerprint returns a cheap content hash for the project's
// resources at the given status. The runtime's repo driver uses it to answer
// RepoStore.Hash without shipping every definition over the wire: (count, newest
// updated_on, max version) changes on any insert or delete, which is all the
// watcher needs to decide whether to re-parse.
func (c *connection) FindSemanticResourceFingerprint(ctx context.Context, projectID string, status database.SemanticResourceStatus) (string, error) {
	var count int
	var maxVersion int
	var newest *time.Time
	err := c.getDB(ctx).QueryRowxContext(ctx, `
		SELECT COUNT(*), COALESCE(MAX(version), 0), MAX(updated_on)
		FROM semantic_resources
		WHERE project_id = $1 AND status = $2
	`, projectID, status).Scan(&count, &maxVersion, &newest)
	if err != nil {
		return "", parseErr("semantic resource fingerprint", err)
	}

	ts := ""
	if newest != nil {
		ts = newest.UTC().Format(time.RFC3339Nano)
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%d|%d|%s", projectID, status, count, maxVersion, ts)))
	return hex.EncodeToString(sum[:]), nil
}
