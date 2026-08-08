<script lang="ts">
  /**
   * The TimestampTooltipContent is used in the TimestampDetail component.
   * The goal is to provide user a quick & easy onboarding for the basic TimestampDetail
   * actions of zooming and panning. This component is a bit extra.
   */
  import Shortcut from "@rilldata/web-common/components/tooltip/Shortcut.svelte";
  import StackingWord from "@rilldata/web-common/components/tooltip/StackingWord.svelte";
  import TooltipContent from "@rilldata/web-common/components/tooltip/TooltipContent.svelte";
  import TooltipShortcutContainer from "@rilldata/web-common/components/tooltip/TooltipShortcutContainer.svelte";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
  import {
    formatBigNumberPercentage,
    formatInteger,
  } from "@rilldata/web-common/lib/formatters";
  import { isClipboardApiSupported } from "../../../../lib/actions/copy-to-clipboard";
  import TimestampSpark from "./TimestampSpark.svelte";
  import type { TimestampDataPoint } from "@rilldata/web-common/features/column-profile/queries";

  export let data: TimestampDataPoint[];
  // FIXME: document meaning of these special looking numbers
  // e.g. something like width = y* CHAR_HEIGHT, height = CHAR_HEIGHT?
  export let width = 84;
  export let height = 12;

  export let totalRows: number;
  export let zoomedRows: number;

  // these flags change the text in the tooltip.
  export let zoomed = false;
  export let zooming = false;
  // this determines the "shake" of the pan label when panning.
  export let tooltipPanShakeAmount = 0;
  // the window bounds for the spark within the zoom row of the tooltip.
  export let zoomWindowXMin: Date | undefined = undefined;
  export let zoomWindowXMax: Date | undefined = undefined;
</script>

<TooltipContent>
  <div class="pt-1 pb-1 italic font-semibold">
    {#if zoomed}
      <div
        class="grid space-between w-full"
        style="grid-template-columns: auto max-content;"
      >
        <div>
          {#if zooming}<span>{m.graphic_zoomed()}</span>{:else}<span
              >{m.graphic_zooming()}</span
            >{/if}
          {m.graphic_to_n_rows({
            count: zoomedRows,
            countStr: formatInteger(zoomedRows),
          })}
        </div>
        <div class="text-right text-gray-300 font-normal not-italic">
          {formatBigNumberPercentage(zoomedRows / totalRows)}
        </div>
      </div>
    {:else}
      {m.graphic_showing_all_rows({ countStr: formatInteger(totalRows) })}
    {/if}
  </div>
  <TooltipShortcutContainer>
    {#if isClipboardApiSupported()}
      <div>
        <StackingWord key="shift">{m.graphic_copy()}</StackingWord>
        {m.graphic_to_clipboard()}
      </div>
      <Shortcut>
        <span
          style="
          font-family: var(--system); 
          font-size: 11.5px;
        ">⇧</span
        >
        <!-- i18n-ignore: keyboard shortcut -->
        + Click
      </Shortcut>
    {/if}
    <div>
      <div style:transform="translateX({tooltipPanShakeAmount}px)">
        {m.graphic_pan()}
      </div>
    </div>
    <!-- i18n-ignore: keyboard shortcut -->
    <Shortcut>Click + Drag</Shortcut>
    <div>
      {m.graphic_zoom()}
      <div style:display="inline-grid">
        <TimestampSpark
          {data}
          {width}
          {height}
          left={0}
          right={0}
          top={0}
          bottom={0}
          color="var(--color-teal-300)"
          {zoomWindowXMin}
          {zoomWindowXMax}
        />
      </div>
    </div>
    <!-- i18n-ignore: keyboard shortcut -->
    <Shortcut>Ctrl + Click + Drag</Shortcut>
  </TooltipShortcutContainer>
</TooltipContent>
