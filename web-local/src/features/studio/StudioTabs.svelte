<script lang="ts">
  import { page } from "$app/stores";
  import { LayoutDashboard, Database, Network, Rocket, Code2 } from "lucide-svelte";
  import { isIdeRoute } from "../../routes/route-constants";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  const tabs = [
    { label: m.studio_tabs_overview(), href: "/studio", icon: LayoutDashboard },
    { label: m.studio_tabs_sources(), href: "/studio/sources", icon: Database },
    { label: m.studio_tabs_semantics(), href: "/studio/semantics", icon: Network },
    { label: m.studio_tabs_publish(), href: "/studio/publish", icon: Rocket },
    { label: m.studio_tabs_ide(), href: "/files", icon: Code2 },
  ];

  $: pathname = $page.url.pathname;

  function isActive(href: string, path: string): boolean {
    if (href === "/studio") return path === "/studio";
    if (href === "/files") return isIdeRoute(path);
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
