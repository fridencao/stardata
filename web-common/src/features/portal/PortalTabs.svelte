<script lang="ts">
  import { page } from "$app/stores";
  import { Home, MessageSquare, LayoutDashboard, FileText, Bell } from "lucide-svelte";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  /** 路由前缀(web-local 为 "";web-admin 为 "/[org]/[project]" 等) */
  export let basePath = "";
  /** 「我的报告」入口;null = 不显示(功能未启用或无权限) */
  export let reportsHref: string | null = null;
  /** 「我的订阅」入口;null = 不显示(功能未启用或无权限) */
  export let alertsHref: string | null = null;

  $: tabs = [
    { label: m.portal_tabs_home(), href: basePath || "/", exact: true, icon: Home },
    { label: m.portal_tabs_chat(), href: `${basePath}/chat`, exact: false, icon: MessageSquare },
    { label: m.portal_tabs_boards(), href: `${basePath}/boards`, exact: false, icon: LayoutDashboard },
    ...(reportsHref
      ? [{ label: m.nav_tab_reports(), href: reportsHref, exact: false, icon: FileText }]
      : []),
    ...(alertsHref
      ? [{ label: m.nav_tab_alerts(), href: alertsHref, exact: false, icon: Bell }]
      : []),
  ];

  $: pathname = $page.url.pathname;

  function isActive(href: string, exact: boolean, path: string): boolean {
    if (exact) return path === href || path === `${href}/`;
    return path.startsWith(href);
  }
</script>

<div class="h-[52px] flex items-center gap-1 border-b border-gray-200 bg-surface-background px-8">
  {#each tabs as tab (tab.href)}
    <a
      href={tab.href}
      class="relative flex items-center gap-2 px-3.5 py-2 text-sm no-underline transition-colors {isActive(tab.href, tab.exact, pathname)
          ? 'font-bold text-primary-700'
          : 'text-gray-600 hover:text-gray-900'}"
    >
      <svelte:component this={tab.icon} class="size-4" />
      {tab.label}
      {#if isActive(tab.href, tab.exact, pathname)}
        <span class="absolute bottom-0 left-0 right-0 h-[2px] bg-primary-600"></span>
      {/if}
    </a>
  {/each}
</div>
