-- Dual-approval rollback requests (StarData Phase 5.3).
--
-- Rollback is the single most destructive action a governor can take: it can
-- retract weeks of published metric definitions in one click. For a financial
-- client that is not something one person should be able to do alone, so it is
-- gated on a second governor's approval (Q26).
--
-- Two safety properties are enforced in the schema rather than only in the API:
--   1. A project can have at most one pending request at a time — otherwise two
--      governors could each get a different rollback approved and race.
--   2. The approver cannot be the requester. The API checks this too, but the
--      CHECK constraint means a bug there cannot produce a self-approved rollback.
CREATE TYPE rollback_request_status AS ENUM (
    'pending', 'approved', 'rejected', 'executed'
);

CREATE TABLE rollback_requests (
    id                   UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id           UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    -- The project_versions.version being rolled back to (not the row id, so the
    -- request reads naturally in an audit trail).
    target_version       INTEGER NOT NULL,
    requested_by_user_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    approved_by_user_id  UUID REFERENCES users (id) ON DELETE RESTRICT,
    status               rollback_request_status NOT NULL DEFAULT 'pending',
    reason               TEXT NOT NULL DEFAULT '',
    requested_on         TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_on          TIMESTAMPTZ,
    CONSTRAINT rollback_dual_approval
        CHECK (approved_by_user_id IS NULL OR approved_by_user_id <> requested_by_user_id)
);

-- At most one pending request per project.
CREATE UNIQUE INDEX rollback_requests_single_pending
    ON rollback_requests (project_id)
    WHERE status = 'pending';

CREATE INDEX rollback_requests_project_history
    ON rollback_requests (project_id, requested_on DESC);
