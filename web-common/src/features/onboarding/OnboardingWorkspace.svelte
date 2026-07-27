<script lang="ts">
  import { useRuntimeClient } from "../../runtime-client/v2";
  import GenerateSampleData from "@rilldata/web-common/features/sample-data/GenerateSampleData.svelte";
  import { resourceIconMapping } from "@rilldata/web-common/features/entity-management/resource-icon-mapping.ts";
  import { ResourceKind } from "@rilldata/web-common/features/entity-management/resource-selectors.ts";
  import { createResourceAndNavigate } from "@rilldata/web-common/features/entity-management/add/new-files.ts";
  import {
    BehaviourEventMedium,
  } from "@rilldata/web-common/metrics/service/BehaviourEventTypes.ts";
  import {
    MetricsEventScreenName,
    MetricsEventSpace,
  } from "@rilldata/web-common/metrics/service/MetricsTypes.ts";
  import { LightbulbIcon } from "lucide-svelte";
  import ConnectYourDataWidget from "@rilldata/web-common/features/add-data/ConnectYourDataWidget.svelte";
  import AddDataModal from "@rilldata/web-common/features/add-data/AddDataModal.svelte";

  const runtimeClient = useRuntimeClient();

  let openAddDataDialog = false;
  let selectedAddDataSchema: string | null = null;
</script>

<div class="container">
  <div class="cta-container">
    <ConnectYourDataWidget
      startConnectorSelection={(newSchemaName) => {
        selectedAddDataSchema = newSchemaName;
        openAddDataDialog = true;
      }}
    />

    <div class="my-auto text-gray-400 text-base">or</div>

    <div class="onboarding-cta-container">
      <GenerateSampleData type="home" />
      <button
        on:click={() =>
          createResourceAndNavigate(runtimeClient, ResourceKind.Model)}
        class="onboarding-cta"
      >
        <svelte:component
          this={resourceIconMapping[ResourceKind.Model]}
          size="14px"
        />
        Create blank model
      </button>
      <button
        on:click={() =>
          createResourceAndNavigate(runtimeClient, ResourceKind.MetricsView)}
        class="onboarding-cta"
      >
        <svelte:component
          this={resourceIconMapping[ResourceKind.MetricsView]}
          size="14px"
        />
        Create a metrics view
      </button>
    </div>
  </div>

  <div class="flex flex-row gap-x-8 items-center w-full">
    <div class="h-px grow border"></div>
    <LightbulbIcon class="text-border" size="16px" strokeWidth={3} />
    <div class="h-px grow border"></div>
  </div>

  <div class="flex flex-col mx-auto w-fit gap-y-2 text-xs text-slate-500">
    <div class="font-semibold text-center">Tips</div>
    <ul class="list-decimal">
      <li>Import data – Add or drag files (Parquet, NDJSON, CSV).</li>
      <li>Model sources – Combine and shape data with SQL.</li>
      <li>Define metrics – Create metrics and dimensions.</li>
      <li>Explore insights – Visualize data in interactive dashboards.</li>
    </ul>
  </div>
</div>

<AddDataModal
  config={{
    medium: BehaviourEventMedium.Card,
    space: MetricsEventSpace.Workspace,
    screen: MetricsEventScreenName.Home,
  }}
  bind:open={openAddDataDialog}
  schema={selectedAddDataSchema ?? undefined}
/>

<style lang="postcss">
  .container {
    @apply flex flex-col m-auto px-8 gap-y-6 w-fit;
  }

  .cta-container {
    @apply flex flex-row text-center gap-x-6;
  }

  .onboarding-cta-container {
    @apply grid w-64 min-h-full gap-y-4;
  }

  .onboarding-cta {
    @apply flex flex-row gap-2 items-center justify-center p-2;
    @apply text-sm bg-surface-overlay rounded-md border;
  }
  .onboarding-cta:hover {
    @apply bg-surface-hover;
  }
</style>
