<script lang="ts">
  import { page } from "$app/stores";
  import ErrorPage from "@rilldata/web-common/components/ErrorPage.svelte";
  import CanvasDashboardEmbed from "@rilldata/web-common/features/canvas/CanvasDashboardEmbed.svelte";
  import CanvasProvider from "@rilldata/web-common/features/canvas/CanvasProvider.svelte";
  import { useCanvas } from "@rilldata/web-common/features/canvas/selector";
  import DashboardChat from "@rilldata/web-common/features/chat/DashboardChat.svelte";
  import { ResourceKind } from "@rilldata/web-common/features/entity-management/resource-selectors.ts";
  import { isNotFoundError } from "@rilldata/web-common/lib/errors";
  import { useRuntimeClient } from "@rilldata/web-common/runtime-client/v2";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  const runtimeClient = useRuntimeClient();

  $: ({
    params: { organization, project },
  } = $page);

  $: canvasName = $page.params.name;
  $: canvasQuery = useCanvas(runtimeClient, canvasName);

  $: isCanvasNotFound =
    !$canvasQuery.data &&
    $canvasQuery.isError &&
    isNotFoundError($canvasQuery.error);
</script>

<svelte:head>
  <title>StarData · {canvasName}</title>
</svelte:head>

{#key `${runtimeClient.instanceId}::${canvasName}`}
  {#if isCanvasNotFound}
    <ErrorPage
      statusCode={404}
      header={m.portal_boards_not_found()}
      href={`/${organization}/${project}/boards`}
    />
  {:else}
    <div class="flex h-full overflow-hidden">
      <div class="flex-1 overflow-hidden">
        <CanvasProvider
          {canvasName}
          instanceId={runtimeClient.instanceId}
          showBanner
        >
          <CanvasDashboardEmbed {canvasName} />
        </CanvasProvider>
      </div>
      <DashboardChat kind={ResourceKind.Canvas} />
    </div>
  {/if}
{/key}
