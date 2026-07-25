/**
 * Functions that depend on StardataTime types.
 * Separated from new-grains.ts to avoid circular dependency with StardataTime.ts
 */
import {
  StardataLegacyDaxInterval,
  StardataLegacyIsoInterval,
  type StardataTime,
} from "@rilldata/web-common/features/dashboards/url-state/time-ranges/StardataTime";
import { V1TimeGrain } from "@rilldata/web-common/runtime-client";
import {
  GrainAliasToV1TimeGrain,
  getSmallestGrain,
  type TimeGrainAlias,
} from "./new-grains";

export function getRangePrecision(stardataTime: StardataTime) {
  const asOfSnap = stardataTime.asOfLabel?.snap;

  const asOfSnapV1Grain = GrainAliasToV1TimeGrain[asOfSnap as TimeGrainAlias];
  const rangeV1Grain = stardataTime.rangeGrain;
  const intervalV1Grain = stardataTime.interval.getGrain();

  return getSmallestGrain([asOfSnapV1Grain, rangeV1Grain, intervalV1Grain]);
}

export function getAggregationGrain(stardataTime: StardataTime | undefined) {
  if (!stardataTime) return undefined;

  const asOfSnap = stardataTime.asOfLabel?.snap;

  const asOfSnapV1Grain = GrainAliasToV1TimeGrain[asOfSnap as TimeGrainAlias];
  const rangeV1Grain = stardataTime.rangeGrain;
  const intervalV1Grain = stardataTime.interval.getGrain();

  return getSmallestGrain([asOfSnapV1Grain, rangeV1Grain, intervalV1Grain]);
}

export function getTruncationGrain(stardataTime: StardataTime | undefined) {
  if (!stardataTime) return undefined;

  const asOfSnap = stardataTime.asOfLabel?.snap;

  if (asOfSnap) return GrainAliasToV1TimeGrain[asOfSnap as TimeGrainAlias];

  if (stardataTime.interval instanceof StardataLegacyIsoInterval) {
    return stardataTime.interval.getGrain();
  }

  if (stardataTime.interval instanceof StardataLegacyDaxInterval) {
    if (stardataTime.interval.name.endsWith("C")) return undefined;
    return V1TimeGrain.TIME_GRAIN_DAY;
  }

  return undefined;
}
