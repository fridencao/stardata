<script lang="ts">
  import { page } from "$app/stores";
  import { portalRole } from "./portal-role-store";

  const links = [
    { label: "首页", href: "/" },
    { label: "对话", href: "/chat" },
    { label: "看板", href: "/boards" },
  ];

  $: pathname = $page.url.pathname;

  function isActive(href: string, path: string) {
    return href === "/" ? path === "/" : path.startsWith(href);
  }
</script>

<nav
  class="flex h-[60px] items-center gap-7 border-b border-gray-200 bg-white/90 px-9 backdrop-blur"
>
  <a href="/" class="flex items-center gap-2 text-base font-bold text-gray-900">
    <span
      class="grid size-[26px] place-items-center rounded-lg bg-primary-600 text-sm text-white"
    >
      ✦
    </span>
    StarData
  </a>
  <div class="flex gap-1">
    {#each links as link (link.href)}
      <a
        href={link.href}
        class="rounded-lg px-3.5 py-1.5 text-sm no-underline {isActive(
          link.href,
          pathname,
        )
          ? 'bg-primary-50 font-semibold text-primary-700'
          : 'text-gray-600 hover:bg-gray-100'}"
      >
        {link.label}
      </a>
    {/each}
  </div>
  <div class="ml-auto flex items-center gap-3">
    <!-- 演示级角色切换器 -->
    <div class="flex rounded-lg bg-gray-100 p-0.5 text-xs">
      <button
        class="rounded-md px-2.5 py-1 {$portalRole === 'business'
          ? 'bg-white font-semibold shadow-sm'
          : 'text-gray-500'}"
        on:click={() => portalRole.set("business")}
      >
        业务视角
      </button>
      <button
        class="rounded-md px-2.5 py-1 {$portalRole === 'tech'
          ? 'bg-white font-semibold shadow-sm'
          : 'text-gray-500'}"
        on:click={() => portalRole.set("tech")}
      >
        技术视角
      </button>
    </div>
    {#if $portalRole === "tech"}
      <a
        href="/studio"
        class="flex items-center gap-1.5 rounded-lg border border-gray-200 bg-white px-3 py-1.5 text-[13px] text-gray-600 no-underline hover:border-primary-300"
      >
        🔧 技术工作台
      </a>
    {/if}
  </div>
</nav>
