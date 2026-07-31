import { page } from "$app/stores";
import { AXIOS_INSTANCE } from "@rilldata/web-admin/client/http-client";
import type { RequestItem } from "@rilldata/web-common/features/chat/requests/requests-file";
import type { RequestsBackend } from "@rilldata/web-common/features/studio/requests-backend";
import { get } from "svelte/store";

/**
 * Submit a chat data request through the admin service (StarData).
 *
 * The admin server persists it as a dev-environment virtual file (requests.yaml),
 * so submissions work for viewers who have no runtime repo permissions and even
 * when no dev deployment is running.
 */
export async function submitDataRequest(
  question: string,
  note?: string,
): Promise<void> {
  const { organization, project } = get(page).params;
  if (!organization || !project) {
    throw new Error("data requests can only be submitted within a project");
  }
  await AXIOS_INSTANCE.post(
    `/v1/orgs/${encodeURIComponent(organization)}/projects/${encodeURIComponent(project)}/data-requests`,
    {
      question,
      ...(note?.trim() ? { note: note.trim() } : {}),
    },
  );
}

/**
 * Read/write backend for the Studio backlog views (StarData).
 *
 * The backlog lives in the admin virtual file, not in the runtime repo at
 * /requests.yaml, so Studio must list and update it through the admin service
 * (GET/PUT require ManageProject).
 */
export function createRequestsBackend(
  organization: string,
  project: string,
): RequestsBackend {
  const base = `/v1/orgs/${encodeURIComponent(organization)}/projects/${encodeURIComponent(project)}/data-requests`;
  return {
    async list(): Promise<RequestItem[]> {
      const { data } = await AXIOS_INSTANCE.get<{ requests: RequestItem[] }>(
        base,
      );
      return data?.requests ?? [];
    },
    async save(items: RequestItem[]): Promise<void> {
      await AXIOS_INSTANCE.put(base, { requests: items });
    },
  };
}
