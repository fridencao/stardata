-- project_publishes records the publish history of a project (StarData).
-- Each row is a versioned snapshot of the project files, packaged as an archive asset.
-- It powers the publish page (release history) and rollback to a previous version.
CREATE TABLE project_publishes (
    id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    asset_id UUID NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    published_by TEXT NOT NULL DEFAULT '',
    created_on TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, version)
);

CREATE INDEX project_publishes_project_id_created_on_idx ON project_publishes (project_id, created_on DESC);
