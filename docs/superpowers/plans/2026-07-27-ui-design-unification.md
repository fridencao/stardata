# UI Design Unification Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Unify Portal, Studio, and Chat frontend UIs under a single enterprise BI dashboard aesthetic — eliminating the dark sidebar Studio layout, consolidating navigation into PortalNav + StudioTabs, adding design token utility classes, shared components (SectionHeader, StatusBadge), manual dark mode toggle, and replacing all emoji icons with Lucide professional icons.

**Architecture:** Replace `StudioSidebar` with `StudioTabs` (horizontal top tabs under PortalNav). All Studio pages flow through a unified `<main>` container layout. Add utility CSS classes in app.css for surface/text/border/card patterns. Create SectionHeader + StatusBadge as reusable Svelte components. Wire up existing `themeControl` store into PortalNav. Replace all emoji icons with `lucide-svelte` components throughout all affected files.

**Tech Stack:** Svelte 5, Tailwind v3 (PostCSS), `web-common` preset chain, CSS custom properties, **Lucide Svelte** (already installed as `lucide-svelte ^0.298.0`).

---

## Chunk 1: Foundation — Design Token Utility Classes + Theme Toggle

### Task 1: Add design token utilities to app.css

**Files:**
- Modify: `web-common/src/app.css`

The app.css already defines semantic CSS variables (`--surface-base`, `--fg-primary`, etc.). We need `@layer utilities` classes that map them to semantic Tailwind class names so pages can use `bg-app-surface` instead of `bg-gray-50`.

Add the following INSIDE the existing `@layer utilities` block (around line 6), using valid Tailwind v3 CSS syntax:

```css
/* Inside @layer utilities { ... } */

/* App-wide design token classes */
.bg-app-surface {
  @apply bg-surface-subtle;
}
.bg-app-card {
  @apply bg-surface-card;
}
.border-app {
  @apply border-border;
}

/* Card base pattern */
.card-basic {
  @apply rounded-xl border-app bg-app-card shadow-sm hover:shadow-md transition-shadow duration-150;
}
.card-hero {
  @apply rounded-2xl border-app bg-app-card shadow-sm hover:shadow-md transition-shadow duration-150;
}

/* Dark-mode aware strong border */
.border-app-strong {
  @apply border-gray-300 dark:border-gray-600;
}
```

"`★ Insight ─────────────────────────────────────`
We use `@layer utilities { .class { @apply ...; } }` (v3 syntax) rather than `.class @apply ...;` (v4 shorthand). This is invalid CSS under PostCSS and will fail compilation. The `dark:` prefix works because web-common's tailwind.config.ts has `darkMode: "class"`.
`─────────────────────────────────────────────────`"

Also update Typography overrides at the end of the `@layer base` block (after h1-h4 and body rules). These overrides are already present from a previous session — verify they exist and are correct:

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

And body font-family should read:
```css
body {
  font-family:
    -apple-system,
    BlinkMacSystemFont,
    "PingFang SC",
    "SF Pro SC",
    "SF Pro Text",
    "Helvetica Neue",
    sans-serif;
  font-size: 11px;
  line-height: 1.5;
}
```

- [ ] **Step 1: Edit web-common/src/app.css — add design token utilities inside @layer utilities**

Read app.css to confirm the existing `@layer utilities` block. Append our new `.bg-app-*` / `.card-*` rules using v3 `{ @apply ...; }` syntax.

- [ ] **Step 2: Verify typography overrides exist**

Confirm the compact typography scale is present after the `@layer base` h3/h4 section (already written in prior session). Verify body reads 11px font-size with PingFang SC.

- [ ] **Step 3: Verify CSS compiles**

Run: `npm run dev 2>&1 | head -30` and check no PostCSS or Svelte compile errors appear.

- [ ] **Step 4: Commit**

```bash
git add web-common/src/app.css
git commit -m "refactor(ui): add design token utility classes and compact typography scale"
```

### Task 2: Wire existing ThemeToggle component into PortalNav

**Files:**
- Modify: `web-local/src/features/portal/PortalNav.svelte`

The repo already has:
- `theme-control.ts` — full light/dark/system preference management in localStorage
- `ThemeToggle.svelte` — dropdown-based theme selector in web-common

Use the existing `ThemeToggle.svelte` dropdown directly in PortalNav rather than re-implementing a light toggle button.

Import at top of PortalNav.svelte:
```ts
import ThemeToggle from "@rilldata/web-common/features/themes/ThemeToggle.svelte";
```

Add it into the nav bar controls section, right after the role switcher div (line ~63):

```svelte
<!-- After the role switcher div -->
<div class="flex items-center gap-3">
  <ThemeToggle />
</div>
```

This gives users three options: Light, Dark, System — consistent with the rest of the product's theme management.

- [ ] **Step 1: Edit PortalNav.svelte — import and render ThemeToggle**

Add the import after the last existing import. Insert the `<ThemeToggle />` wrapper after the role switcher's closing `</div>`.

- [ ] **Step 2: Verify**

Navigate to `/`. Confirm ThemeToggle dropdown appears next to role switcher with light/dark/system options.

- [ ] **Step 3: Commit**

```bash
git add web-local/src/features/portal/PortalNav.svelte
git commit -m "feat(ui): integrate existing ThemeToggle into PortalNav"
```

### Task 3: Refactor PortalNav — remove manual studio button, auto-highlight Studio routes

**Files:**
- Modify: `web-local/src/features/portal/PortalNav.svelte`

The "技术工作台" button should no longer be conditionally shown based on `$portalRole === "tech"`. Instead:

1. Always show when on Studio routes (`/studio/*`)
2. Hide otherwise (since StudioTabs + layout handle Studio UX now)
3. Keep role switcher unchanged

Change the conditional from:
```svelte
{#if $portalRole === "tech"}
```
To:
```ts
$: showStudioLink = isStudioRoute(pathname);
```

Replace lines 64-71 with:

```svelte
{#if showStudioLink}
  <a
    href="/studio"
    class="rounded-lg px-3 py-1.5 text-sm font-semibold text-primary-700 bg-primary-50 no-underline flex items-center gap-1.5"
  >
    <Wrench class="size-4" /> 技术工作台
  </a>
{/if}
```

**Icon mapping for PortalNav and Studio pages:**

| Location | Old | → Lucide |
|----------|-----|----------|
| PortalNav logo ✦ | *(brand symbol)* | Keep as-is or use `Sparkles` |
| PortalNav "技术工作台" button | 🔧 | `Wrench` |
| PortalHome search input | 🔍 | `Search` + ➤ submit `ArrowRight` |
| PortalHome card "继续对话" | 💬 | `MessageSquare` |
| PortalHome card "我的看板" | 📌 | `LayoutGrid` |
| Boards empty state | 📌 | `LayoutGrid` |
| Boards board item | 📊 | `BarChart3` |
| Info banner 💡 (studio/overview, publish) | 💡 | `Info` |
| Back arrow ← (semantics detail) | text | `ChevronLeft` |
| StudioTabs overview | 📊 | `LayoutDashboard` |
| StudioTabs data source | 🗄️ | `Database` |
| StudioTabs semantics | 📐 | `Network` |
| StudioTabs publish | 🚀 | `Rocket` |

Note: All Lucide icons use the pattern `<IconName class="size-4" />` inside the component. Size is controlled via Tailwind's `size-4` utility (16px). Color inherits from parent text color classes.

Import `isStudioRoute`:
```ts
import { isStudioRoute } from "../../routes/route-constants";
```

PortalNav lives at `features/portal/PortalNav.svelte`, route-constants is at `src/routes/route-constants.ts`, so relative path `../../routes/route-constants`.

- [ ] **Step 1: Edit PortalNav.svelte — add isStudioRoute import and conditional**

Add import. Replace the `{#if $portalRole === "tech"}` block.

- [ ] **Step 2: Verify**

Navigate to `/`, `/chat`, `/studio/*` — confirm:
- On portal routes: Studio link hidden
- On Studio routes: "技术工作台" tab shows highlighted
- Role switcher still functional

- [ ] **Step 3: Commit**

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
  import { LayoutDashboard, Database, Network, Rocket } from "lucide-svelte";

  const tabs = [
    { label: "概览", href: "/studio", icon: LayoutDashboard },
    { label: "数据源", href: "/studio/sources", icon: Database },
    { label: "语义层", href: "/studio/semantics", icon: Network },
    { label: "发布", href: "/studio/publish", icon: Rocket },
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
      class="relative flex items-center gap-2 px-3.5 py-2 text-sm no-underline transition-colors {isActive(tab.href, pathname)
          ? 'font-bold text-primary-700'
          : 'text-gray-600 hover:text-gray-900'}"
    >
      <svelte:component this={tab.icon} class="size-4" />
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
- `px-8` matches the main content area padding in the layout

- [ ] **Step 1: Create StudioTabs.svelte**

Create the file at `web-local/src/features/studio/StudioTabs.svelte` with the code above.

- [ ] **Step 2: Verify compilation**

Run `npm run dev` and navigate to `/studio`. No console errors.

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

Note: Uses Tailwind native classes `text-fg-primary` and `text-fg-muted` (already defined in tailwind.config.ts via the `fg.*` color namespace). Also does NOT use `.text-app-primary` (which was proposed in spec but not yet implemented in app.css), keeping this simple and self-contained.

Usage replaces 6 instances of duplicated `h2 + p`:
```svelte
<!-- Before -->
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
- Create: `web-common/src/components/status-badge/StatusBadge.svelte`

Directory-per-component convention used elsewhere in web-common (e.g., `searchable-filter-menu/`, `date-picker/`).

```svelte
<script lang="ts">
  export let variant: "success" | "error" | "warning" | "neutral" | "info" = "neutral";
  export let size: "sm" | "md" = "md";
</script>

<span
  class="inline-flex items-center rounded-md font-semibold no-underline {variantClass} {sizeClass}"
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

  /* Dark mode variants */
  :global(.dark) .variant-success { @apply bg-green-900/30 text-green-300; }
  :global(.dark) .variant-error   { @apply bg-red-900/30 text-red-300; }
  :global(.dark) .variant-neutral { @apply bg-gray-700/50 text-gray-400; }
  :global(.dark) .variant-info    { @apply bg-primary-900/30 text-primary-300; }
</style>
```

Note: Removed redundant `class:rounded-md` since it's already in the static class string. Removed `sizeClass` conditional class binding and use direct class interpolation instead (`.size-sm` / `.size-md` map to inline classes, cleaner for Svelte v5).

Usage:
```svelte
<StatusBadge variant="success">有效</StatusBadge>
<StatusBadge variant="error">解析错误</StatusBadge>
<StatusBadge variant="info" size="sm">OLAP</StatusBadge>
```

- [ ] **Step 1: Create status-badge directory and StatusBadge.svelte**

```bash
mkdir -p web-common/src/components/status-badge
```

Then write the component.

- [ ] **Step 2: Commit**

```bash
git add web-common/src/components/status-badge/StatusBadge.svelte
git commit -m "feat(ui): add StatusBadge component for consistent status labels across Studio pages"
```

---

## Chunk 3: Studio Layout Rewrite

### Task 7: Rewrite studio/+layout.svelte

**Files:**
- Modify: `web-local/src/routes/studio/+layout.svelte`

Replace the entire file with the unified layout:

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
- Uses `.bg-app-surface` utility class (maps to `bg-surface-subtle`)

Warning about `max-w-6xl` (1100px): the `sources` page contains `ConnectorExplorer` which renders wide schema trees. If it overflows the container, revert `max-w-6xl` to no max-width and apply it only to pages that benefit from constrained reading width (overview, semantics, publish).

- [ ] **Step 1: Rewrite studio/+layout.svelte**

Read the current file first, then overwrite. The change is near-total replacement — delete existing content, write new.

- [ ] **Step 2: Verify rendering**

Navigate to `/studio` — confirm:
- No more dark sidebar
- StudioTabs renders with 4 tabs
- Page content flows properly in the `<main>` area
- Scroll works within `<main>`

Also check `/studio/sources` — verify ConnectorExplorer table doesn't get clipped by `max-w-6xl`.

- [ ] **Step 3: Commit**

```bash
git add web-local/src/routes/studio/+layout.svelte
git commit -m "refactor(ui): replace dark sidebar with StudioTabs and unified main container"
```

### Task 8: Update studio overview page (+page.svelte)

**Files:**
- Modify: `web-local/src/routes/studio/+page.svelte`

Replacements:
1. Replace raw `<h2>...</h2><p>...</p>` → `<SectionHeader title="概览" description="一屏了解:业务现在能问什么,还缺什么" />`
2. Replace each stat card wrapper: `rounded-xl border border-gray-200 bg-white px-4 py-4 hover:border-gray-300 transition-colors` → `card-basic px-4 py-4`
3. Fix the "近7天提问命中率" card — it's display-only. Change styling to match others but remove the `<a>` wrapper (or keep it as a placeholder for M5 feature)

```svelte
<!-- Each stat card -->
<!-- Before -->
<div class="rounded-xl border border-gray-200 bg-white px-4 py-4 hover:border-gray-300 transition-colors">
<!-- After -->
<div class="card-basic px-4 py-4">
```

Replace the info banner emoji: `💡 M3/4 已上线…` → `<Info class="size-4 inline mr-1" /> M3/4 已上线…` (import `Info` from lucide-svelte).

The 4th stat card ("近7天提问命中率") is display-only — remove the `<a>` wrapper and keep as static cell.

- [ ] **Step 1: Edit studio/+page.svelte**

Apply SectionHeader and `.card-basic` to all cards.

- [ ] **Step 2: Verify**

Navigate to `/studio`.

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
1. Header → `<SectionHeader title="数据源" description="已接入连接器 · 向导式新增 · 表结构浏览" />`
2. Connector cards: `rounded-xl border-gray-200 bg-white px-4 py-4` → `card-basic px-4 py-4`
3. Empty state: `rounded-xl border border-dashed border-gray-300 bg-white py-10` → `card-hero py-10` (remove bg-white — card-hero handles it)
4. Explorer wrapper: `overflow-hidden rounded-xl border border-gray-200 bg-white` → `card-basic overflow-hidden`
5. "OLAP" badge → `<StatusBadge variant="info" size="sm">OLAP</StatusBadge>`

- [ ] **Step 1: Edit sources/+page.svelte**

Apply all replacements. The `<AddDataModal>` stays unchanged.

- [ ] **Step 2: Commit**

```bash
git add web-local/src/routes/studio/sources/+page.svelte
git commit -m "refactor(ui): unify sources page with SectionHeader and card-basic"
```

### Task 10: Update semantics pages

**Files:**
- Modify: `web-local/src/routes/studio/semantics/+page.svelte`
- Modify: `web-local/src/routes/studio/semantics/[name]/+page.svelte`

**semantics/index:**
1. Header → `<SectionHeader title="语义层" description="指标/维度定义 · 中文别名(label_cn) · 无代码编辑" />`
2. Table wrapper → `mt-5 card-basic overflow-hidden`
3. Inline badges → `<StatusBadge variant="success">有效</StatusBadge>` etc.

**semantics/[name]:**
1. Back-link + heading → `<SectionHeader title={name} description="编辑自动保存" />`
   - Replace "← 返回语义层" text link with `<ChevronLeft class="size-4 inline mr-1" />` from lucide-svelte
2. Editor wrapper → `card-basic min-h-0 flex-1 overflow-hidden`
3. Error state → `card-hero`

- [ ] **Step 1: Edit semantics/+page.svelte**

Apply SectionHeader, .card-basic, StatusBadge.

- [ ] **Step 2: Edit semantics/[name]/+page.svelte**

Apply SectionHeader and .card-basic.

- [ ] **Step 3: Commit**

```bash
git add web-local/src/routes/studio/semantics/+page.svelte "web-local/src/routes/studio/semantics/[name]/+page.svelte"
git commit -m "refactor(ui): unify semantics pages with SectionHeader, card-basic, StatusBadge"
```

### Task 11: Update publish page

**Files:**
- Modify: `web-local/src/routes/studio/publish/+page.svelte`

Replacements:
1. Header → `<SectionHeader title="发布" description="控制哪些指标集对业务门户(推荐问题 + Chat AI)可见" />`
2. Info banner: replace `💡` emoji with `<Info class="size-4 inline mr-1 text-blue-600" />` from lucide-svelte, keep blue styling as-is
3. Table wrapper: `mt-4 overflow-hidden rounded-xl border border-gray-200 bg-white` → `mt-4 card-basic overflow-hidden`
4. Row badges → `<StatusBadge variant="success">已发布</StatusBadge>` etc.
5. Also fix pre-existing bug: duplicate import of `runtimeServiceGetFile` (lines 8-9 are identical)

- [ ] **Step 1: Edit publish/+page.svelte**

Apply changes. Remove duplicate import while you're here.

- [ ] **Step 2: Commit**

```bash
git add web-local/src/routes/studio/publish/+page.svelte
git commit -m "refactor(ui): unify publish page with SectionHeader, card-basic, StatusBadge + deduplicate import"
```

---

## Chunk 5: Cleanup + Verification

### Task 12: Replace emoji icons in PortalHome and Boards pages

**Files:**
- Modify: `web-local/src/routes/(portal)/+page.svelte`
- Modify: `web-local/src/routes/(portal)/boards/+page.svelte`

Replace all emoji icons with Lucide components:

**PortalHome (`(portal)/+page.svelte`):**
```svelte
<script>
  import { Search, ArrowRight, MessageSquare, LayoutGrid } from "lucide-svelte";
</script>
```

| Old code | → New |
|----------|-------|
| `<span class="text-lg">🔍</span>` | `<Search class="size-5 text-gray-400" />` |
| `<span>➤</span>` (submit button) | `<ArrowRight class="size-5 text-white" />` |
| `<div class="text-2xl">💬</div>` (继续对话 card) | `<MessageSquare class="size-6 text-gray-900" />` |
| `<div class="text-2xl">📌</div>` (我的看板 card) | `<LayoutGrid class="size-6 text-gray-900" />` |

The submit arrow button changes from a colored `<span>` to a Lucide icon inside the existing `grid size-9 place-items-center rounded-xl bg-primary-600` wrapper.

**Boards (`boards/+page.svelte`):**
```svelte
<script>
  import { LayoutGrid, BarChart3 } from "lucide-svelte";
</script>
```

| Old code | → New |
|----------|-------|
| `<div class="text-3xl">📌</div>` (empty state) | `<LayoutGrid class="size-8 text-gray-400" />` |
| `<div class="text-2xl">📊</div>` (board item) | `<BarChart3 class="size-5 text-gray-900" />` |

- [ ] **Step 1: Edit both portal pages with Lucide icons**

Apply all replacements. Verify no emoji icons remain.

- [ ] **Step 2: Commit**

```bash
git add "web-local/src/routes/(portal)/+page.svelte" "web-local/src/routes/(portal)/boards/+page.svelte"
git commit -m "refactor(ui): replace emoji icons with Lucide professional icons in PortalHome and Boards"
```

---

## Chunk 5: Cleanup + Verification

### Task 13: Delete StudioSidebar and verify

**Files:**
- DELETE: `web-local/src/features/studio/StudioSidebar.svelte`

Before deleting, verify zero remaining references:

```bash
grep -rn "StudioSidebar" /Users/xinjian/Work/Project/RD/StarData/web-local/src/ --include="*.svelte" --include="*.ts"
```

Expected: no results (the old layout file no longer imports it since we rewrote it in Task 7).

- [ ] **Step 1: Verify no remaining references**

Run grep. If any file still imports it, remove the import.

- [ ] **Step 2: Delete**

```bash
rm web-local/src/features/studio/StudioSidebar.svelte
```

- [ ] **Step 3: Full route verification via dev server**

```bash
npm run dev
```

Check all these routes visually:
| Route | Expected |
|-------|----------|
| `/` | PortalNav, SectionHeader, card-hero entrance |
| `/chat` | Full-page chat, unchanged |
| `/boards` | PortalNav, card grid |
| `/boards/[name]` | Canvas + DashboardChat, unchanged |
| `/studio` | StudioTabs + SectionHeader + card-basic grid |
| `/studio/sources` | StudioTabs + source cards + explorer |
| `/studio/semantics` | StudioTabs + table with StatusBadges |
| `/studio/semantics/[name]` | StudioTabs + VisualMetrics editor |
| `/studio/publish` | StudioTabs + publish table + StatusBadges |
| `/files` | Legacy IDE header, still works |

- [ ] **Step 4: Verify ThemeToggle**

Click ThemeToggle → switch to dark. Navigate between portal and studio. Dark class persists.

- [ ] **Step 5: Commit cleanup**

```bash
git rm web-local/src/features/studio/StudioSidebar.svelte
git commit -m "chore(ui): remove StudioSidebar, replaced by StudioTabs"
```

### Task 13: Frontend quality checks

- [ ] **Step 1: Type check**

```bash
npx svelte-check --threshold warning 2>&1 | tail -30
```

Fix any type errors.

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

| File | Action | Phase | Chunk |
|------|--------|-------|-------|
| `web-common/src/app.css` | Modify | design tokens + typography | 1 |
| `web-common/src/components/status-badge/StatusBadge.svelte` | Create | StatusBadge | 2 |
| `web-local/src/features/portal/PortalNav.svelte` | Modify | theme toggle + Studio auto-detect | 1, 2 |
| `web-local/src/features/studio/StudioTabs.svelte` | Create | Horizontal tabs | 2 |
| `web-local/src/features/studio/SectionHeader.svelte` | Create | Page header | 2 |
| `web-local/src/features/studio/StudioSidebar.svelte` | Delete | cleanup | 5 |
| `web-local/src/routes/studio/+layout.svelte` | Rewrite | new layout shell | 3 |
| `web-local/src/routes/studio/+page.svelte` | Modify | SectionHeader + card-basic | 3 |
| `web-local/src/routes/studio/sources/+page.svelte` | Modify | SectionHeader + StatusBadge | 4 |
| `web-local/src/routes/studio/semantics/+page.svelte` | Modify | SectionHeader + StatusBadge | 4 |
| `web-local/src/routes/studio/semantics/[name]/+page.svelte` | Modify | SectionHeader | 4 |
| `web-local/src/routes/studio/publish/+page.svelte` | Modify + dedupe import | 4 |
| `web-local/src/routes/(portal)/+page.svelte` | Modify | Lucide icons in PortalHome | 5 |
| `web-local/src/routes/(portal)/boards/+page.svelte` | Modify | Lucide icons in Boards | 5 |

Total files changed: **14** (12 modifications, 3 creations, 1 deletion)
