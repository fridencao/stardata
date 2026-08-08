package postgres

import (
	"context"
	"time"

	"github.com/fridencao/stardata/admin/database"
)

// Project versions are the unit of atomic publish. Two invariants drive the code
// below:
//
//  1. A version's resource set is fixed at snapshot time. Later draft edits create
//     new semantic_resources rows and do not disturb any existing version — which
//     is what lets an already-published metric keep its old definition while a
//     governor edits the new one.
//  2. A version only becomes 'published' after the dry-run gate passes. Insert
//     therefore opens it as 'validating', and nothing in this file promotes it
//     without an explicit status update.

// InsertProjectVersion opens a new version for the project. The version number is
// derived in-statement so two concurrent publishes cannot both read the same
// max(version); the unique index is the backstop.
func (c *connection) InsertProjectVersion(ctx context.Context, opts *database.InsertProjectVersionOptions) (*database.ProjectVersion, error) {
	res := &database.ProjectVersion{}
	err := c.getDB(ctx).QueryRowxContext(ctx, `
		INSERT INTO project_versions (project_id, version, status, note, published_by_user_id)
		VALUES (
			$1,
			COALESCE((SELECT MAX(version) FROM project_versions WHERE project_id = $1), 0) + 1,
			'validating', $2, $3
		)
		RETURNING id, project_id, version, status, published_by_user_id, published_on,
		          note, validation_report, created_on, updated_on
	`, opts.ProjectID, opts.Note, opts.PublishedByUserID).StructScan(res)
	if err != nil {
		return nil, parseErr("project version", err)
	}
	return res, nil
}

func (c *connection) FindProjectVersion(ctx context.Context, id string) (*database.ProjectVersion, error) {
	res := &database.ProjectVersion{}
	err := c.getDB(ctx).QueryRowxContext(ctx, `
		SELECT id, project_id, version, status, published_by_user_id, published_on,
		       note, validation_report, created_on, updated_on
		FROM project_versions WHERE id = $1
	`, id).StructScan(res)
	if err != nil {
		return nil, parseErr("project version", err)
	}
	return res, nil
}

// FindLatestProjectVersion returns the highest-numbered version with the given
// status. Used to resolve "what is currently published" and "is a publish already
// in flight".
func (c *connection) FindLatestProjectVersion(ctx context.Context, projectID string, status database.ProjectVersionStatus) (*database.ProjectVersion, error) {
	res := &database.ProjectVersion{}
	err := c.getDB(ctx).QueryRowxContext(ctx, `
		SELECT id, project_id, version, status, published_by_user_id, published_on,
		       note, validation_report, created_on, updated_on
		FROM project_versions
		WHERE project_id = $1 AND status = $2
		ORDER BY version DESC
		LIMIT 1
	`, projectID, status).StructScan(res)
	if err != nil {
		return nil, parseErr("project version", err)
	}
	return res, nil
}

// ListProjectVersions returns the publish history, newest first.
func (c *connection) ListProjectVersions(ctx context.Context, projectID string, limit int) ([]*database.ProjectVersion, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var res []*database.ProjectVersion
	err := c.getDB(ctx).SelectContext(ctx, &res, `
		SELECT id, project_id, version, status, published_by_user_id, published_on,
		       note, validation_report, created_on, updated_on
		FROM project_versions
		WHERE project_id = $1
		ORDER BY version DESC
		LIMIT $2
	`, projectID, limit)
	if err != nil {
		return nil, parseErr("project versions", err)
	}
	return res, nil
}

// UpdateProjectVersionStatus moves a version through its lifecycle. published_on is
// stamped only on the transition to 'published', so the column always means "when
// this became live" rather than "when it was last touched".
func (c *connection) UpdateProjectVersionStatus(ctx context.Context, id string, status database.ProjectVersionStatus, report []byte) error {
	var publishedOn *time.Time
	if status == database.ProjectVersionStatusPublished {
		now := time.Now()
		publishedOn = &now
	}

	res, err := c.getDB(ctx).ExecContext(ctx, `
		UPDATE project_versions
		SET status = $2,
		    validation_report = COALESCE($3, validation_report),
		    published_on = COALESCE($4, published_on),
		    updated_on = now()
		WHERE id = $1
	`, id, status, report, publishedOn)
	return checkUpdateRow("project version", res, err)
}

// SnapshotDraftResources freezes the project's current draft resource set into the
// version. It uses the same DISTINCT ON collapse as the parser's read path, so the
// snapshot contains exactly the definitions the runtime would have loaded at that
// moment — not an older version of some resource.
func (c *connection) SnapshotDraftResources(ctx context.Context, projectVersionID, projectID string) (int, error) {
	res, err := c.getDB(ctx).ExecContext(ctx, `
		INSERT INTO project_version_resources (project_version_id, semantic_resource_id)
		SELECT $1, sr.id
		FROM (
			SELECT DISTINCT ON (resource_kind, lower(resource_name)) id
			FROM semantic_resources
			WHERE project_id = $2 AND status = 'draft'
			ORDER BY resource_kind, lower(resource_name), version DESC
		) sr
		ON CONFLICT DO NOTHING
	`, projectVersionID, projectID)
	if err != nil {
		return 0, parseErr("project version resources", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, parseErr("project version resources", err)
	}
	return int(n), nil
}

// SetProjectCurrentPublishedVersion repoints the project at a version. This is the
// single switch that makes a version live for business users, and the same call
// serves rollback (pointing back at an older version).
func (c *connection) SetProjectCurrentPublishedVersion(ctx context.Context, projectID, versionID string) error {
	res, err := c.getDB(ctx).ExecContext(ctx, `
		UPDATE projects SET current_published_version_id = $2, updated_on = now() WHERE id = $1
	`, projectID, versionID)
	return checkUpdateRow("project", res, err)
}
