# StarData Frontend UI Design Unification

**Date:** 2026-07-27
**Status:** Proposed
**Branch:** feature/dual-portal-m2

## Background

After migrating from Rill to StarData, the frontend has accumulated three different visual languages:
- **Portal** (home/chat/boards): light top nav (`PortalNav`), `bg-gray-50`, large rounded cards
- **Studio** (/studio/*): dark sidebar (`StudioSidebar` `#191B26`), hardcoded colors, compact tables
- **Legacy**: `ApplicationHeader` from Rill, hidden behind route checks but still in codebase

This creates a jarring user experience — switching between portal and studio feels like switching products. The goal is a unified enterprise BI dashboard aesthetic.

## Design Principles

1. **One nav shell**: PortalNav is the global navigation. Studio uses the same top bar.
2. **Design tokens via CSS variables**: All semantic values flow through `app.css` custom properties.
3. **Enterprise BI style**: light backgrounds, clear information hierarchy, consistent card/table language.
4. **Dark mode support**: User-toggleable theme stored in localStorage, applied to `html` element's class.

## Decisions Summary

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Studio layout | Top sub-nav tabs (A) | PortalNav already exists; role switcher gives access to Studio |
| Title area | `SectionHeader` component | 6 pages duplicate h2/p pattern |
| Tab indicator style | 52px with underline | Matches PortalNav height, clear active state |
| Theme switching | Manual toggle button in PortalNav | Business users prefer stable theme; tech users want flexibility |
| Token management | Predefined Tailwind utility classes | Easiest migration path: search-replace `.card-primary` etc. |
| Dark mode scope | Both Portal and Studio share one theme | Single `html.dark` toggle affects all routes |

## Visual Changes

### Navigation
```
Before: PortalNav (white topbar) <--> StudioSidebar (dark left panel) [Jarring]
After:  PortalNav + StudioTabs (all white, horizontal, unified) [Smooth]
```

**PortalNav changes:**
- Remove manual "技术工作台" button — replace with automatic highlight when on Studio routes
- Add `ThemeToggle` button next to role switcher (☀ / 🌙)

**New StudioTabs component:**
- Horizontal tabs matching PortalNav style (52px height, underline indicator)
- Items: 概览, 数据源, 语义层, 发布
- Active tab: primary color underline + bold text
- Inherits from `$previewModeStore` — only renders when `isStudioRoute`

### Color System

All pages will use these predefined utility classes instead of raw Tailwind colors:

```css
@layer utilities {
  /* Surface layers */
  .bg-app-surface    @apply bg-surface-subtle;   /* page background */
  .bg-app-card       @apply bg-surface-card;      /* card/dialog background */
  .bg-app-elevated   @apply bg-white;              /* modal/overlay card */

  /* Text */
  .text-app-primary   @apply text-fg-primary;
  .text-app-secondary @apply text-fg-secondary;
  .text-app-muted     @apply text-fg-tertiary;

  /* Borders */
  .border-app        @apply border-border;
  .border-app-strong @apply border-gray-300 dark:border-gray-600;

  /* Card patterns */
  .card-basic        @apply rounded-xl border-app bg-app-card shadow-sm hover:shadow-md transition-shadow;
  .card-hero         @apply rounded-2xl border-app bg-app-card shadow-sm hover:shadow-md transition-shadow;
}
```

### Spacing

All Studio pages go through a single `<main>` container:

```svelte
<main class="flex-1 overflow-y-auto p-8 xl:max-w-6xl xl:mx-auto">
```

Removes `px-9 py-7` / `pt-16 pb-20` scattered across individual pages.

| Context | Horizontal | Vertical | Max width |
|---------|-----------|----------|-----------|
| Portal pages | `p-8` | `p-8` | 1100px (home), none (chat/boards detail) |
| Studio pages | `p-8` | `p-8` | 1100px (`xl:max-w-6xl`) |
| Chat fullscreen | unchanged | unchanged | — |

### Component Library Additions

#### `SectionHeader.svelte`
```svelte
<script>
  export let title: string;
  export let description?: string = undefined;
</script>

<div>
  <h2 class="text-lg font-bold text-app-primary">{title}</h2>
  {#if description}
    <p class="mt-0.5 text-[13px] text-app-muted">{description}</p>
  {/if}
  <slot name="actions" />
</div>
```

Usage replaces 6 instances of duplicated `h2 + p`:
```svelte
<!-- Before -->
<h2 class="text-lg font-bold text-gray-900">概览 · 配置健康度</h2>
<p class="mt-0.5 text-[13px] text-gray-400">一屏了解:业务现在能问什么,还缺什么</p>

<!-- After -->
<SectionHeader title="概览" description="一屏了解:业务现在能问什么,还缺什么">
  <!-- optional slot for action buttons -->
</SectionHeader>
```

#### `StatusBadge.svelte`
Replaces 3+ inline badge implementations:
```svelte
<StatusBadge variant="success">有效</StatusBadge>
<StatusBadge variant="error">解析错误</StatusBadge>
<StatusBadge variant="neutral">未发布</StatusBadge>
<StatusBadge variant="info" size="sm">OLAP</StatusBadge>
```

## Files Changed

| File | Action | Description |
|------|--------|-------------|
| `web-local/src/features/studio/StudioSidebar.svelte` | DELETE | Replaced by StudioTabs |
| `web-local/src/features/studio/StudioTabs.svelte` | CREATE | Horizontal tabs for Studio nav |
| `web-local/src/features/studio/SectionHeader.svelte` | CREATE | Unified page header component |
| `web-local/src/features/studio/RequestsTodo.svelte` | no change | Already used by publish page |
| `web-local/src/routes/studio/+layout.svelte` | MODIFY | Dark sidebar → PortalNav + StudioTabs + main container |
| `web-local/src/routes/studio/+page.svelte` | MODIFY | Use SectionHeader, unified card styles |
| `web-local/src/routes/studio/sources/+page.svelte` | MODIFY | Use SectionHeader, card-basic on connector cards |
| `web-local/src/routes/studio/semantics/+page.svelte` | MODIFY | Use SectionHeader |
| `web-local/src/routes/studio/semantics/[name]/+page.svelte` | MODIFY | Use SectionHeader, styled back link |
| `web-local/src/routes/studio/publish/+page.svelte` | MODIFY | Use SectionHeader |
| `web-local/src/features/portal/PortalNav.svelte` | MODIFY | Add ThemeToggle, auto-detect Studio route, remove manual button |
| `web-common/src/app.css` | MODIFY | Add `.bg-app-*` / `.card-*` utilities |
| `web-common/tailwind.config.ts` | MODIFY | Register custom utility classes |
| `web-local/tailwind.config.ts` | MODIFY | Register custom utility classes |

## Migration Strategy

Phase 1 — Foundation:
1. Add design token utility classes to Tailwind config + app.css
2. Create `StudioTabs.svelte` and `SectionHeader.svelte` components
3. Update `PortalNav.svelte` with theme toggle and Studio auto-highlight

Phase 2 — Studio layout:
4. Rewrite `studio/+layout.svelte` (replace sidebar with Tabs + main container)
5. Update `studio/+page.svelte` (overview) with new components

Phase 3 — Studio pages:
6. Update `sources/+page.svelte`
7. Update `semantics/+page.svelte`
8. Update `semantics/[name]/+page.svelte`
9. Update `publish/+page.svelte`

Phase 4 — Cleanup:
10. Delete `StudioSidebar.svelte`
11. Run full test suite + Playwright e2e on all routes
12. Verify dark mode works on all pages

## Dark Mode Implementation

```javascript
// In PortalNav, add theme toggle
let isDark = localStorage.getItem('theme') === 'dark';

$effect(() => {
  if ($isDark) {
    document.documentElement.classList.add('dark');
  } else {
    document.documentElement.classList.remove('dark');
  }
  localStorage.setItem('theme', $isDark ? 'dark' : 'light');
});
```

Key considerations for dark mode:
- The `app.css` already defines `:root.dark` with all semantic variable overrides
- StudioSidebar's hardcoded `#191B26` won't work in dark mode — removal is necessary
- All newly added utility classes must account for `dark:` variants
- Chart colors (vega-rendered) need theming — this is a known limitation of current dark mode
