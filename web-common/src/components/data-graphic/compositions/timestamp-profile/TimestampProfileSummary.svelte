<script lang="ts">
  /** TimestampProfileSummary
   * ------------------------
   * This component provides summary information about the
   * timestamp profile at the top of the detail plot.
   */
  import Tooltip from "@rilldata/web-common/components/tooltip/Tooltip.svelte";
  import TooltipContent from "@rilldata/web-common/components/tooltip/TooltipContent.svelte";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
  import { datesToFormattedTimeRange } from "@rilldata/web-common/lib/formatters";
  import type { V1TimeGrain } from "@rilldata/web-common/runtime-client";
  import { GridCell, LeftRightGrid } from "../../../grid";

  export let start: Date;
  export let end: Date;
  export let estimatedSmallestTimeGrain: V1TimeGrain;
  export let rollupTimeGrain: V1TimeGrain;

  const NicerTimeGrain: Record<string, string> = {
    TIME_GRAIN_MILLISECOND: "milliseconds",
    TIME_GRAIN_SECOND: "seconds",
    TIME_GRAIN_MINUTE: "minutes",
    TIME_GRAIN_HOUR: "hourly",
    TIME_GRAIN_DAY: "daily",
    TIME_GRAIN_WEEK: "weekly",
    TIME_GRAIN_MONTH: "monthly",
    TIME_GRAIN_QUARTER: "quarterly",
    TIME_GRAIN_YEAR: "yearly",
  };

  let displayEstimatedSmallestTimegrain: string;
  $: displayEstimatedSmallestTimegrain =
    NicerTimeGrain?.[estimatedSmallestTimeGrain] || estimatedSmallestTimeGrain;

  $: formattedTimeRange = datesToFormattedTimeRange(start, end);

  $: displayRollupGrain = NicerTimeGrain[rollupTimeGrain];
</script>

<div class="text-fg-muted" style:font-size="11px">
  <LeftRightGrid>
    <GridCell>
      <Tooltip distance={16} location="top">
        <div>
          {#if rollupTimeGrain}
            <span class="font-semibold">{formattedTimeRange}</span>
          {/if}
        </div>
        <TooltipContent slot="tooltip-content">
          <div style:max-width="315px">
            {m.graphic_range_of_column({ range: formattedTimeRange })}
          </div>
        </TooltipContent>
      </Tooltip>
    </GridCell>
    <GridCell side="right">
      <Tooltip distance={16} location="top">
        <div>
          <span class="font-semibold">{displayRollupGrain}</span>
          {m.graphic_row_counts()}
        </div>

        <TooltipContent slot="tooltip-content">
          <div style:max-width="315px">
            {m.graphic_rollup_prefix()}
            <b style:font-weight="600"
              >{displayRollupGrain} {m.graphic_level()}</b
            >{m.graphic_rollup_suffix()}
          </div>
        </TooltipContent>
      </Tooltip>
    </GridCell>
    <GridCell side="right">
      <Tooltip distance={16} location="top">
        <div class="text-right">
          {#if estimatedSmallestTimeGrain}
            {m.graphic_min_interval_at()}
            <span class="font-semibold"
              >{displayEstimatedSmallestTimegrain}</span
            >
            {m.graphic_level()}
          {/if}
        </div>
        <TooltipContent slot="tooltip-content">
          <div style:max-width="315px">
            {m.graphic_smallest_interval_prefix()}
            <i>{displayEstimatedSmallestTimegrain}</i>
            {m.graphic_smallest_interval_suffix()}
          </div>
        </TooltipContent>
      </Tooltip>
    </GridCell>
  </LeftRightGrid>
</div>
