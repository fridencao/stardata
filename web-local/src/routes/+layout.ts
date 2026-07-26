import { WelcomeStatus } from "@rilldata/web-common/features/welcome/status.ts";

export const ssr = false;

import { redirect } from "@sveltejs/kit";
import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient.js";
import {
  getRuntimeServiceListFilesQueryKey,
  runtimeServiceListFiles,
  type V1ListFilesResponse,
} from "@rilldata/web-common/runtime-client/index.js";
import { fileArtifacts } from "@rilldata/web-common/features/entity-management/file-artifacts.js";
import { handleUninitializedProject } from "@rilldata/web-common/features/welcome/is-project-initialized.js";
import { localServiceGetMetadata } from "@rilldata/web-common/runtime-client/local-service";
import {
  getStardataToken,
  clearStardataToken,
} from "@rilldata/web-common/runtime-client/auth-token";
import { ConnectError, Code } from "@connectrpc/connect";
import { eventBus } from "@rilldata/web-common/lib/event-bus/event-bus";
import { getLocalRuntimeClient } from "../lib/runtime-client";
import {
  DEVELOPER_ALLOWED_PREFIXES,
  PREVIEW_ALLOWED_PREFIXES,
} from "./route-constants";
import { Settings } from "luxon";
import { RuntimeFileIO } from "@rilldata/web-common/features/entity-management/file-io.ts";

Settings.defaultLocale = "en";

// Cache metadata: previewMode is static for the server lifetime, so fetch once
let cachedMetadata: Awaited<ReturnType<typeof localServiceGetMetadata>> | null =
  null;

/** Builds the login URL, preserving the originally requested page. */
function loginURL(url: URL): string {
  const target = url.pathname + url.search;
  return target === "/"
    ? "/login"
    : `/login?redirect=${encodeURIComponent(target)}`;
}

export async function load({ url, depends, untrack, route }) {
  depends("app:init");

  // Fetch metadata to check preview mode (cached after first load)
  if (!cachedMetadata) {
    cachedMetadata = await localServiceGetMetadata();
  }
  const metadata = cachedMetadata;
  const previewMode = metadata.previewMode ?? false;

  // Self-hosted auth: the login page must render without any authenticated
  // API calls and must bypass the mode-based route locking below.
  if (untrack(() => url.pathname.startsWith("/login"))) {
    return { initialized: false, previewMode, metadata };
  }

  // Not signed in yet? Send the user to the login page.
  if (!getStardataToken()) {
    throw redirect(
      303,
      untrack(() => loginURL(url)),
    );
  }

  // Enforce mode-based route locking.
  // Wrapped in untrack() so SvelteKit does not register url.pathname as a
  // dependency; without this, the entire load function re-runs on every
  // client-side navigation, causing unnecessary data refetches and UI flicker.
  untrack(() => {
    if (previewMode) {
      // Preview mode: only allow preview-related and shared routes
      const isAllowed = PREVIEW_ALLOWED_PREFIXES.some((prefix) =>
        url.pathname.startsWith(prefix),
      );
      if (!isAllowed) {
        eventBus.emit("notification", {
          message: "This page is only available in Developer mode",
        });
        throw redirect(303, "/dashboards");
      }
    } else {
      // Developer mode: block preview-exclusive routes
      const isAllowed =
        url.pathname === "/" ||
        DEVELOPER_ALLOWED_PREFIXES.some((prefix) =>
          url.pathname.startsWith(prefix),
        );
      if (!isAllowed) {
        eventBus.emit("notification", {
          message: "This page is only available in Preview mode",
        });
        throw redirect(303, "/");
      }
    }
  });

  const client = getLocalRuntimeClient();

  // Set the client on fileArtifacts early so child page load functions
  // (e.g., files/[...file]/+page.ts) can access it before components render.
  fileArtifacts.setClient(client, new RuntimeFileIO());

  const files = await queryClient
    .fetchQuery<V1ListFilesResponse>({
      queryKey: getRuntimeServiceListFilesQueryKey(client.instanceId, {}),
      queryFn: ({ signal }) => {
        return runtimeServiceListFiles(client, {}, { signal });
      },
    })
    .catch((e) => {
      const ce = ConnectError.from(e);
      if (
        ce.code === Code.Unauthenticated ||
        ce.code === Code.PermissionDenied
      ) {
        // Stale or invalid token: clear it and restart the login flow
        clearStardataToken();
        throw redirect(
          303,
          untrack(() => loginURL(url)),
        );
      }
      throw e;
    });

  const firstDashboardFile = files.files?.find((file) =>
    file.path?.startsWith("/dashboards/"),
  );
  const redirectPath = firstDashboardFile
    ? `/files${firstDashboardFile?.path}`
    : "/";

  let initialized = !!files.files?.some(({ path }) => path === "/rill.yaml");

  const trackedRedirectPath = untrack(() => {
    if (!url.searchParams.get("redirect")) return false;

    // In preview mode, redirect to /dashboards instead of /files
    if (previewMode) {
      return url.pathname !== "/dashboards" && "/dashboards";
    }

    return url.pathname !== redirectPath && redirectPath;
  });

  if (!initialized) {
    initialized = await handleUninitializedProject(client);
    if (!initialized && !route?.id?.startsWith("/(misc)/welcome")) {
      WelcomeStatus.set(true);
      throw redirect(303, "/welcome");
    }
  } else if (trackedRedirectPath) {
    throw redirect(303, trackedRedirectPath);
  }

  return { initialized, previewMode, metadata };
}
