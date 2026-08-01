<script lang="ts">
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
  import InfoCircle from "@rilldata/web-common/components/icons/InfoCircle.svelte";
  import Shortcut from "@rilldata/web-common/components/tooltip/Shortcut.svelte";
  import Tooltip from "@rilldata/web-common/components/tooltip/Tooltip.svelte";
  import TooltipContent from "@rilldata/web-common/components/tooltip/TooltipContent.svelte";
  import TooltipShortcutContainer from "@rilldata/web-common/components/tooltip/TooltipShortcutContainer.svelte";
  import TooltipTitle from "@rilldata/web-common/components/tooltip/TooltipTitle.svelte";
  import { fly } from "svelte/transition";
  import { createLocalServiceGetMetadata } from "@rilldata/web-common/runtime-client/local-service";
  import { DOCS_BASE_URL } from "@rilldata/web-common/lib/stardata-links";
  import RuntimeTrafficLights from "@rilldata/web-common/features/entity-management/RuntimeTrafficLights.svelte";

  const metadataQuery = createLocalServiceGetMetadata();

  $: version = $metadataQuery?.data?.version ?? null;
  $: commitHash = $metadataQuery?.data?.buildCommit ?? null;
</script>

<div
  class="flex flex-col pt-3 pb-3 gap-y-1 bg-surface-subtle border-t sticky bottom-0"
>
  <div
    class="px-4 py-1 text-fg-secondary flex items-center flex-row w-full gap-x-2 truncate line-clamp-1"
    style:font-size="10px"
  >
    <span>
      <Tooltip alignment="start" distance={16} location="top">
        <a
          href={DOCS_BASE_URL}
          target="_blank"
          rel="noreferrer noopener"
          class="text-fg-secondary"
        >
          <InfoCircle size="16px" />
        </a>
        <div
          slot="tooltip-content"
          transition:fly|global={{ duration: 100, y: 8 }}
        >
          <TooltipContent>
            <TooltipTitle>
              <svelte:fragment slot="name"
                >{m.footer_rill_developer()}</svelte:fragment
              >
            </TooltipTitle>
            <TooltipShortcutContainer>
              <div>{m.footer_view_documentation()}</div>
              <Shortcut>{m.footer_shortcut_click()}</Shortcut>
            </TooltipShortcutContainer>
          </TooltipContent>
        </div>
      </Tooltip>
    </span>

    <span class="truncate">
      {m.footer_version()}
      {version || m.footer_unknown_version()}{commitHash
        ? ` – ${commitHash}`
        : ""}
    </span>
    <RuntimeTrafficLights />
  </div>
</div>
