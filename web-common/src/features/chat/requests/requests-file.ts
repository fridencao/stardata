import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient";
import {
  getRuntimeServiceGetFileQueryKey,
  runtimeServiceGetFile,
  runtimeServicePutFile,
} from "@rilldata/web-common/runtime-client";
import type { RuntimeClient } from "@rilldata/web-common/runtime-client/v2";
import { parseDocument } from "yaml";

export const REQUESTS_PATH = "/requests.yaml";

export interface RequestItem {
  question: string;
  note?: string;
  created_at: string;
  status: "open" | "done";
}

/**
 * Parse /requests.yaml blob. Missing/empty/corrupt file returns [] (page shows empty state, no white-screen).
 */
export function parseRequestsYaml(blob: string | undefined): RequestItem[] {
  if (!blob) return [];
  try {
    const doc = parseDocument(blob);
    if (doc.errors.length > 0) return [];
    const raw = doc.toJS()?.requests;
    if (!Array.isArray(raw)) return [];
    return raw
      .filter((it) => it && typeof it.question === "string" && it.question)
      .map((it) => ({
        question: it.question as string,
        note: typeof it.note === "string" && it.note ? it.note : undefined,
        created_at: typeof it.created_at === "string" ? it.created_at : "",
        status: it.status === "done" ? ("done" as const) : ("open" as const),
      }));
  } catch {
    return [];
  }
}

async function readRequests(client: RuntimeClient): Promise<RequestItem[]> {
  try {
    const file = await runtimeServiceGetFile(client, { path: REQUESTS_PATH });
    return parseRequestsYaml(file.blob);
  } catch {
    return []; // file does not exist yet
  }
}

/** Write back the full item list (machine-managed file, comments not preserved) and invalidate GetFile cache */
export async function writeRequests(
  client: RuntimeClient,
  items: RequestItem[],
): Promise<void> {
  const doc = parseDocument("requests:\n");
  doc.set(
    "requests",
    doc.createNode(
      items.map((it) => ({
        question: it.question,
        ...(it.note ? { note: it.note } : {}),
        created_at: it.created_at,
        status: it.status,
      })),
    ),
  );
  await runtimeServicePutFile(client, {
    path: REQUESTS_PATH,
    blob: doc.toString(),
    create: true,
    createOnly: false,
  });
  await queryClient.invalidateQueries({
    queryKey: getRuntimeServiceGetFileQueryKey(client.instanceId, {
      path: REQUESTS_PATH,
    }),
  });
}

/** Read latest and append one open request */
export async function appendRequest(
  client: RuntimeClient,
  question: string,
  note?: string,
): Promise<void> {
  const items = await readRequests(client);
  items.push({
    question,
    note: note?.trim() ? note.trim() : undefined,
    created_at: new Date().toISOString(),
    status: "open",
  });
  await writeRequests(client, items);
}
