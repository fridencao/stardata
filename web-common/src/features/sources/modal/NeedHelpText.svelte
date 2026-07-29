<script lang="ts">
  import type { V1ConnectorDriver } from "@rilldata/web-common/runtime-client";
  import { docsUrl } from "@rilldata/web-common/lib/stardata-links";
  import { ExternalLinkIcon } from "lucide-svelte";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  export let connector: V1ConnectorDriver;
</script>

<div>
  <div class="text-sm leading-none font-medium text-fg-secondary mb-4">
    {m.source_help_title()}
  </div>
  <div class="text-sm leading-normal font-medium text-fg-muted mb-2">
    {m.source_help_text({ name: connector.displayName ?? "" })}
  </div>
  <span class="flex flex-row items-center gap-2 group">
    <a
      href={connector.docsUrl ||
        docsUrl("/developers/build/connectors/")}
      rel="noreferrer noopener"
      target="_blank"
      class="text-sm leading-normal text-primary-500 hover:text-primary-600 font-medium group-hover:underline break-all"
    >
      {m.source_help_how_to({ name: connector.displayName ?? "" })}
    </a>
    <ExternalLinkIcon size="16px" color="#6366F1" />
  </span>
  {#if connector.displayName === "DuckDB" || connector.displayName === "SQLite"}
    <div class="mt-8">
      <div class="text-sm leading-none font-medium text-fg-secondary mb-4">
        {m.source_help_additional()}
      </div>

      <div class="text-sm leading-normal font-medium text-fg-muted mb-2">
        {m.source_help_local_files_warning({ name: connector.displayName ?? "" })}
      </div>
    </div>
  {/if}
</div>
