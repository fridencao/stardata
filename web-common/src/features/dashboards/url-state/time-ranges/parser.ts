import type { StardataTime } from "@rilldata/web-common/features/dashboards/url-state/time-ranges/StardataTime";
import grammar from "./stardata-time.js";
import nearley from "nearley";

const compiledGrammar = nearley.Grammar.fromCompiled(grammar);
export function parseStardataTime(stardataTimeRange: string): StardataTime {
  const parser = new nearley.Parser(compiledGrammar);
  parser.feed(stardataTimeRange);
  const st = parser.results[0] as StardataTime;
  if (!st) throw new Error("Unknown error");
  return st;
}

export function isNewStardataTimeFormat(stardataTime: string): boolean {
  try {
    const st = parseStardataTime(stardataTime);
    return !st.isOldFormat;
  } catch {
    return false;
  }
}

export function validateStardataTime(stardataTime: string): Error | undefined {
  try {
    const st = parseStardataTime(stardataTime);
    if (!st) return new Error("Unknown error");
  } catch (err) {
    return err;
  }
  return undefined;
}

/**
 * Convenience method to parse a stardata time and return its label.
 */
export function getStardataTimeLabel(stardataTime: string): string {
  try {
    const st = parseStardataTime(stardataTime);
    return st.getLabel();
  } catch {
    return stardataTime;
  }
}

/**
 * Overrides the ref part of a stardata time range.
 * @param st StardataTime instance to override
 * @param refOverride Ref to override with, should be in the format of `watermark` or `watermark/Y` or `watermark/Y+1Y` etc
 */
export function overrideStardataTimeRef(st: StardataTime, refOverride: string) {
  const overriddenStardataTime = parseStardataTime(`7D as of ${refOverride}`);
  const overriddenPoint = overriddenStardataTime.anchorOverrides[0];
  if (!overriddenPoint) throw new Error("No anchor overrides found");
  st.overrideRef(overriddenPoint);
}
