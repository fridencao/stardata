<script lang="ts">
  import { page } from "$app/stores";
  import {
    LayoutDashboard,
    Database,
    Network,
    Rocket,
    Code2,
    Activity,
    Settings,
  } from "lucide-svelte";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  /** 路由前缀(web-local 为 "";web-admin 为 "/[org]/[project]/-/edit") */
  export let basePath = "";
  /** 高级模式(IDE)入口;web-local 为 "/files",web-admin 为 edit 工作区根 */
  export let ideHref = "/files";
  /** IDE Tab 的激活判定(IDE 路由结构两端不同,由调用方决定) */
  export let ideActive: (path: string) => boolean = (p) =>
    ["/files", "/connector/", "/graph"].some((prefix) => p.startsWith(prefix));
  /** 运行状态入口(StarData:收编自遗留控制台的 -/status);不传则不显示 */
  export let statusHref: string | undefined = undefined;
  /** 项目设置入口(StarData:收编自遗留控制台的 -/settings);不传则不显示 */
  export let settingsHref: string | undefined = undefined;

  $: tabs = [
    { label: m.studio_tabs_overview(), href: `${basePath}/studio`, icon: LayoutDashboard },
    { label: m.studio_tabs_sources(), href: `${basePath}/studio/sources`, icon: Database },
    { label: m.studio_tabs_semantics(), href: `${basePath}/studio/semantics`, icon: Network },
    { label: m.studio_tabs_publish(), href: `${basePath}/studio/publish`, icon: Rocket },
    ...(settingsHref
      ? [{ label: m.nav_tab_settings(), href: settingsHref, icon: Settings }]
      : []),
    { label: m.studio_tabs_ide(), href: ideHref, icon: Code2 },
    ...(statusHref
      ? [{ label: m.nav_tab_status(), href: statusHref, icon: Activity }]
      : []),
  ];

  $: pathname = $page.url.pathname;

  function isActive(href: string, path: string): boolean {
    if (href === `${basePath}/studio`) return path === href;
    if (href === ideHref) return ideActive(path);
    return path.startsWith(href);
  }
</script>

<div class="h-[52px] flex items-center gap-1 border-b border-gray-200 bg-surface-background px-8">
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
        <span class="absolute bottom-0 left-0 right-0 h-[2px] bg-primary-600"></span>
      {/if}
    </a>
  {/each}
</div>
