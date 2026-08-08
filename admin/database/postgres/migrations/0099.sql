-- Org-scoped LLM configuration (StarData).
--
-- Until now the LLM provider, model, and API key came exclusively from admin
-- process env vars, which meant a config change required a redeploy and every
-- org on a shared control plane was forced onto the same provider. This table
-- lets an org admin configure their own provider from the UI. Rows are optional:
-- when absent the deployment-wide env-var config still applies.
CREATE TABLE org_ai_config (
  org_id UUID PRIMARY KEY REFERENCES orgs (id) ON DELETE CASCADE,
  -- Driver name as understood by runtime/drivers (e.g. "openai", "deepseek").
  driver TEXT NOT NULL,
  base_url TEXT NOT NULL DEFAULT '',
  model TEXT NOT NULL DEFAULT '',
  -- API key stored via the admin encryption keyring, same pattern as
  -- project_variables. Empty key id means the keyring was not configured and
  -- the bytes are plaintext.
  api_key BYTEA NOT NULL DEFAULT ''::BYTEA,
  api_key_encryption_key_id TEXT NOT NULL DEFAULT '',
  updated_by_user_id UUID REFERENCES users (id) ON DELETE SET NULL,
  created_on TIMESTAMPTZ DEFAULT now() NOT NULL,
  updated_on TIMESTAMPTZ DEFAULT now() NOT NULL
);
