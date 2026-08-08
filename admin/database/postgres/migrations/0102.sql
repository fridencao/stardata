-- Per-project semantic layer mode (StarData Phase 5).
--
-- Phase 5 migration strategy is deliberately non-destructive: projects created
-- before the upgrade keep the archive-based model and are frozen read-only,
-- while new projects opt into the DB-versioned semantic layer. That means both
-- code paths must coexist during the transition, selected by this column rather
-- than by a global flag.
--
-- Defaulting to 'archive' means existing rows keep working untouched.
CREATE TYPE semantic_layer_mode AS ENUM ('archive', 'db');

ALTER TABLE projects
    ADD COLUMN semantic_layer_mode semantic_layer_mode NOT NULL DEFAULT 'archive';
