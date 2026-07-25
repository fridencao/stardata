<!-- ANCHORED_SUMMARY: Rename getRillDefaultExploreState -> getStarDataDefaultExploreState -->
<!-- State: ALL_COMPLETE -->
<!-- Updated: 2026-07-25 -->

## Objective
- Rename `getRillDefaultExploreState` → `getStarDataDefaultExploreState` and update all references

## Important Details
- File renamed: `get-rill-default-explore-state.ts` → `get-stardata-default-explore-state.ts`
- Verified zero stale refs: `grep -r rillDefaultExploreState` = 0 matches, `grep -r getRillDefaultExploreState` = 0 matches
- Separate function `getRillDefaultExploreUrlParams` has its own stale refs but was NOT in scope

## Work State
### Completed
- **File rename**: `get-rill-default-explore-state.ts` → `get-stardata-default-explore-state.ts`
- **Barrel export**: `stores/index.ts` import path updated
- **DashboardStateDataLoader.ts**: All refs fixed — property name, comments, method parameter, `exploreStateOrder` sorted order
- **data.ts**: Import path + function call fixed
- **Post-category scan**: Broader stale patterns identified — all belong to separate function `getRillDefaultExploreUrlParams`, not in scope

### Active
- None — all in-scope tasks complete

### Blocked
- "Category F" (ExploreBookmarks variables/comments) and "Category G" (Chinese translations) — no plan doc found defining these; stale Rill refs in ExploreBookmarks.svelte etc. are for a different function

## Final Verification
- `grep -r getRillDefaultExploreState` → **0 results** ✅
- `grep -r rillDefaultExploreState` → **0 results** ✅
- `ls stores/get-*` → shows `get-stardata-default-explore-state.ts` ✅
- Barrel import `stores/index.ts` → references new file name ✅

## Relevant Files
- `stores/get-stardata-default-explore-state.ts`: Renamed file, function `getStarDataDefaultExploreState(v)`
- `stores/index.ts`: Barrel export updated
- `state-managers/loaders/DashboardStateDataLoader.ts`: Consumer — all refs fixed
- `stores/test-data/data.ts`: Test data — import + call fixed
- `state-managers/loaders/DashboardStateSync.ts`: Contains stale refs to `getRillDefaultExploreUrlParams` (separate scope)
- `bookmarks/ExploreBookmarks.svelte`: Contains stale refs to `createRillDefaultExploreUrlParamsV2` (separate scope)

## Next If Resuming
1. If the goal is broader Rill→StarData rebranding, repeat this pattern for `getRillDefaultExploreUrlParams` (3 files, ~7 usages) and `createRillDefaultExploreUrlParamsV2` (1 file, 1 usage)
2. If Categories F/G exist as actionable tasks, recover the original plan that defined them
