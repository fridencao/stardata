package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fridencao/stardata/admin/database"
)

// InsertAuditEvent records a new audit event.
// Errors are returned but callers should log and continue (audit must not block business logic).
func (c *connection) InsertAuditEvent(ctx context.Context, opts *database.InsertAuditEventOptions) error {
	payload := []byte("{}")
	if opts.Payload != nil {
		var err error
		payload, err = json.Marshal(opts.Payload)
		if err != nil {
			return fmt.Errorf("audit: marshal payload: %w", err)
		}
	}
	_, err := c.getDB(ctx).ExecContext(ctx, `
		INSERT INTO admin_audit_events (org_id, project_id, actor_user_id, event_type, target_id, payload)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, opts.OrgID, opts.ProjectID, opts.ActorUserID, opts.EventType, opts.TargetID, payload)
	if err != nil {
		return parseErr("audit insert", err)
	}
	return nil
}

// ListAuditEventsForOrg returns recent audit events for the org, newest first.
func (c *connection) ListAuditEventsForOrg(ctx context.Context, orgID string, filter *database.AuditEventFilter, limit int) ([]*database.AuditEvent, error) {
	args := []any{orgID}
	var qry strings.Builder
	qry.WriteString(`SELECT id, org_id, project_id, actor_user_id, event_type, target_id, payload, created_on FROM admin_audit_events WHERE org_id = $1`)
	if filter != nil {
		if filter.ProjectID != nil {
			args = append(args, *filter.ProjectID)
			qry.WriteString(fmt.Sprintf(` AND project_id = $%d`, len(args)))
		}
		if filter.EventType != "" {
			args = append(args, filter.EventType)
			qry.WriteString(fmt.Sprintf(` AND event_type = $%d`, len(args)))
		}
	}
	qry.WriteString(` ORDER BY created_on DESC`)
	if limit > 0 {
		args = append(args, limit)
		qry.WriteString(fmt.Sprintf(` LIMIT $%d`, len(args)))
	}

	var res []*database.AuditEvent
	if err := c.getDB(ctx).SelectContext(ctx, &res, qry.String(), args...); err != nil {
		return nil, parseErr("audit list", err)
	}
	return res, nil
}
