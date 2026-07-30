-- Feature access control: per-user / per-user-group feature visibility,
-- with org-level defaults and per-project overrides.

-- Organization-level default feature switches (baseline for the matrix).
CREATE TABLE org_feature_defaults (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id     UUID NOT NULL REFERENCES orgs (id) ON DELETE CASCADE,
  feature_key TEXT NOT NULL,
  granted    BOOLEAN NOT NULL DEFAULT true,
  created_on TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_on TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (org_id, feature_key)
);

-- Per-user / per-user-group feature access overrides.
-- project_id IS NULL  => org-scoped override; otherwise project-scoped.
CREATE TABLE feature_access (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id            UUID NOT NULL REFERENCES orgs (id) ON DELETE CASCADE,
  project_id        UUID REFERENCES projects (id) ON DELETE CASCADE,
  subject_type      TEXT NOT NULL CHECK (subject_type IN ('user', 'group')),
  subject_id        UUID NOT NULL,
  feature_key       TEXT NOT NULL,
  granted           BOOLEAN NOT NULL DEFAULT true,
  created_on        TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by_user_id UUID REFERENCES users (id) ON DELETE SET NULL,
  -- NULLS NOT DISTINCT so org-scoped rows (project_id IS NULL) dedupe correctly.
  UNIQUE NULLS NOT DISTINCT (org_id, project_id, subject_type, subject_id, feature_key)
);

CREATE INDEX idx_feature_access_subject
  ON feature_access (org_id, project_id, subject_type, subject_id);
