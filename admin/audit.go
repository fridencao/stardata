package admin

import (
	"context"

	"github.com/fridencao/stardata/admin/database"
	"go.uber.org/zap"
)

// Audit event types (StarData). Keep these stable — the compliance view and any
// external audit tooling query on them.
const (
	AuditEventProjectPublish         = "project_publish"
	AuditEventProjectRollback        = "project_rollback"
	AuditEventFeatureAccessSet       = "feature_access_set"
	AuditEventOrgFeatureDefaults     = "org_feature_defaults_set"
	AuditEventOrgAIConfigSet         = "org_ai_config_set"
	AuditEventEditLockForceRelease   = "edit_lock_force_release"
	AuditEventSemanticResourceSave   = "semantic_resource_save"
	AuditEventSemanticResourceDelete = "semantic_resource_delete"
	AuditEventMemberAdd              = "member_add"
	AuditEventMemberRemove           = "member_remove"
	AuditEventMemberRoleChange       = "member_role_change"
	AuditEventUsergroupMemberAdd     = "usergroup_member_add"
	AuditEventUsergroupMemberDrop    = "usergroup_member_remove"
)

// AuditEventOptions describes an audit event to record.
type AuditEventOptions struct {
	OrgID       string
	ProjectID   *string
	ActorUserID *string
	EventType   string
	TargetID    string
	Payload     map[string]any
}

// RecordAudit appends an audit event.
//
// Auditing is observational: a failure to write must never fail the business
// operation that triggered it, so errors are logged rather than returned. Callers
// invoke this after the mutation has succeeded.
func (s *Service) RecordAudit(ctx context.Context, opts *AuditEventOptions) {
	if opts == nil || opts.OrgID == "" || opts.EventType == "" {
		return
	}

	err := s.DB.InsertAuditEvent(ctx, &database.InsertAuditEventOptions{
		OrgID:       opts.OrgID,
		ProjectID:   opts.ProjectID,
		ActorUserID: opts.ActorUserID,
		EventType:   opts.EventType,
		TargetID:    opts.TargetID,
		Payload:     opts.Payload,
	})
	if err != nil {
		s.Logger.Error("failed to record audit event",
			zap.String("event_type", opts.EventType),
			zap.String("org_id", opts.OrgID),
			zap.Error(err),
		)
	}
}
