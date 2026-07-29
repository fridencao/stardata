import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient";
import {
  createRuntimeServiceGetFile,
  runtimeServicePutFile,
} from "@rilldata/web-common/runtime-client";
import { getRuntimeServiceGetFileQueryKey } from "@rilldata/web-common/runtime-client/v2/gen/runtime-service";
import type { RuntimeClient } from "@rilldata/web-common/runtime-client/v2";
import { parse, stringify } from "yaml";

export const PUBLISH_FILE_PATH = "/publish.yaml";

export interface PublishGate {
  /** false = 不门控(文件不存在/为空/解析失败),全部指标集可见 */
  gated: boolean;
  published: Set<string>;
}

export const UNGATED: PublishGate = { gated: false, published: new Set() };

/** 解析 publish.yaml 内容。与 runtime 侧 parsePublishedList 语义一致。 */
export function parsePublishYaml(blob: string): PublishGate {
  let doc: { published?: unknown } | null = null;
  try {
    doc = parse(blob, { logLevel: "silent" }) as { published?: unknown };
  } catch {
    return UNGATED;
  }
  const list = Array.isArray(doc?.published)
    ? doc.published.filter((n): n is string => typeof n === "string")
    : [];
  if (list.length === 0) return UNGATED;
  return { gated: true, published: new Set(list) };
}

/** publish.yaml 文件查询。文件不存在时 isError,调用方按 UNGATED 处理。 */
export function usePublishFile(client: RuntimeClient) {
  return createRuntimeServiceGetFile(
    client,
    { path: PUBLISH_FILE_PATH },
    { query: { retry: false } },
  );
}

/** 重写 publish.yaml 并失效相关查询。 */
export async function writePublishYaml(client: RuntimeClient, names: string[]) {
  const blob =
    "# StarData 发布门控:仅列出的指标集对业务门户可见;文件不存在或名单为空 = 不门控\n" +
    stringify({ published: [...names].sort() });
  await runtimeServicePutFile(client, {
    path: PUBLISH_FILE_PATH,
    blob,
    create: true,
    createOnly: false,
  });
  await queryClient.invalidateQueries({
    queryKey: getRuntimeServiceGetFileQueryKey(client.instanceId, {
      path: PUBLISH_FILE_PATH,
    }),
  });
}
