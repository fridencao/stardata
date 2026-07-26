# UI Design Unification Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Unify Portal, Studio, and Chat frontend UIs under a single enterprise BI dashboard aesthetic — eliminating the dark sidebar Studio layout, consolidating navigation into PortalNav + StudioTabs, adding design token utility classes, shared components (SectionHeader, StatusBadge), and manual dark mode toggle.

**Architecture:** Replace `StudioSidebar` with `StudioTabs` (horizontal top tabs under PortalNav). All Studio pages flow through a unified `<main>` container layout. Add utility CSS classes in app.css for surface/text/border/card patterns. Create SectionHeader + StatusBadge as reusable Svelte components. Wire up existing `themeControl` store into PortalNav.

**Tech Stack:** Svelte 5, Tailwind v4 (via `@tailwindcss/vite`), `web-common` preset chain, CSS custom properties.

---

## Chunk 1: Foundation — Design Token Utility Classes + Theme Toggle

### Task 1: Add design token utilities to app.css

**Files:**
- Modify: `web-common/src/app.css`

The app.css already defines semantic CSS variables (`--surface-base`, `--fg-primary`, etc.). We need `@layer utilities` classes that map them to semantic Tailwind class names so pages can use `bg-app-surface` instead of `bg-gray-50`.

Add the following after the existing `@layer utilities` block (around line 6):

```css
@layer utilities {
  /* ... existing .ui-copy-number, .ui-measure-bar-* rules ... */

  /* App-wide design token classes */
  .bg-app-surface   @apply bg-surface-subtle;
  .bg-app-card      @apply bg-surface-card;
  .border-app       @apply border-border;

  /* Card base pattern */
  .card-basic @apply rounded-xl border-app bg-app-card shadow-sm hover:shadow-md transition-shadow duration-150;
  .card-hero  @apply rounded-2xl border-app bg-app-card shadow-sm hover:shadow-md transition-shadow duration-150;

  /* Dark-mode aware strong border */
  .border-app-strong @apply border-gray-300 dark:border-gray-600;
}
```

"`★ Insight ─────────────────────────────────────`
We map to Tailwind's existing `bg-surface-subtle`, `bg-surface-card`, `border-border` etc. rather than creating a second layer of CSS variable references. This keeps the utility chain one level deep and lets Tailwind's JIT generate the right classes in both light/dark modes automatically.
`─────────────────────────────────────────────────`"

Also update Typography overrides at the end of the `@layer base` block (after h1-h4 and body rules). Append:

```css
/* Compact enterprise typography scale — all sizes relative to 11px base */
h1, .text-3xl { font-size: 16px; line-height: 1.3; }
.text-2xl { font-size: 14px; line-height: 1.35; }
.text-xl { font-size: 13px; line-height: 1.4; }
.text-lg { font-size: 12px; line-height: 1.4; }
.text-base { font-size: 11px; line-height: 1.45; }
.text-sm { font-size: 10.5px; line-height: 1.5; }
.text-xs { font-size: 9.5px; line-height: 1.5; }
.text-[13px] { font-size: 12px; }
.text-[12px] { font-size: 11px; }
.text-[11px] { font-size: 10px; }
.text-[10.5px] { font-size: 9.5px; }
.text-[10px] { font-size: 9px; }
.text-[9.5px] { font-size: 8.5px; }
```

- [ ] **Step 1: Edit web-common/src/app.css — add design token utilities**

Verify the existing `@layer utilities` block around line 6, then append our new classes after it. Verify the `body` section still reads correctly with PingFang SC font-family and 11px base size (already done in previous session).

- [ ] **Step 2: Edit web-common/src/app.css — append typography overrides**

Append the compact typography scale after the existing h3/h4 rules.

- [ ] **Step 3: Verify CSS compiles**

Run: `npm run dev -w web-common 2>&1 | head -20` or simply start `npm run dev` at root and check for Svelte compilation errors.

- [ ] **Step 4: Commit**

```bash
git add web-common/src/app.css
git commit -m "refactor(ui): add design token utility classes and compact typography scale"
```

### Task 2: Wire existing themeControl into PortalNav

**Files:**
- Modify: `web-local/src/features/portal/PortalNav.svelte`

`theme-control.ts` already exists with a complete implementation: `themeControl` manages light/dark/system preferences in localStorage, applies `html.dark` class, and exposes `.current` writable store. We just need a simple UI trigger.

Replace the role switcher div in PortalNav (line 46-63) with an expanded control group:

```svelte
<div class="flex items-center gap-2 ml-auto">
  <!-- Theme toggle -->
  <button
    class="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100 hover:text-gray-700 transition-colors"
    on:click={() => {
      const current = $themeControl.current;
      $themeControl.set(current === 'dark' ? 'light' : 'dark');
    }}
    title={$themeControl.current === 'dark' ? '切换浅色模式' : '切换深色模式'}
  >
    {$themeControl.current === 'dark' ? '☀️' : '🌙'}
  </button>
  <!-- Role switcher -->
  <div class="flex rounded-lg bg-gray-100 p-0.5 text-xs">
    ...
  </div>
</div>
```

Import `themeControl` at top:
```ts
import { themeControl } from "@rilldata/web-common/features/themes/theme-control";
```

- [ ] **Step 1: Edit PortalNav.svelte — import themeControl and add toggle button**

Add import after existing imports. Then find the role switcher `<div>` (line 46) and insert a theme toggle button before it inside the existing `ml-auto flex items-center gap-3` wrapper.

- [ ] **Step 2: Verify PortalNav renders**

Start dev server and navigate to `/` — confirm theme toggle appears next to role switcher, clicking it changes icon between ☀️/🌙.

- [ ] **Step 3: Commit**

```bash
git add web-local/src/features/portal/PortalNav.svelte
git commit -m "feat(ui): add theme toggle button to PortalNav wired to themeControl"
```

### Task 3: Refactor PortalNav — remove manual studio button, auto-highlight Studio routes

**Files:**
- Modify: `web-local/src/features/portal/PortalNav.svelte`

The "技术工作台" button should no longer be manually managed. Instead:
1. Remove the standalone `<a href="/studio">🔧 技术工作台</a>` (lines 64-71)
2. When on a Studio route (`/studio/*`), auto-show a "Studio" link in the nav bar with active highlight
3. The role switcher ("业务视角"/"技术视角") remains as-is

Update `links` array conditionally:

```ts
const baseLinks = [
  { label: "首页", href: "/" },
  { label: "对话", href: "/chat" },
  { label: "看板", href: "/boards" },
];

// Dynamically include Studio link when on Studio routes
$: links = isStudioMode
  ? [{ label: "Studio", href: "/studio" }, ...baseLinks]
  : baseLinks;

$: isStudioMode = PORTAL_ROUTE_PREFIXES.some(p => p === "/" && pathname !== "/") 
                   || STUDIO_ROUTE_PREFIXES.some(prefix => pathname.startsWith(prefix));
```

Actually simpler — since we have `isStudioRoute` already in `route-constants.ts`, use it directly:

```ts
import { isStudioRoute } from "$lib/routes/route-constants"; // adjust path alias
```

Wait — `route-constants.ts` is inside `src/routes/`. Let me check how other files import it:

From `+layout.svelte` line 28: `import { isPortalRoute, isStudioRoute } from "./route-constants";`

So PortalNav should import from relative path `../../routes/route-constants` — actually PortalNav lives in `features/portal/` so relative is `../routes/route-constants` or use `$lib/...` if aliased. Let me check the path alias setup:

- [ ] **Step 1: Check path alias for routes directory**

Run: `grep -A5 "alias" /Users/xinjian/Work/Project/RD/StarData/web-local/svelte.config.js 2>/dev/null || grep vite /Users/xinjian/Work/Project/RD/StarData/web-local/vite.config.ts | head -10`

If no alias, import via relative path: `import { isStudioRoute } from "../../routes/route-constants";`

- [ ] **Step 2: Update PortalNav.svelte**

Import `isStudioRoute`. Add conditional `studioLink`:

```ts
$: showStudioLink = isStudioRoute(pathname);
```

Replace the hardcoded `{#if $portalRole === "tech"} <a href="/studio">...</a> {/if}` block (lines 64-71) with:

```svelte
{#if showStudioLink}
  <a
    href="/studio"
    class="rounded-lg px-3 py-1.5 text-sm font-semibold text-primary-700 bg-primary-50"
  >
    技术工作台
  </a>
{/if}
```

Keep the old button only when NOT on Studio route AND role is tech.

- [ ] **Step 3: Test**

Navigate to `/`, `/chat`, `/studio` — confirm:
- On `/` and `/chat`: Studio link hidden (unless role is tech)
- On `/studio/*`: "技术工作台" tab shows highlighted in nav
- Role switcher still works

- [ ] **Step 4: Commit**

```bash
git add web-local/src/features/portal/PortalNav.svelte
git commit -m "refactor(ui): auto-detect Studio routes in PortalNav, remove manual studio button"
```

---

## Chunk 2: New Components — StudioTabs + SectionHeader + StatusBadge

### Task 4: Create StudioTabs component

**Files:**
- Create: `web-local/src/features/studio/StudioTabs.svelte`

A horizontal tab bar matching PortalNav style, rendered under PortalNav within Studio pages.

```svelte
<script lang="ts">
  import { page } from "$app/stores";

  const tabs = [
    { label: "概览", href: "/studio" },
    { label: "数据源", href: "/studio/sources" },
    { label: "语义层", href: "/studio/semantics" },
    { label: "发布", href: "/studio/publish" },
  ];

  $: pathname = $page.url.pathname;

  function isActive(href: string, path: string): boolean {
    if (href === "/studio") return path === "/studio";
    return path.startsWith(href);
  }
</script>

<div class="h-[52px] flex items-center gap-1 border-b border-gray-200 bg-white px-8">
  {#each tabs as tab (tab.href)}
    <a
      href={tab.href}
      class="relative px-3.5 py-2 text-sm no-underline transition-colors {isActive(tab.href, pathname)
          ? 'font-bold text-primary-700'
          : 'text-gray-600 hover:text-gray-900'}"
    >
      {tab.label}
      {#if isActive(tab.href, pathname)}
        <span class="absolute bottom-0 left-0 right-0 h-[2px] bg-primary-600" />
      {/if}
    </a>
  {/each}
</div>
```

Key decisions:
- Height 52px as specified
- 2px primary underline indicator for active state
- Matches PortalNav's white bg + gray-200 border
- Active tab gets bold text + primary color

- [ ] **Step 1: Create StudioTabs.svelte**

Create the file at `web-local/src/features/studio/StudioTabs.svelte` with the code above.

- [ ] **Step 2: Verify compilation**

```bash
cd web-local && npx svelte-check -- compilerOptions { "baseUrl": ".", "paths": { "@rilldata/web-common": ["../web-common/src"] } } 2>&1 | head -20
```

Or simply run `npm run dev` and check browser console for Svelte compile errors.

- [ ] **Step 3: Commit**

```bash
git add web-local/src/features/studio/StudioTabs.svelte
git commit -m "feat(ui): add StudioTabs horizontal tab component for studio navigation"
```

### Task 5: Create SectionHeader component

**Files:**
- Create: `web-local/src/features/studio/SectionHeader.svelte`

```svelte
<script lang="ts">
  export let title: string;
  export let description: string = "";
</script>

<div>
  <h2 class="text-lg font-bold text-fg-primary">{title}</h2>
  {#if description}
    <p class="mt-0.5 text-[13px] text-fg-muted">{description}</p>
  {/if}
  <slot name="actions" />
</div>
```

Usage in pages replaces duplicated `h2 + p` blocks:
```svelte
<!-- Before (per page) -->
<h2 class="text-lg font-bold text-gray-900">概览 · 配置健康度</h2>
<p class="mt-0.5 text-[13px] text-gray-400">一屏了解:业务现在能问什么,还缺什么</p>

<!-- After -->
<SectionHeader title="概览 · 配置健康度" description="一屏了解:业务现在能问什么,还缺什么" />
```

- [ ] **Step 1: Create SectionHeader.svelte**

Create the file at `web-local/src/features/studio/SectionHeader.svelte`.

- [ ] **Step 2: Commit**

```bash
git add web-local/src/features/studio/SectionHeader.svelte
git commit -m "feat(ui): add SectionHeader component for unified page titles in Studio"
```

### Task 6: Create StatusBadge component

**Files:**
- Create: `web-common/src/components/StatusBadge.svelte`

Reusable badge replacing 4+ inline badge implementations across Studio pages.

```svelte
<script lang="ts">
  export let variant: "success" | "error" | "warning" | "neutral" | "info" = "neutral";
  export let size: "sm" | "md" = "md";
</script>

<span
  class="inline-flex items-center rounded-md font-semibold no-underline {sizeClass} {variantClass}"
  class:rounded-md
>
  <slot />
</span>

<style>
  /* Variant styles */
  .variant-success { @apply bg-green-50 text-green-700; }
  .variant-error   { @apply bg-red-50 text-red-600; }
  .variant-warning { @apply bg-yellow-50 text-yellow-700; }
  .variant-neutral { @apply bg-gray-100 text-gray-500; }
  .variant-info    { @apply bg-primary-50 text-primary-700; }

  /* Size styles */
  .size-sm { @apply px-1.5 py-0.5 text-[10.5px]; }
  .size-md { @apply px-2 py-0.5 text-[11px]; }
</style>
```

Handles `dark:` variants by relying on Tailwind's `dark:` selector prefix:

```css
  :global(.dark) .variant-success { @apply bg-green-900/30 text-green-300; }
  :global(.dark) .variant-error   { @apply bg-red-900/30 text-red-300; }
  :global(.dark) .variant-neutral { @apply bg-gray-700/50 text-gray-400; }
  :global(.dark) .variant-info    { @apply bg-primary-900/30 text-primary-300; }
```

- [ ] **Step 1: Create StatusBadge.svelte**

Create at `web-common/src/components/StatusBadge.svelte`.

- [ ] **Step 2: Commit**

```bash
git add web-common/src/components/StatusBadge.svelte
git commit -m "feat(ui): add StatusBadge component for consistent status labels across Studio pages"
```

---

## Chunk 3: Studio Layout Rewrite

### Task 7: Rewrite studio/+layout.svelte

**Files:**
- Modify: `web-local/src/routes/studio/+layout.svelte`

Replace the entire content with the unified layout:

```svelte
<script lang="ts">
  import StudioTabs from "../../features/studio/StudioTabs.svelte";
</script>

<div class="flex min-h-0 flex-col bg-app-surface">
  <StudioTabs />
  <main class="flex-1 overflow-y-auto p-8 xl:max-w-6xl xl:mx-auto">
    <slot />
  </main>
</div>
```

Changes from current:
- Remove `StudioSidebar` import and dark panel `bg-[#191B26] w-[216px]`
- Remove outer `flex` + `bg-gray-100` wrapper
- Add `StudioTabs` horizontal nav bar (defined at 52px height in component)
- Single `<main>` with `p-8` padding and `xl:max-w-6xl xl:mx-auto` max-width constraint
- Uses `bg-app-surface` utility class (maps to `bg-surface-subtle`)

- [ ] **Step 1: Rewrite studio/+layout.svelte**

Read the current file first, then write the new content. The change is near-total replacement — just keep the `<script>` structure and swap the template.

- [ ] **Step 2: Verify rendering**

Navigate to `/studio` — confirm:
- No more dark sidebar
- StudioTabs renders with 4 tabs
- Page content flows properly in the `<main>` area
- Scroll works within `<main>`

- [ ] **Step 3: Commit**

```bash
git add web-local/src/routes/studio/+layout.svelte
git commit -m "refactor(ui): replace dark sidebar with StudioTabs and unified main container"
```

### Task 8: Update studio overview page (+page.svelte)

**Files:**
- Modify: `web-local/src/routes/studio/+page.svelte`

Replacements:
1. Replace raw `<h2>...</h2><p>...</p>` with `<SectionHeader title="概览" description="一屏了解:业务现在能问什么,还缺什么" />`
2. Replace the stats grid cards styling — each card currently uses `rounded-xl border border-gray-200 bg-white`. Change to use `.card-basic` utility class
3. Fix the "近7天提问命中率" card which has no `<a>` wrapper — make it display-only or link to the planned M5 feature

```svelte
<!-- Replace each stat card wrapper -->
<!-- Before -->
<div class="rounded-xl border border-gray-200 bg-white px-4 py-4 hover:border-gray-300 transition-colors">
<!-- After -->
<div class="card-basic px-4 py-4">
```

- [ ] **Step 1: Edit studio/+page.svelte**

Use SectionHeader, apply `.card-basic` to all 4 stat cards, remove `hover:border-gray-300 transition-colors` (shadow transition is now in .card-basic).

- [ ] **Step 2: Verify**

Navigate to `/studio` — confirm visual parity with the design spec.

- [ ] **Step 3: Commit**

```bash
git add web-local/src/routes/studio/+page.svelte
git commit -m "refactor(ui): unify overview page with SectionHeader and card-basic utility"
```

---

## Chunk 4: Update Remaining Studio Pages

### Task 9: Update sources page

**Files:**
- Modify: `web-local/src/routes/studio/sources/+page.svelte`

Replacements:
1. Replace header: `<SectionHeader title="数据源" description="已接入连接器 · 向导式新增 · 表结构浏览" />`
2. Connector cards: change `rounded-xl border-gray-200 bg-white` → `.card-basic`
3. Empty state card: use `.card-hero` instead of inline dashed border
4. Table explorer wrapper: use `.card-basic` for the border/shadow around ConnectorExplorer
5. Replace "OLAP" inline badge → `<StatusBadge variant="info" size="sm">OLAP</StatusBadge>`

- [ ] **Step 1: Edit sources/+page.svelte**

Apply all replacements. The `<AddDataModal>` component stays unchanged.

- [ ] **Step 2: Commit**

```bash
git add web-local/src/routes/studio/sources/+page.svelte
git commit -m "refactor(ui): unify sources page with SectionHeader and card-basic"
```

### Task 10: Update semantics pages

**Files:**
- Modify: `web-local/src/routes/studio/semantics/+page.svelte`
- Modify: `web-local/src/routes/studio/semantics/[name]/+page.svelte`

**semantics/index page:**
1. Replace header → `<SectionHeader title="语义层" description="指标/维度定义 · 中文别名(label_cn) · 无代码编辑" />`
2. Replace table wrapper border: `rounded-xl border border-gray-200 bg-white` → `.card-basic`
3. Inline status badges (`bg-green-50 ...`, `bg-red-50 ...`) → `<StatusBadge variant="success">有效</StatusBadge>`

**semantics/[name] page:**
1. Replace back-link + heading row → `<SectionHeader title={name} description="编辑自动保存" />`
2. Replace table wrapper → `.card-basic`
3. Replace empty state border → `.card-hero`
4. Replace "返回列表" link color → `text-accent-primary-action`

- [ ] **Step 1: Edit semantics/+page.svelte**

Apply SectionHeader, .card-basic, and StatusBadge.

- [ ] **Step 2: Edit semantics/[name]/+page.svelte**

Apply SectionHeader and .card-basic.

- [ ] **Step 3: Commit**

```bash
git add web-local/src/routes/studio/semantics/+page.svelte web-local/src/routes/studio/semantics/\[name\]/+page.svelte
git commit -m "refactor(ui): unify semantics pages with SectionHeader, card-basic, StatusBadge"
```

### Task 11: Update publish page

**Files:**
- Modify: `web-local/src/routes/studio/publish/+page.svelte`

Replacements:
1. Replace header → `<SectionHeader title="发布" description="控制哪些指标集对业务门户(推荐问题 + Chat AI)可见" />`
2. Info banner (blue box): keep as-is, it serves a different purpose
3. Table wrapper → `.card-basic`
4. Row badges → `<StatusBadge variant="success">已发布</StatusBadge>` etc.
5. Inline checkbox row styling consistent with card-basic pattern

- [ ] **Step 1: Edit publish/+page.svelte**

Apply SectionHeader, .card-basic, and StatusBadge. Keep `RequestsTodo` component as-is (it's not a visual concern here).

- [ ] **Step 2: Commit**

```bash
git add web-local/src/routes/studio/publish/+page.svelte
git commit -m "refactor(ui): unify publish page with SectionHeader, card-basic, StatusBadge"
```

---

## Chunk 5: Cleanup + Verification

### Task 12: Delete StudioSidebar and verify

**Files:**
- DELETE: `web-local/src/features/studio/StudioSidebar.svelte`
- No other references to it remain after layout rewrite

Before deleting, verify no remaining references:

```bash
grep -rn "StudioSidebar" /Users/xinjian/Work/Project/RD/StarData/web-local/src/ --include="*.svelte" --include="*.ts"
```

Expected output: only the import in the OLD `studio/+layout.svelte` (which we already rewrote).

- [ ] **Step 1: Confirm no remaining references**

Run the grep above. If any other file imports StudioSidebar, remove that import.

- [ ] **Step 2: Delete StudioSidebar.svelte**

```bash
rm web-local/src/features/studio/StudioSidebar.svelte
```

- [ ] **Step 3: Full build verification**

```bash
npm run dev  # starts both runtime and web dev server
```

Navigate through ALL routes and verify:
- `/` (portal home) — renders with PortalNav
- `/chat` (full-page chat) — full screen, no nav
- `/chat?new=true&q=...` — auto-send still works
- `/boards` — PortalNav + card grid
- `/boards/[name]` — canvas + dashboard chat
- `/studio` — StudioTabs + overview card grid
- `/studio/sources` — StudioTabs + source cards
- `/studio/semantics` — StudioTabs + table with StatusBadges
- `/studio/semantics/[name]` — StudioTabs + VisualMetrics editor
- `/studio/publish` — StudioTabs + publish table
- `/files` — legacy IDE, should still work with legacy header

- [ ] **Step 4: Verify dark mode toggle**

Click ☀️/🌙 in PortalNav. Navigate between portal and studio. Dark class should persist across all routes via `document.documentElement.classList`.

- [ ] **Step 5: Clean commit**

```bash
git add -A web-local/src/features/studio/StudioSidebar.svelte
git rm web-local/src/features/studio/StudioSidebar.svelte
git commit -m "chore(ui): remove StudioSidebar, replaced by StudioTabs"
```

### Task 13: Frontend quality checks

- [ ] **Step 1: Type check**

```bash
npx svelte-check --threshold warning 2>&1 | tail -30
```

Fix any type errors introduced by the refactoring.

- [ ] **Step 2: Lint**

```bash
npm run lint 2>&1 | tail -20
```

- [ ] **Step 3: Unit tests (web-common)**

```bash
npm run test -w web-common 2>&1 | tail -20
```

- [ ] **Step 4: Final commit**

```bash
git commit -m "ci(ui): run frontend quality checks after UI unification refactor"
```

---

## File Change Summary

| File | Action | Phase |
|------|--------|-------|
| `web-common/src/app.css` | Modify: add utility classes & typography overrides | 1 |
| `web-common/src/components/StatusBadge.svelte` | Create | 2 |
| `web-local/src/features/portal/PortalNav.svelte` | Modify: theme toggle + auto-detect Studio | 1, 3 |
| `web-local/src/features/studio/StudioTabs.svelte` | Create | 2 |
| `web-local/src/features/studio/SectionHeader.svelte` | Create | 2 |
| `web-local/src/features/studio/StudioSidebar.svelte` | Delete | 5 |
| `web-local/src/routes/studio/+layout.svelte` | Rewrite | 3 |
| `web-local/src/routes/studio/+page.svelte` | Modify: SectionHeader + card-basic | 3 |
| `web-local/src/routes/studio/sources/+page.svelte` | Modify: SectionHeader + card-basic + StatusBadge | 4 |
| `web-local/src/routes/studio/semantics/+page.svelte` | Modify: SectionHeader + card-basic + StatusBadge | 4 |
| `web-local/src/routes/studio/semantics/[name]/+page.svelte` | Modify: SectionHeader + card-basic | 4 |
| `web-local/src/routes/studio/publish/+page.svelte` | Modify: SectionHeader + card-basic + StatusBadge | 4 |

Total files changed: **12** (10 modifications, 3 creations, 1 deletion)
