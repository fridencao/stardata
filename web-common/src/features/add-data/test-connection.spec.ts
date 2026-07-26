import { describe, expect, it } from "vitest";
import type { MultiStepFormSchema } from "@rilldata/web-common/features/templates/schemas/types";
import { buildTestConnectionConfig } from "./test-connection";

const schema: MultiStepFormSchema = {
  type: "object",
  properties: {
    host: { type: "string", title: "Host" },
    port: { type: "string", title: "Port" },
    password: { type: "string", title: "Password", "x-secret": true },
    path: { type: "string", title: "Path", "x-step": "source" },
    connector_type: {
      type: "string",
      title: "Connection type",
      enum: ["parameters", "dsn"],
      "x-display": "tabs",
      "x-tab-group": {
        parameters: ["host", "port", "password"],
        dsn: ["dsn"],
      },
    },
    dsn: { type: "string", title: "DSN" },
    ui_only_field: { type: "string", title: "UI only", "x-ui-only": true },
    headers: { type: "object", title: "Headers", "x-display": "key-value" },
    write_mode: {
      type: "boolean",
      title: "Enable write mode",
      "x-yaml-value": "readwrite",
    },
    read_mode: {
      type: "boolean",
      title: "Read only",
      "x-yaml-value": { true: "readonly", false: "readwrite" },
    },
  },
  required: ["host"],
};

describe("buildTestConnectionConfig", () => {
  it("keeps connector-step values and drops empty strings", () => {
    const config = buildTestConnectionConfig(schema, {
      connector_type: "parameters",
      host: "localhost",
      port: "9000",
      password: "",
    });
    expect(config).toEqual({
      connector_type: "parameters",
      host: "localhost",
      port: "9000",
    });
  });

  it("drops source-step and ui-only fields", () => {
    const config = buildTestConnectionConfig(schema, {
      connector_type: "parameters",
      host: "localhost",
      path: "data/file.csv",
      ui_only_field: "x",
    });
    expect(config).not.toHaveProperty("path");
    expect(config).not.toHaveProperty("ui_only_field");
  });

  it("drops fields from inactive tab groups", () => {
    const config = buildTestConnectionConfig(schema, {
      connector_type: "dsn",
      dsn: "clickhouse://localhost:9000",
      host: "stale-value",
    });
    expect(config).toEqual({
      connector_type: "dsn",
      dsn: "clickhouse://localhost:9000",
    });
  });

  it("converts key-value entries into a plain map and drops empty ones", () => {
    const config = buildTestConnectionConfig(schema, {
      connector_type: "parameters",
      host: "localhost",
      headers: [
        { key: "Authorization", value: "Bearer abc" },
        { key: "  ", value: "ignored" },
      ],
    });
    expect(config.headers).toEqual({ Authorization: "Bearer abc" });

    const emptyConfig = buildTestConnectionConfig(schema, {
      connector_type: "parameters",
      host: "localhost",
      headers: [],
    });
    expect(emptyConfig).not.toHaveProperty("headers");
  });

  it("maps x-yaml-value booleans like the YAML generator", () => {
    // Scalar rule: emitted only when checked, dropped when unchecked
    const checked = buildTestConnectionConfig(schema, {
      connector_type: "parameters",
      host: "localhost",
      write_mode: true,
    });
    expect(checked.write_mode).toBe("readwrite");

    const unchecked = buildTestConnectionConfig(schema, {
      connector_type: "parameters",
      host: "localhost",
      write_mode: false,
    });
    expect(unchecked).not.toHaveProperty("write_mode");

    // Object rule: both toggle states map to their configured values
    const readonly = buildTestConnectionConfig(schema, {
      connector_type: "parameters",
      host: "localhost",
      read_mode: true,
    });
    expect(readonly.read_mode).toBe("readonly");

    const readwrite = buildTestConnectionConfig(schema, {
      connector_type: "parameters",
      host: "localhost",
      read_mode: false,
    });
    expect(readwrite.read_mode).toBe("readwrite");
  });

  it("passes through raw values when no schema is available", () => {
    const config = buildTestConnectionConfig(null, {
      host: "localhost",
      empty: "",
      count: 3,
    });
    expect(config).toEqual({ host: "localhost", count: 3 });
  });
});
