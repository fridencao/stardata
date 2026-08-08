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

// FindOrgAIConfig returns the org's LLM configuration, with the API key decrypted.
// Returns database.ErrNotFound when the org has no override configured.
func (c *connection) FindOrgAIConfig(ctx context.Context, orgID string) (*database.OrgAIConfig, error) {
	res := &database.OrgAIConfig{}
	err := c.getDB(ctx).QueryRowxContext(ctx, `
		SELECT org_id, driver, base_url, model, api_key, api_key_encryption_key_id,
		       updated_by_user_id, created_on, updated_on
		FROM org_ai_config WHERE org_id = $1
	`, orgID).StructScan(res)
	if err != nil {
		return nil, parseErr("org ai config", err)
	}

	plain, err := c.decrypt(res.APIKey, res.APIKeyEncKeyID)
	if err != nil {
		return nil, fmt.Errorf("org ai config: decrypt api key: %w", err)
	}
	res.APIKey = plain
	return res, nil
}

// UpsertOrgAIConfig writes the org's LLM configuration, encrypting the API key.
func (c *connection) UpsertOrgAIConfig(ctx context.Context, opts *database.UpsertOrgAIConfigOptions) (*database.OrgAIConfig, error) {
	// KeepExistingKey lets the UI save provider/model changes without re-sending
	// the secret. COALESCE-style preservation is done in SQL to stay atomic.
	if opts.KeepExistingKey {
		_, err := c.getDB(ctx).ExecContext(ctx, `
			INSERT INTO org_ai_config (org_id, driver, base_url, model, updated_by_user_id, updated_on)
			VALUES ($1, $2, $3, $4, $5, now())
			ON CONFLICT (org_id) DO UPDATE SET
				driver = excluded.driver,
				base_url = excluded.base_url,
				model = excluded.model,
				updated_by_user_id = excluded.updated_by_user_id,
				updated_on = now()
		`, opts.OrgID, opts.Driver, opts.BaseURL, opts.Model, opts.UpdatedByUserID)
		if err != nil {
			return nil, parseErr("org ai config", err)
		}
		return c.FindOrgAIConfig(ctx, opts.OrgID)
	}

	encrypted, keyID, err := c.encrypt(opts.APIKey)
	if err != nil {
		return nil, fmt.Errorf("org ai config: encrypt api key: %w", err)
	}

	_, err = c.getDB(ctx).ExecContext(ctx, `
		INSERT INTO org_ai_config (org_id, driver, base_url, model, api_key, api_key_encryption_key_id, updated_by_user_id, updated_on)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now())
		ON CONFLICT (org_id) DO UPDATE SET
			driver = excluded.driver,
			base_url = excluded.base_url,
			model = excluded.model,
			api_key = excluded.api_key,
			api_key_encryption_key_id = excluded.api_key_encryption_key_id,
			updated_by_user_id = excluded.updated_by_user_id,
			updated_on = now()
	`, opts.OrgID, opts.Driver, opts.BaseURL, opts.Model, encrypted, keyID, opts.UpdatedByUserID)
	if err != nil {
		return nil, parseErr("org ai config", err)
	}
	return c.FindOrgAIConfig(ctx, opts.OrgID)
}

// DeleteOrgAIConfig removes the org override, reverting the org to the
// deployment-wide env-var AI config.
func (c *connection) DeleteOrgAIConfig(ctx context.Context, orgID string) error {
	_, err := c.getDB(ctx).ExecContext(ctx, "DELETE FROM org_ai_config WHERE org_id = $1", orgID)
	return parseErr("org ai config", err)
}
