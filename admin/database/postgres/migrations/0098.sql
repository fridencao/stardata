-- Audit event log: append-only record of administrative mutations.
-- UI and tooling read this table for compliance audits and operational tracing.
CREATE TABLE admin_audit_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES orgs (id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects (id) ON DELETE SET NULL,
    actor_user_id UUID REFERENCES users (id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    target_id  TEXT NOT NULL DEFAULT '',
    payload    JSONB NOT NULL DEFAULT '{}',
    created_on TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Most reads are "latest events for org" filtered by event_type.
CREATE INDEX idx_audit_events_org_created ON admin_audit_events (org_id, created_on DESC);
CREATE INDEX idx_audit_events_project_created ON admin_audit_events (project_id, created_on DESC) WHERE project_id IS NOT NULL;
