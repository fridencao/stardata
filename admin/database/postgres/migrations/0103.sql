-- Project-level version snapshots (StarData Phase 5.2).
--
-- A version is an atomic snapshot of all draft semantic resources at one point in
-- time, sealed by the governor's "publish" action. The lifecycle:
--   draft -> validating (dry-run in progress) -> published | rejected
--
-- Once published, the version's content is immutable: it can be rolled back to,
-- and every query in that window must return exactly those definitions. The
-- ON DELETE RESTRICT on the junction table enforces that.
CREATE TYPE project_version_status AS ENUM (
    'draft', 'published', 'validating', 'rejected'
);

CREATE TABLE project_versions (
    id                   UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id           UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    version              INTEGER NOT NULL,
    status               project_version_status NOT NULL DEFAULT 'validating',
    published_by_user_id UUID REFERENCES users (id) ON DELETE SET NULL,
    published_on         TIMESTAMPTZ,
    note                 TEXT NOT NULL DEFAULT '',
    -- Dry-run outcome stored so the governor can understand why a version was
    -- rejected, and the admin can surface a "last publish attempt failed" banner
    -- without re-running the validation.
    validation_report    JSONB,
    created_on           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_on           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX project_versions_unique
    ON project_versions (project_id, version);

-- "Current published version" lookups happen on every business-user request
-- (the resolver needs to know which resources to serve).
CREATE INDEX project_versions_published
    ON project_versions (project_id, version DESC)
    WHERE status = 'published';
