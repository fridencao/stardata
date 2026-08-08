-- Semantic resources: the DB-versioned replacement for yaml files (StarData Phase 5).
--
-- Until now a project's semantic definitions lived as yaml/sql files inside a
-- tar.gz archive, edited through a dev-deployment draft area synced to disk.
-- That model coupled three unrelated things: version control, draft isolation,
-- and the runtime's file watcher. This table makes the definition itself the
-- versioned unit, so a draft is a row with status='draft' rather than a
-- separate deployment with its own disk.
--
-- Every save inserts a new row rather than updating in place, so the version
-- chain per (project, kind, name) is a complete, append-only audit trail.
CREATE TYPE semantic_resource_kind AS ENUM (
    'source', 'model', 'metrics_view', 'explore', 'canvas',
    'report', 'alert', 'theme', 'api', 'config'
);

CREATE TYPE semantic_resource_status AS ENUM (
    'draft', 'published', 'validating', 'rejected'
);

CREATE TABLE semantic_resources (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id         UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    resource_kind      semantic_resource_kind NOT NULL,
    resource_name      TEXT NOT NULL,
    -- The structured equivalent of the old yaml body. JSONB rather than TEXT so
    -- reference queries ("which metrics views depend on model X") can be served
    -- by a path query instead of parsing every row.
    definition         JSONB NOT NULL,
    -- Monotonic per (project, kind, name). Not globally unique.
    version            INTEGER NOT NULL,
    status             semantic_resource_status NOT NULL DEFAULT 'draft',
    created_by_user_id UUID REFERENCES users (id) ON DELETE SET NULL,
    created_on         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_on         TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Resource names are case-insensitive, matching the runtime's resource naming.
CREATE UNIQUE INDEX semantic_resources_unique_version
    ON semantic_resources (project_id, resource_kind, lower(resource_name), version);

-- The parser's hot path: pull every resource of a given status for a project.
CREATE INDEX semantic_resources_project_status
    ON semantic_resources (project_id, status);

-- Resolving "latest draft of this resource" without a sort over all versions.
CREATE INDEX semantic_resources_latest_draft
    ON semantic_resources (project_id, resource_kind, lower(resource_name), version DESC)
    WHERE status = 'draft';
