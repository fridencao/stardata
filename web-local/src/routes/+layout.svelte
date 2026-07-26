<script lang="ts">
  import { dev } from "$app/environment";
  import { page } from "$app/stores";
  import BannerCenter from "@rilldata/web-common/components/banner/BannerCenter.svelte";
  import NotificationCenter from "@rilldata/web-common/components/notifications/NotificationCenter.svelte";
  import RepresentingUserBanner from "@rilldata/web-common/features/authentication/RepresentingUserBanner.svelte";
  import FileAndResourceWatcher from "@rilldata/web-common/features/entity-management/FileAndResourceWatcher.svelte";
  import { featureFlags } from "@rilldata/web-common/features/feature-flags";
  import { initPylonWidget } from "@rilldata/web-common/features/help/initPylonWidget";
  import RemoteProjectManager from "@rilldata/web-common/features/project/RemoteProjectManager.svelte";
  import ApplicationHeader from "@rilldata/web-common/layout/ApplicationHeader.svelte";
  import BlockingOverlayContainer from "@rilldata/web-common/layout/BlockingOverlayContainer.svelte";
  import { overlay } from "@rilldata/web-common/layout/overlay-store";
  import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient";
  import {
    errorEventHandler,
    initMetrics,
  } from "@rilldata/web-common/metrics/initMetrics";
  import { previewModeStore } from "@rilldata/web-common/layout/preview-mode-store";
  import { LOCAL_HOST, LOCAL_INSTANCE_ID } from "../lib/runtime-client";
  import { getStardataToken } from "@rilldata/web-common/runtime-client/auth-token";
  import RuntimeProvider from "@rilldata/web-common/runtime-client/v2/RuntimeProvider.svelte";
  import type { Query } from "@tanstack/query-core";
  import { QueryClientProvider } from "@tanstack/svelte-query";
  import { onMount } from "svelte";
  import * as Tooltip from "@rilldata/web-common/components/tooltip-v2";
  import type { LayoutData } from "./$types";
  import { isPortalRoute, isStudioRoute } from "./route-constants";
  import "@rilldata/web-common/app.css";

  export let data: LayoutData;

  const { deploy } = featureFlags;

  queryClient.getQueryCache().config.onError = (error: unknown, query: Query) =>
    errorEventHandler?.requestErrorEventHandler(error, query);
  initPylonWidget();

  // Preview mode store sync:
  // 1. Backend lock: if --preview flag is set, always true
  // 2. URL-derived: portal routes (/, /chat, /boards) → true,
  //    studio routes (/studio, /files, ...) → false
  // 3. Preserved: shared routes (/explore, /canvas, /deploy) keep previous value
  $: {
    if (data.previewMode) {
      previewModeStore.set(true);
    } else if (isPortalRoute($page.url.pathname)) {
      previewModeStore.set(true);
    } else if (isStudioRoute($page.url.pathname)) {
      previewModeStore.set(false);
    }
  }

  let removeJavascriptListeners: () => void;
  onMount(async () => {
    const config = data.metadata;

    const shouldSendAnalytics =
      config.analyticsEnabled && !import.meta.env.VITE_PLAYWRIGHT_TEST && !dev;

    if (shouldSendAnalytics) {
      await initMetrics(config, host); // Proxies events through the StarData "intake" service

      removeJavascriptListeners =
        errorEventHandler.addJavascriptErrorListeners();
    }

    featureFlags.set(false, "adminServer");
    featureFlags.set(config.readonly || data.previewMode, "readOnly");
  });

  /**
   * Async mount doesnt support an unsubscribe method.
   * So we need this to make sure javascript listeners for error handler is removed.
   */
  onMount(() => {
    return () => removeJavascriptListeners?.();
  });

  const host = LOCAL_HOST;
  const instanceId = LOCAL_INSTANCE_ID;

  // Self-hosted auth: pass any stored StarData JWT to the runtime client
  // so it can attach `Authorization: Bearer <token>` on API calls.
  const authToken = getStardataToken() ?? undefined;

  $: ({ route } = $page);
  $: isPreviewMode = $previewModeStore;

  $: mode = isPreviewMode ? "Preview" : "Developer";

  $: onWelcomePage = route.id?.startsWith("/(misc)/welcome");

  // The login page must render without RuntimeProvider/FileAndResourceWatcher:
  // unauthenticated watcher requests would 403 and replace the page with the
  // "Error connecting to runtime" screen.
  $: onLoginPage = route.id?.startsWith("/(misc)/login");
</script>

<Tooltip.Provider>
  {#if onLoginPage}
    <slot />
  {:else}
    <QueryClientProvider client={queryClient}>
      <RuntimeProvider {host} {instanceId} jwt={authToken}>
        <FileAndResourceWatcher lifecycle="aggressive">
          <div
            class="body h-screen w-screen overflow-hidden absolute flex flex-col"
          >
            {#if data.initialized && !onWelcomePage}
              <BannerCenter />
              <RepresentingUserBanner />
              <ApplicationHeader {mode} />
              {#if $deploy}
                <RemoteProjectManager />
              {/if}
            {/if}

            <slot />
          </div>
        </FileAndResourceWatcher>
      </RuntimeProvider>
    </QueryClientProvider>
  {/if}

  {#if $overlay !== null}
    <BlockingOverlayContainer
      bg="linear-gradient(to right, rgba(0,0,0,.6), rgba(0,0,0,.8))"
    >
      <div slot="title" class="font-bold">
        {$overlay?.title}
      </div>
      <svelte:fragment slot="detail">
        {#if $overlay?.detail}
          <svelte:component
            this={$overlay.detail.component}
            {...$overlay.detail.props}
          />
        {/if}
      </svelte:fragment>
    </BlockingOverlayContainer>
  {/if}

  <NotificationCenter />
</Tooltip.Provider>

<style>
  /* Prevent trackpad navigation (like other code editors, like vscode.dev). */
  :global(body) {
    overscroll-behavior: none;
  }
</style>
