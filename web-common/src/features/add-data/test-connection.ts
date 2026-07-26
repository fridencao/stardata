import {
  filterSchemaValuesForSubmit,
  getConditionalValues,
} from "@rilldata/web-common/features/templates/schema-utils";
import type { MultiStepFormSchema } from "@rilldata/web-common/features/templates/schemas/types";
import type { RuntimeClient } from "@rilldata/web-common/runtime-client/v2";

export type TestConnectionResult = {
  ok: boolean;
  message: string;
};

/**
 * Builds the config payload for the test connection API from raw form values.
 * Mirrors the YAML generator's filtering: applies conditional const/default
 * values, keeps only connector-step non-internal fields, maps x-yaml-value
 * booleans to their emitted values, and drops empty values.
 */
export function buildTestConnectionConfig(
  schema: MultiStepFormSchema | null,
  formValues: Record<string, unknown>,
): Record<string, unknown> {
  const merged = schema
    ? { ...formValues, ...getConditionalValues(schema, formValues) }
    : formValues;
  const filtered = schema
    ? filterSchemaValuesForSubmit(schema, merged, { step: "connector" })
    : merged;

  const config: Record<string, unknown> = {};
  for (const [key, rawValue] of Object.entries(filtered)) {
    if (rawValue === undefined || rawValue === null) continue;
    if (typeof rawValue === "string" && rawValue.trim() === "") continue;
    const value = normalizeValue(
      applyYamlValueRule(schema?.properties?.[key]?.["x-yaml-value"], rawValue),
    );
    if (value === undefined) continue;
    config[key] = value;
  }
  return config;
}

/**
 * Applies a schema's x-yaml-value rule to a boolean form value, matching the
 * YAML generator: the object form maps both toggle states; the scalar form is
 * emitted only when checked (unchecked fields are dropped via undefined).
 */
function applyYamlValueRule(rule: unknown, value: unknown): unknown {
  if (rule === undefined || rule === null || typeof value !== "boolean") {
    return value;
  }
  if (typeof rule === "object") {
    const mapped = (rule as Record<string, unknown>)[value ? "true" : "false"];
    return mapped !== undefined ? mapped : value;
  }
  return value === true ? rule : undefined;
}

/**
 * Converts key-value input entries (Array<{key, value}>) into a plain map,
 * since drivers expect map-shaped config (e.g. `headers`). Empty arrays and
 * arrays without valid keys are dropped.
 */
function normalizeValue(value: unknown): unknown {
  if (Array.isArray(value)) {
    const entries = value.filter(
      (e): e is { key: string; value: string } =>
        !!e &&
        typeof e === "object" &&
        typeof (e as { key?: unknown }).key === "string" &&
        (e as { key: string }).key.trim() !== "",
    );
    if (entries.length === 0) return undefined;
    return Object.fromEntries(entries.map((e) => [e.key.trim(), e.value]));
  }
  return value;
}

/**
 * Calls the raw HTTP endpoint `POST /v1/instances/{id}/connectors:testconnection`
 * to verify a connection with unsaved config. Never throws — network and
 * protocol errors are folded into the result.
 */
export async function testConnection(
  client: RuntimeClient,
  driver: string,
  config: Record<string, unknown>,
): Promise<TestConnectionResult> {
  const url = `${client.host}/v1/instances/${client.instanceId}/connectors:testconnection`;
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  const jwt = client.getJwt();
  if (jwt) headers["Authorization"] = `Bearer ${jwt}`;

  try {
    const resp = await fetch(url, {
      method: "POST",
      headers,
      body: JSON.stringify({ driver, config }),
    });
    const data = (await resp.json()) as {
      ok?: boolean;
      message?: string;
      error?: string;
    };
    return {
      ok: Boolean(data.ok),
      message:
        data.message ??
        data.error ??
        `Unexpected response (HTTP ${resp.status})`,
    };
  } catch (e) {
    return {
      ok: false,
      message: e instanceof Error ? e.message : String(e),
    };
  }
}
