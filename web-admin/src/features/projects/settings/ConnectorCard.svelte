<script lang="ts">
  import Button from "@rilldata/web-common/components/button/Button.svelte";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
  import type { V1AnalyzedConnector } from "@rilldata/web-common/runtime-client";

  let {
    connector,
    onTest,
    testing,
    testResult,
  }: {
    connector: V1AnalyzedConnector;
    onTest: () => void;
    testing: boolean;
    testResult: { ok: boolean; message: string } | null;
  } = $props();

  let name = $derived(connector.name?.trim() || "(unnamed)");
  let displayName = $derived(
    connector.driver?.displayName || connector.driver?.name || "",
  );
  let description = $derived(connector.driver?.description || "");

  let statusLabel = $derived(
    connector.errorMessage
      ? m.connector_status_error()
      : m.connector_status_connected(),
  );

  let statusClass = $derived(
    connector.errorMessage
      ? "bg-red-100 text-red-800"
      : "bg-green-100 text-green-800",
  );
</script>

<div
  class="flex items-start justify-between gap-4 rounded-lg border border-gray-200 bg-white p-4"
>
  <div class="flex flex-col gap-1.5 min-w-0 flex-1">
    <div class="flex items-center gap-2">
      <span
        class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium {statusClass}"
      >
        {statusLabel}
      </span>
      <span class="text-sm font-semibold text-gray-900 truncate">{name}</span>
    </div>
    <div class="flex flex-col gap-0.5">
      <span class="text-xs text-gray-500">{displayName}</span>
      {#if description}
        <span class="text-xs text-gray-400 truncate">{description}</span>
      {/if}
    </div>
    {#if connector.errorMessage}
      <p class="text-xs text-red-600 truncate" title={connector.errorMessage}>
        {connector.errorMessage}
      </p>
    {/if}
    {#if testResult}
      <p
        class="text-xs {testResult.ok ? 'text-green-600' : 'text-red-600'}"
      >
        {testResult.message}
      </p>
    {/if}
  </div>
  <div class="shrink-0">
    <Button
      type="secondary"
      disabled={testing}
      onClick={onTest}
    >
      {#if testing}
        <span class="flex items-center gap-1">
          <span class="inline-block h-3 w-3 animate-spin rounded-full border-2 border-gray-400 border-t-transparent"></span>
          {m.connector_test_connection()}
        </span>
      {:else}
        {m.connector_test_connection()}
      {/if}
    </Button>
  </div>
</div>
