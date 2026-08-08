<script lang="ts">
  import { isClipboardApiSupported } from "@rilldata/web-common/lib/actions/copy-to-clipboard";
  import { portal } from "@rilldata/web-common/lib/actions/portal";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
  import FormattedDataType from "../data-types/FormattedDataType.svelte";
  import Shortcut from "../tooltip/Shortcut.svelte";
  import StackingWord from "../tooltip/StackingWord.svelte";
  import TooltipContent from "../tooltip/TooltipContent.svelte";
  import TooltipShortcutContainer from "../tooltip/TooltipShortcutContainer.svelte";
  import TooltipTitle from "../tooltip/TooltipTitle.svelte";

  type HoveringData = {
    value: string | number | null;
    index?: number;
    column?: string;
    type?: string;
    isHeader?: boolean;
    isPin?: boolean;
  };

  export let hoverPosition = { top: 0, left: 0, width: 0 };
  export let pinned: boolean;
  export let hovering: HoveringData;
  export let sortable: boolean;
  export let customShortcuts: { description: string; shortcut: string }[] = [];
</script>

<aside
  class="w-fit h-fit absolute -translate-x-1/2 -translate-y-full z-[1000]"
  use:portal
  style:top="{hoverPosition.top - 8}px"
  style:left="{hoverPosition.left + hoverPosition.width / 2}px"
>
  <TooltipContent maxWidth="360px">
    {#if hovering.isPin}
      {pinned ? m.vtable_unpin_column() : m.vtable_pin_column()}
    {:else}
      <TooltipTitle>
        <svelte:fragment slot="name">
          {#if hovering.isHeader}
            {hovering.value}
          {:else}
            <FormattedDataType
              color="text-fg-inverse"
              type={hovering?.type ?? "VARCHAR"}
              value={hovering?.value}
            />
          {/if}
        </svelte:fragment>

        <svelte:fragment slot="description">
          {hovering.isHeader ? hovering.type : ""}
        </svelte:fragment>
      </TooltipTitle>

      {#if !hovering.isPin}
        <TooltipShortcutContainer>
          {#if hovering.isHeader && sortable}
            <div>{m.vtable_sort_column()}</div>
            <Shortcut>{m.vtable_click()}</Shortcut>
          {/if}
          {#if isClipboardApiSupported()}
            <div>
              <StackingWord key="shift">{m.chart_copy_to_clipboard()}</StackingWord>
              {hovering.isHeader ? m.vtable_copy_column_name() : m.vtable_copy_value()}
            </div>
            <Shortcut>
              <span style="font-family: var(--system);">⇧</span>
              {m.bignumber_shift_click()}
            </Shortcut>
          {/if}
          {#if customShortcuts.length > 0}
            {#each customShortcuts as { description, shortcut }}
              <div>{description}</div>
              <Shortcut>{shortcut}</Shortcut>
            {/each}
          {/if}
        </TooltipShortcutContainer>
      {/if}
    {/if}
  </TooltipContent>
</aside>
