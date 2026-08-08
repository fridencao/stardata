-- Version-to-resource junction table + projects.current_published_version_id
-- (StarData Phase 5.2).
--
-- When a governor publishes, we snapshot the current draft resources by inserting
-- one row here for each resource row that was the "latest draft" at that moment.
-- ON DELETE RESTRICT prevents cleaning up resource rows that are still referenced
-- by any published version, which is what makes rollback safe (you always have the
-- actual definition rows to revert to).
CREATE TABLE project_version_resources (
    project_version_id  UUID NOT NULL REFERENCES project_versions (id) ON DELETE CASCADE,
    semantic_resource_id UUID NOT NULL REFERENCES semantic_resources (id) ON DELETE RESTRICT,
    PRIMARY KEY (project_version_id, semantic_resource_id)
);

CREATE INDEX project_version_resources_by_version
    ON project_version_resources (project_version_id);

CREATE INDEX project_version_resources_by_resource
    ON project_version_resources (semantic_resource_id);

-- Pointer from the project to the version the runtime is currently serving.
-- NULL means no version has ever been published (the project is invisible to
-- business users per the Q10 design decision).
ALTER TABLE projects
    ADD COLUMN current_published_version_id UUID
        REFERENCES project_versions (id) ON DELETE SET NULL;
