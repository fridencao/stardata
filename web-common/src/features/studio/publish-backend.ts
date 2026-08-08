/**
 * Publish model backend contract (StarData).
 *
 * The publish/rollback/history endpoints live on the admin server, which is
 * only reachable from web-admin. StudioPublishPage takes an optional
 * PublishBackend prop: web-admin injects an implementation backed by the
 * admin HTTP API; web-local passes nothing and the release UI is hidden
 * (the publish.yaml gating table works everywhere).
 */

/** One publish history entry, mirroring the admin server's publishItem JSON. */
export interface PublishEntry {
  version: number;
  note?: string;
  published_by?: string;
  created_at: string;
  current: boolean;
}

export interface PublishBackend {
  /** List the publish history, newest first. */
  list(): Promise<PublishEntry[]>;
  /** Package the dev draft and publish it to production. */
  publish(note: string): Promise<PublishEntry>;
  /** Re-deploy a previous release to production. */
  rollback(version: number): Promise<PublishEntry>;
}
