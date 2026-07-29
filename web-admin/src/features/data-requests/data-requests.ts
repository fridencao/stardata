import { page } from "$app/stores";
import { AXIOS_INSTANCE } from "@rilldata/web-admin/client/http-client";
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
