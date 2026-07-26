<script lang="ts">
  import { page } from "$app/state";
  import ConnectorCard from "@rilldata/web-admin/features/projects/settings/ConnectorCard.svelte";
  import Button from "@rilldata/web-common/components/button/Button.svelte";
  import RadixLarge from "@rilldata/web-common/components/typography/RadixLarge.svelte";
  import DelayedSpinner from "@rilldata/web-common/features/entity-management/DelayedSpinner.svelte";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
  import { createRuntimeServiceAnalyzeConnectors } from "@rilldata/web-common/runtime-client";
  import { useRuntimeClient } from "@rilldata/web-common/runtime-client/v2";
  import { testConnection } from "@rilldata/web-common/features/add-data/test-connection";
  import { Plus } from "lucide-svelte";

  let { organization, project } = $derived(page.params);
  const client = useRuntimeClient();

  let connectorsQuery = $derived(
    createRuntimeServiceAnalyzeConnectors(client, {}, {}),
  );

  let connectors = $derived(
    ($connectorsQuery.data?.connectors ?? []).filter(
      (c) => !c.driver?.name?.startsWith("__"),
    ),
  );

  let testingConnector = $state<string | null>(null);
  let testResults = $state<
    Record<string, { ok: boolean; message: string }>
  >({});

  async function handleTestConnection(connectorName: string) {
    testingConnector = connectorName;
    testResults = {
      ...testResults,
      [connectorName]: { ok: false, message: "Testing..." },
    };

    try {
      const connector = $connectorsQuery.data?.connectors?.find(
        (c) => c.name === connectorName,
      );
      if (!connector) return;

      const result = await testConnection(client, connector.driver?.name ?? "", {
        ...connector.config,
        ...connector.envConfig,
      });

      testResults = { ...testResults, [connectorName]: result };
    } catch (e) {
      testResults = {
        ...testResults,
        [connectorName]: {
          ok: false,
          message:
            e instanceof Error ? e.message : "Unknown error during test",
        },
      };
    } finally {
      testingConnector = null;
    }
  }
</script>

<div class="flex flex-col w-full overflow-hidden">
  <div class="flex md:flex-row flex-col gap-6">
    {#if $connectorsQuery.isLoading}
      <DelayedSpinner isLoading={$connectorsQuery.isLoading} size="1rem" />
    {:else if $connectorsQuery.isError}
      <div class="text-red-500">
        {$connectorsQuery.error?.message ??
          "Failed to load connectors"}
      </div>
    {:else if $connectorsQuery.isSuccess}
      <div class="flex flex-col gap-3 w-full overflow-hidden">
        <div class="flex items-center justify-between">
          <div class="flex flex-col">
            <RadixLarge>{m.settings_nav_data_sources()}</RadixLarge>
            <p class="text-sm text-fg-tertiary font-medium">
              Manage your project's data source connections.
            </p>
          </div>
          <a
            href="/{organization}/{project}/-/edit"
            class="no-underline"
          >
            <Button type="primary" large>
              <Plus size="16px" />
              {m.connector_add()}
            </Button>
          </a>
        </div>

        {#if connectors.length > 0}
          <div class="flex flex-col gap-3">
            {#each connectors as connector (connector.name)}
              <ConnectorCard
                {connector}
                testing={testingConnector === connector.name}
                testResult={testResults[connector.name] ?? null}
                onTest={() => handleTestConnection(connector.name!)}
              />
            {/each}
          </div>
        {:else}
          <div class="flex flex-col items-center gap-3 py-12 text-gray-500">
            <p class="text-sm">{m.connector_empty()}</p>
          </div>
        {/if}
      </div>
    {/if}
  </div>
</div>
