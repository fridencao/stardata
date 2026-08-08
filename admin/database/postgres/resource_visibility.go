package postgres

import (
	"context"

	"github.com/fridencao/stardata/admin/database"
)

// The visibility table is fail-closed: absence of a row and visible = false both
// mean "not visible". Callers should therefore treat the returned map as sparse —
// only rows explicitly opted in appear. This is the security default: it prevents
// a governor from accidentally exposing an internal intermediate metric just by
// having its row exist.

func (c *connection) ListResourceVisibility(ctx context.Context, projectID string) ([]*database.ResourceVisibility, error) {
	var res []*database.ResourceVisibility
	err := c.getDB(ctx).SelectContext(ctx, &res, `
		SELECT id, project_id, resource_kind, resource_name, visible,
		       updated_by_user_id, updated_on
		FROM resource_visibility
		WHERE project_id = $1
		ORDER BY resource_kind, lower(resource_name)
	`, projectID)
	if err != nil {
		return nil, parseErr("resource visibility", err)
	}
	return res, nil
}

// UpsertResourceVisibility flips the visibility of one resource. The unique index
// on (project, kind, lower(name)) is what makes ON CONFLICT usable, and lower()
// matches the case-insensitive naming the runtime uses everywhere else.
func (c *connection) UpsertResourceVisibility(ctx context.Context, opts *database.UpsertResourceVisibilityOptions) (*database.ResourceVisibility, error) {
	res := &database.ResourceVisibility{}
	err := c.getDB(ctx).QueryRowxContext(ctx, `
		INSERT INTO resource_visibility (project_id, resource_kind, resource_name, visible, updated_by_user_id, updated_on)
		VALUES ($1, $2, $3, $4, $5, now())
		ON CONFLICT (project_id, resource_kind, lower(resource_name)) DO UPDATE SET
			visible = excluded.visible,
			updated_by_user_id = excluded.updated_by_user_id,
			updated_on = now()
		RETURNING id, project_id, resource_kind, resource_name, visible,
		          updated_by_user_id, updated_on
	`, opts.ProjectID, opts.ResourceKind, opts.ResourceName, opts.Visible, opts.UpdatedByUserID).StructScan(res)
	if err != nil {
		return nil, parseErr("resource visibility", err)
	}
	return res, nil
}
