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
        <span class="absolute bottom-0 left-0 right-0 h-[2px] bg-primary-600"></span>
      {/if}
    </a>
  {/each}
</div>
