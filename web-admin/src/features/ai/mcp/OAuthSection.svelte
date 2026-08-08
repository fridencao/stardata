<script lang="ts">
  import CLICommandDisplay from "@rilldata/web-common/components/commands/CLICommandDisplay.svelte";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  export let apiUrl: string;
</script>

<div class="flex flex-col gap-y-4">
  <p class="text-sm text-fg-secondary">
    {m.mcp_add_url()}
  </p>
  <CLICommandDisplay command={apiUrl} />
  <p class="text-xs text-fg-secondary">
    {m.mcp_oauth_auto()} {m.mcp_learn_more()}
  </p>
</div>
