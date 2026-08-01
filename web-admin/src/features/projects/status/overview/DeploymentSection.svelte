<script lang="ts">
  import { page } from "$app/stores";
  import {
    createAdminServiceGetProject,
    V1DeploymentStatus,
  } from "@rilldata/web-admin/client";
  import { extractBranchFromPath } from "@rilldata/web-admin/features/branches/branch-utils";
  import { useDashboardsLastUpdated } from "@rilldata/web-admin/features/dashboards/listing/selectors";

  import { createRuntimeServiceGetInstance } from "@rilldata/web-common/runtime-client";
  import { createQueryServiceProjectStorage } from "@rilldata/web-common/runtime-client/v2/gen/query-service";
  import { useRuntimeClient } from "@rilldata/web-common/runtime-client/v2";
  import { formatMemorySize } from "@rilldata/web-common/lib/number-formatting/memory-size";
  import {
    useParserReconcileError,
    useProjectDeployment,
    useRuntimeVersion,
  } from "../selectors";
  import {
    formatEnvironmentName,
    formatConnectorName,
    getOlapEngineLabel,
  } from "@rilldata/web-common/features/resources/display-utils";
  import {
    getStatusDotClass,
    getStatusLabel,
    isTransitoryStatus,
  } from "../display-utils";
  import LoadingCircleOutline from "@rilldata/web-common/components/icons/LoadingCircleOutline.svelte";
  import Callout from "@rilldata/web-common/components/callout/Callout.svelte";
  import ProjectClone from "./ProjectClone.svelte";
  import OverviewCard from "@rilldata/web-common/features/projects/status/overview/OverviewCard.svelte";
  import ClusterSize from "./ClusterSize.svelte";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  export let organization: string;
  export let project: string;

  const runtimeClient = useRuntimeClient();

  $: activeBranch = extractBranchFromPath($page.url.pathname);

  // Deployment
  $: projectDeployment = useProjectDeployment(
    organization,
    project,
    activeBranch,
  );
  $: deployment = $projectDeployment.data;
  $: deploymentStatus =
    deployment?.status ?? V1DeploymentStatus.DEPLOYMENT_STATUS_UNSPECIFIED;

  // ProjectParser — detects project-level failures (e.g. git branch not found)
  $: parserErrorQuery = useParserReconcileError(runtimeClient);
  $: parserReconcileError = $parserErrorQuery.data ?? "";

  // Project
  $: proj = createAdminServiceGetProject(organization, project);
  $: projectData = $proj.data?.project;
  // Last synced
  $: dashboardsLastUpdated = useDashboardsLastUpdated(
    runtimeClient,
    organization,
    project,
  );
  $: lastUpdated = $dashboardsLastUpdated;

  // Runtime
  $: runtimeVersionQuery = useRuntimeVersion(runtimeClient);
  $: version = $runtimeVersionQuery.data?.version?.match(/v[\d.]+/)?.[0] ?? "";

  // Connectors — sensitive: true is needed to read projectConnectors (OLAP/AI connector types)
  $: instanceQuery = createRuntimeServiceGetInstance(runtimeClient, {
    sensitive: true,
  });
  $: instance = $instanceQuery.data?.instance;

  // Project storage (OLAP connector data size)
  $: storageQuery = createQueryServiceProjectStorage(runtimeClient, {});
  $: defaultOlapEntry = $storageQuery.data?.entries?.find(
    (e) => e.isDefaultOlap,
  );
  $: isManaged =
    defaultOlapEntry?.managed || defaultOlapEntry?.connector === "duckdb";
  $: dataSizeBytes = (() => {
    const val = $storageQuery.data?.defaultOlapSizeBytes;
    if (val === undefined || val === null) return undefined;
    const n = Number(val);
    return n >= 0 ? n : undefined;
  })();
  $: dataLabel =
    !defaultOlapEntry || isManaged
      ? m.status_data_size()
      : m.status_data_accessible();

  $: olapConnector = instance?.projectConnectors?.find(
    (c) => c.name === instance?.olapConnector,
  );
  $: olapEngineLabel = getOlapEngineLabel(olapConnector);
  $: aiConnector = instance?.projectConnectors?.find(
    (c) => c.name === instance?.aiConnector,
  );

  // Slots
  $: currentSlots = Number(projectData?.prodSlots) || 0;
</script>

<OverviewCard title={m.status_deployment()}>
  <div slot="header-right" class="flex items-center gap-3">
    <!-- TODO: re-add "Upgrade to Pro" link when ready.
         Gate on: canManage && (isTrialPlan || isFreePlan || isTeamPlan) && !subscriptionQuery.isLoading
    -->
    <ProjectClone
      {organization}
      {project}
      disabled={!!parserReconcileError}
    />
  </div>

  <div class="info-grid">
    <div class="info-row">
      <span class="info-label">{m.status_label_status()}</span>
      <span class="info-value flex items-center gap-2">
        {#if isTransitoryStatus(deploymentStatus)}
          <LoadingCircleOutline size="12px" />
        {:else}
          <span class="status-dot {getStatusDotClass(deploymentStatus)}"></span>
        {/if}
        {getStatusLabel(deploymentStatus)}
      </span>
    </div>

    <div class="info-row">
      <span class="info-label">{m.status_label_environment()}</span>
      <span class="info-value">
        {formatEnvironmentName(deployment?.environment)}
      </span>
    </div>

    {#if currentSlots > 0}
      <div class="info-row">
        <span class="info-label">{m.status_label_cluster_size()}</span>
        <span class="info-value">
          <ClusterSize slots={currentSlots} />
        </span>
      </div>
    {/if}

    {#if parserReconcileError}
      <!-- Project failed to load: show the error instead of project-level details
           (Last synced, OLAP, AI) which would be stale defaults -->
      <div class="mt-2">
        <Callout level="error">
          <span class="text-sm">{parserReconcileError}</span>
        </Callout>
      </div>
    {:else}
      {#if lastUpdated}
        <div class="info-row">
          <span class="info-label">{m.status_label_last_synced()}</span>
          <span class="info-value">
            {lastUpdated.toLocaleString(undefined, {
              year: "numeric",
              month: "short",
              day: "numeric",
              hour: "numeric",
              minute: "numeric",
            })}
          </span>
        </div>
      {/if}

      {#if version}
        <div class="info-row">
          <span class="info-label">{m.status_label_runtime()}</span>
          <span class="info-value">{version}</span>
        </div>
      {/if}

      <div class="info-row">
        <span class="info-label">{m.status_label_olap_engine()}</span>
        <span class="info-value">{olapEngineLabel}</span>
      </div>

      <div class="info-row">
        <span class="info-label">{m.status_label_ai_connector()}</span>
        <span class="info-value">
          {#if aiConnector && aiConnector.name !== "admin"}
            {formatConnectorName(aiConnector.type)}
            <span class="text-fg-tertiary text-xs ml-1"
              >({aiConnector.name})</span
            >
          {:else}
            {m.status_rill_managed()}
          {/if}
        </span>
      </div>

      {#if dataSizeBytes !== undefined}
        <div class="info-row">
          <span class="info-label">{dataLabel}</span>
          <span class="info-value">
            <a
              href="/{organization}/{project}/-/status/tables"
              class="data-size-link"
            >
              {formatMemorySize(dataSizeBytes)}
            </a>
          </span>
        </div>
      {/if}
    {/if}
  </div>
</OverviewCard>

<style lang="postcss">
  .info-grid {
    @apply flex flex-col;
  }
  .info-row {
    @apply flex items-center py-2;
  }
  .info-row:last-child {
    @apply border-b-0;
  }
  .info-label {
    @apply text-sm text-fg-secondary w-32 shrink-0;
  }
  .info-value {
    @apply text-sm text-fg-primary;
  }
  .status-dot {
    @apply w-2 h-2 rounded-full inline-block;
  }
  .data-size-link {
    @apply no-underline;
    color: inherit;
    font: inherit;
  }
</style>
