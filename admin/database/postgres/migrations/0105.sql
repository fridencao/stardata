-- Per-resource business visibility (StarData Phase 5.2).
--
-- Replaces the publish.yaml file that used to list which metrics views were
-- released. Moving it into a table gives per-resource control (Q13-B): a version
-- is published atomically, but a governor still chooses which of its resources
-- business users can see. That matters in finance, where a new metric often needs
-- to be validated internally for a while before it is released.
--
-- Fail-closed by design: absence of a row, or visible = false, both mean "not
-- visible". A governor must opt a resource in explicitly, so an internal
-- intermediate table cannot leak to business users just because it got published.
CREATE TABLE resource_visibility (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id         UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    resource_kind      semantic_resource_kind NOT NULL,
    resource_name      TEXT NOT NULL,
    visible            BOOLEAN NOT NULL DEFAULT false,
    updated_by_user_id UUID REFERENCES users (id) ON DELETE SET NULL,
    updated_on         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX resource_visibility_unique
    ON resource_visibility (project_id, resource_kind, lower(resource_name));

-- The runtime resolves visibility for a whole project on each catalog load.
CREATE INDEX resource_visibility_by_project
    ON resource_visibility (project_id);
