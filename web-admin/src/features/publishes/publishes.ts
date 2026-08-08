import { AXIOS_INSTANCE } from "@rilldata/web-admin/client/http-client";
import type {
  PublishBackend,
  PublishEntry,
} from "@rilldata/web-common/features/studio/publish-backend";

/**
 * Publish model client (StarData).
 *
 * The publish endpoints are plain HTTP handlers on the admin server (see
 * admin/server/publishes.go), not gRPC-transcoded, so there are no orval
 * hooks — we call the REST paths directly like the data-requests feature.
 */
export function createPublishBackend(
  organization: string,
  project: string,
): PublishBackend {
  const base = `/v1/orgs/${encodeURIComponent(organization)}/projects/${encodeURIComponent(project)}/publishes`;
  return {
    async list(): Promise<PublishEntry[]> {
      const { data } = await AXIOS_INSTANCE.get<{ publishes: PublishEntry[] }>(
        base,
      );
      return data?.publishes ?? [];
    },
    async publish(note: string): Promise<PublishEntry> {
      const { data } = await AXIOS_INSTANCE.post<PublishEntry>(base, {
        note: note.trim(),
      });
      return data;
    },
    async rollback(version: number): Promise<PublishEntry> {
      const { data } = await AXIOS_INSTANCE.post<PublishEntry>(
        `${base}/${version}/rollback`,
        {},
      );
      return data;
    },
  };
}
