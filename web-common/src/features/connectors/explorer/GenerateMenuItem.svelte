<script lang="ts">
  import { WandIcon } from "lucide-svelte";
  import NavigationMenuItem from "@rilldata/web-common/layout/navigation/NavigationMenuItem.svelte";
  import MetricsViewIcon from "../../../components/icons/MetricsViewIcon.svelte";
  import ExploreIcon from "@rilldata/web-common/components/icons/ExploreIcon.svelte";
  import { featureFlags } from "../../feature-flags";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  export let onClick: () => void;
  export let type: "metrics" | "dashboard";

  const { ai } = featureFlags;

  $: icon = type === "metrics" ? MetricsViewIcon : ExploreIcon;
  $: label =
    type === "metrics"
      ? m.connector_generate_metrics()
      : m.add_data_generate_dashboard();
</script>

<NavigationMenuItem onclick={onClick}>
  <svelte:component this={icon} slot="icon" />
  <div class="flex gap-x-2 items-center">
    {label}
    {#if $ai}
      {m.add_data_with_ai()}
      <WandIcon class="w-3 h-3" />
    {/if}
  </div>
</NavigationMenuItem>
