package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/fridencao/stardata/admin/database"
)

func boolPtr(b bool) *bool { return &b }

// UpsertFeatureAccess inserts or updates a single feature access override.
// projectID is nil for an org-scoped override.
func (c *connection) UpsertFeatureAccess(ctx context.Context, orgID string, projectID *string, subjectType, subjectID, featureKey string, granted bool, createdBy *string) error {
	_, err := c.getDB(ctx).ExecContext(ctx, `
		INSERT INTO feature_access (org_id, project_id, subject_type, subject_id, feature_key, granted, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (org_id, project_id, subject_type, subject_id, feature_key)
		DO UPDATE SET granted = EXCLUDED.granted, created_by_user_id = EXCLUDED.created_by_user_id
	`, orgID, projectID, subjectType, subjectID, featureKey, granted, createdBy)
	if err != nil {
		return parseErr("feature access upsert", err)
	}
	return nil
}

// DeleteFeatureAccess removes a single feature access override.
func (c *connection) DeleteFeatureAccess(ctx context.Context, orgID string, projectID *string, subjectType, subjectID, featureKey string) error {
	_, err := c.getDB(ctx).ExecContext(ctx, `
		DELETE FROM feature_access
		WHERE org_id = $1
		  AND project_id IS NOT DISTINCT FROM $2
		  AND subject_type = $3
		  AND subject_id = $4
		  AND feature_key = $5
	`, orgID, projectID, subjectType, subjectID, featureKey)
	if err != nil {
		return parseErr("feature access delete", err)
	}
	return nil
}

// ListFeatureAccessRows returns feature_access rows for an org.
// When projectID is provided, both project-scoped and org-scoped rows are returned;
// when nil, only org-scoped rows are returned. subjectType/subjectID further narrow the result.
func (c *connection) ListFeatureAccessRows(ctx context.Context, orgID string, projectID *string, subjectType *string, subjectID *string) ([]*database.FeatureAccessRow, error) {
	args := []any{orgID}
	var qry strings.Builder
	qry.WriteString(`SELECT id, org_id, project_id, subject_type, subject_id, feature_key, granted, created_on FROM feature_access WHERE org_id = $1`)
	if projectID != nil && *projectID != "" {
		qry.WriteString(` AND (project_id IS NOT DISTINCT FROM $2 OR project_id IS NULL)`)
		args = append(args, *projectID)
	} else {
		// nil 或空串 "" 都表示「仅组织级」配置
		qry.WriteString(` AND project_id IS NULL`)
	}
	if subjectType != nil {
		args = append(args, *subjectType)
		qry.WriteString(fmt.Sprintf(` AND subject_type = $%d`, len(args)))
	}
	if subjectID != nil {
		args = append(args, *subjectID)
		qry.WriteString(fmt.Sprintf(` AND subject_id = $%d`, len(args)))
	}
	qry.WriteString(` ORDER BY feature_key, subject_type, subject_id`)

	var res []*database.FeatureAccessRow
	if err := c.getDB(ctx).SelectContext(ctx, &res, qry.String(), args...); err != nil {
		return nil, parseErr("feature access list", err)
	}
	return res, nil
}

// GetOrgFeatureDefaults returns the org-level default grants keyed by feature_key.
func (c *connection) GetOrgFeatureDefaults(ctx context.Context, orgID string) (map[string]bool, error) {
	type row struct {
		FeatureKey string `db:"feature_key"`
		Granted    bool   `db:"granted"`
	}
	var rows []row
	if err := c.getDB(ctx).SelectContext(ctx, &rows, `SELECT feature_key, granted FROM org_feature_defaults WHERE org_id = $1`, orgID); err != nil {
		return nil, parseErr("org feature defaults", err)
	}
	m := make(map[string]bool, len(rows))
	for _, r := range rows {
		m[r.FeatureKey] = r.Granted
	}
	return m, nil
}

// SetOrgFeatureDefault upserts an org-level default grant for a feature.
func (c *connection) SetOrgFeatureDefault(ctx context.Context, orgID, featureKey string, granted bool) error {
	_, err := c.getDB(ctx).ExecContext(ctx, `
		INSERT INTO org_feature_defaults (org_id, feature_key, granted) VALUES ($1, $2, $3)
		ON CONFLICT (org_id, feature_key) DO UPDATE SET granted = EXCLUDED.granted, updated_on = now()
	`, orgID, featureKey, granted)
	if err != nil {
		return parseErr("org feature default", err)
	}
	return nil
}

// ResolveSubjectFeatureAccess computes the effective feature access for a subject
// (a user or a user group) in a project.
// Precedence (high -> low): subject(project) > subject(org) > group(project) > group(org) > org default.
// For a user subject, the user's group memberships are also considered.
// A missing org default means the feature is visible by default (granted = true).
func (c *connection) ResolveSubjectFeatureAccess(ctx context.Context, orgID, projectID, subjectType, subjectID string, features []string) (map[string]bool, error) {
	defaults, err := c.GetOrgFeatureDefaults(ctx, orgID)
	if err != nil {
		return nil, err
	}

	var userID, groupIDs []string
	switch subjectType {
	case "user":
		userID = []string{subjectID}
		groups, err := c.FindUsergroupsForUser(ctx, subjectID, orgID)
		if err != nil {
			return nil, err
		}
		groupIDs = make([]string, 0, len(groups))
		for _, g := range groups {
			groupIDs = append(groupIDs, g.ID)
		}
	case "group":
		groupIDs = []string{subjectID}
	default:
		return nil, fmt.Errorf("unknown subject_type %q", subjectType)
	}

	var pid *string
	if projectID != "" {
		pid = &projectID
	}
	rows, err := c.ListFeatureAccessRows(ctx, orgID, pid, nil, nil)
	if err != nil {
		return nil, err
	}

	result := make(map[string]bool, len(features))
	for _, f := range features {
		eff := true
		if v, ok := defaults[f]; ok {
			eff = v
		}
		if groupVal := resolveFromRows(rows, "group", groupIDs, f); groupVal != nil {
			eff = *groupVal
		}
		if userVal := resolveFromRows(rows, "user", userID, f); userVal != nil {
			eff = *userVal
		}
		result[f] = eff
	}
	return result, nil
}

// resolveFromRows returns the effective value for a feature from rows of a given subjectType.
// Project-scoped rows take precedence over org-scoped rows. Across multiple subjects, a deny wins.
func resolveFromRows(rows []*database.FeatureAccessRow, subjectType string, subjectIDs []string, feature string) *bool {
	set := make(map[string]struct{}, len(subjectIDs))
	for _, id := range subjectIDs {
		set[id] = struct{}{}
	}
	hasProj, hasOrg := false, false
	projGranted, projDenied := false, false
	orgGranted, orgDenied := false, false
	for _, r := range rows {
		if r.FeatureKey != feature || r.SubjectType != subjectType {
			continue
		}
		if _, ok := set[r.SubjectID]; !ok {
			continue
		}
		if r.ProjectID != nil {
			hasProj = true
			if r.Granted {
				projGranted = true
			} else {
				projDenied = true
			}
		} else {
			hasOrg = true
			if r.Granted {
				orgGranted = true
			} else {
				orgDenied = true
			}
		}
	}
	if hasProj {
		return boolPtr(projGranted && !projDenied)
	}
	if hasOrg {
		return boolPtr(orgGranted && !orgDenied)
	}
	return nil
}
