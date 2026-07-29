<script lang="ts">
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
  import {
    type AddDataConfig,
    type ImportAddDataStep,
    ImportDataStep,
  } from "@rilldata/web-common/features/add-data/manager/steps/types.ts";
  import { useRuntimeClient } from "@rilldata/web-common/runtime-client/v2";
  import { onMount } from "svelte";
  import { Button } from "@rilldata/web-common/components/button";
  import {
    WandIcon,
    CheckCircle2Icon,
    AlertCircleIcon,
    Loader2Icon,
  } from "lucide-svelte";
  import {
    createCanvasDashboardFromTableWithAgent,
    useCreateMetricsViewWithCanvasAndExploreUIAction,
  } from "@rilldata/web-common/features/metrics-views/ai-generation/generateMetricsView.ts";
  import { MetricsEventSpace } from "@rilldata/web-common/metrics/service/MetricsTypes.ts";
  import { BehaviourEventMedium } from "@rilldata/web-common/metrics/service/BehaviourEventTypes.ts";
  import { featureFlags } from "@rilldata/web-common/features/feature-flags.ts";
  import { addLeadingSlash } from "@rilldata/web-common/features/entity-management/entity-mappers.ts";
  import {
    getFileHref,
    withEditorPrefix,
  } from "@rilldata/web-common/layout/navigation/editor-routing";
  import { previewModeStore } from "@rilldata/web-common/layout/preview-mode-store";
  import { runImportSteps } from "@rilldata/web-common/features/add-data/manager/steps/import.ts";
  import type { AddDataStateManager } from "@rilldata/web-common/features/add-data/manager/AddDataStateManager.svelte.ts";

  export let config: AddDataConfig;
  export let stateManager: AddDataStateManager;
  export let importAddDataStep: ImportAddDataStep;
  export let onDone: () => void;

  const { ai, developerChat } = featureFlags;

  const runtimeClient = useRuntimeClient();

  let importStep = ImportDataStep.Init;
  $: currentFileRoute = $previewModeStore
    ? withEditorPrefix("/dashboards")
    : withEditorPrefix("/");
  $: sourceName = importAddDataStep.config.importTo.modelName ?? "";
  $: isDone = importStep === ImportDataStep.Done;
  let error: string | null = null;

  $: createDashboardFromTable =
    useCreateMetricsViewWithCanvasAndExploreUIAction(
      runtimeClient,
      importAddDataStep.config.connector,
      "",
      "",
      sourceName,
      BehaviourEventMedium.Button,
      MetricsEventSpace.Modal,
    );

  async function runImport() {
    try {
      await runImportSteps(
        runtimeClient,
        config,
        importAddDataStep,
        (step, currentFilePath) => {
          importStep = step;
          if (currentFilePath) {
            if ($previewModeStore) {
              currentFileRoute = withEditorPrefix("/dashboards");
            } else {
              currentFileRoute = getFileHref(addLeadingSlash(currentFilePath));
            }
          }
        },
      );
    } catch (e) {
      error = e?.response?.data?.message ?? e?.message ?? "Unknown error";
      stateManager.fireErrorEvent(error!, importStep);
    }
  }

  async function generateMetrics() {
    onDone();
    if ($developerChat && $ai) {
      await createCanvasDashboardFromTableWithAgent(
        runtimeClient,
        importAddDataStep.config.connector,
        "",
        "",
        sourceName,
      );
    } else {
      await createDashboardFromTable();
    }
  }

  onMount(runImport);
</script>

<div class="flex flex-col gap-4 p-6 mx-auto w-full">
  {#if error}
    <div class="header">
      <AlertCircleIcon class="w-5 h-5 text-red-500" />
      {m.add_data_import_failed()}
    </div>
    <div class="content text-destructive">
      {error}
    </div>
    <div class="footer">
      <Button type="secondary" href={currentFileRoute} onClick={onDone}>
        {m.add_data_view_yaml()}
      </Button>
    </div>
  {:else if isDone}
    <div class="header">
      <CheckCircle2Icon class="w-5 h-5 text-green-500" />
      {m.add_data_import_success()}
    </div>
    <div class="content">
      <span class="font-mono text-fg-primary break-all">{sourceName}</span>
      {m.add_data_ingested_next()}
    </div>
    <div class="footer">
      <Button onClick={generateMetrics} type="primary">
        {m.add_data_generate_dashboard()}

        {#if $ai}
          {m.add_data_with_ai()}
          <WandIcon class="w-3 h-3" />
        {/if}
      </Button>

      <Button type="secondary" href={currentFileRoute} onClick={onDone}>
        {m.add_data_view_source()}
      </Button>
    </div>
  {:else}
    <div class="header">
      <Loader2Icon class="w-5 h-5 text-primary-500 animate-spin" />
      {m.add_data_ingesting()}
    </div>
    <div class="content">
      <p class="font-medium">
        {m.add_data_safe_to_close()}
      </p>
      <p class="mt-2 text-sm text-muted-foreground">
        {m.add_data_ingesting_detail()}
      </p>
    </div>
    <div class="footer">
      <Button type="secondary" href={currentFileRoute} onClick={onDone}>
        {m.add_data_view_source()}
      </Button>

      <Button onClick={onDone} type="primary">{m.common_close()}</Button>
    </div>
  {/if}
</div>

<style lang="postcss">
  .header {
    @apply flex items-center gap-2;
    @apply text-lg text-fg-primary font-semibold;
  }

  .content {
    @apply text-sm text-fg-muted;
  }

  .footer {
    @apply flex flex-row-reverse gap-2;
  }
</style>
