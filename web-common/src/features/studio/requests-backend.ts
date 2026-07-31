import type { RequestItem } from "@rilldata/web-common/features/chat/requests/requests-file";

/**
 * Pluggable read/write backend for the Studio data-request backlog (StarData).
 *
 * web-local reads/writes /requests.yaml through the runtime. web-admin injects
 * a backend that goes through the admin service instead, because cloud
 * submissions are persisted as a dev-environment virtual file in Postgres
 * (mounted at /__virtual__/requests.yaml in the runtime repo) — reading
 * /requests.yaml through the runtime would never see them.
 * See admin/server/data_requests.go.
 */
export interface RequestsBackend {
  /** List all requests in the backlog. */
  list(): Promise<RequestItem[]>;
  /** Replace the full backlog (used by the "mark done" flow). */
  save(items: RequestItem[]): Promise<void>;
}
