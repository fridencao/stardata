<script lang="ts">
  import { page } from "$app/stores";

  const items = [
    { label: "概览", icon: "📊", href: "/studio" },
    { label: "数据源", icon: "🗄️", href: "/studio/sources" },
    { label: "语义层", icon: "📐", href: "/studio/semantics" },
    { label: "发布", icon: "🚀", href: "/studio/publish" },
    { label: "高级(IDE)", icon: "🧰", href: "/files" },
  ];

  $: pathname = $page.url.pathname;

  function isActive(href: string, path: string) {
    return href === "/studio" ? path === "/studio" : path.startsWith(href);
  }
</script>

<aside class="flex w-[216px] flex-col bg-[#191B26] p-3 text-[#B7BAC9]">
  <a
    href="/studio"
    class="flex items-center gap-2 px-2 pb-4 pt-1.5 font-bold text-white no-underline"
  >
    <span
      class="grid size-6 place-items-center rounded-lg bg-primary-600 text-[13px]"
    >
      ✦
    </span>
    StarData
    <span
      class="rounded-md bg-[#343757] px-1.5 text-[10.5px] font-semibold text-[#ADB2E8]"
    >
      STUDIO
    </span>
  </a>
  {#each items as item (item.href)}
    <a
      href={item.href}
      class="mb-0.5 flex items-center gap-2.5 rounded-lg px-3 py-2 text-[13.5px] no-underline {isActive(
        item.href,
        pathname,
      )
        ? 'bg-[#2B2E44] font-semibold text-white'
        : 'hover:bg-[#22242F]'}"
    >
      <span>{item.icon}</span>{item.label}
    </a>
  {/each}
  <div class="mt-auto border-t border-[#2B2E3E] pt-3">
    <a
      href="/"
      class="flex w-full items-center justify-center gap-1.5 rounded-lg bg-primary-600 py-2 text-[13px] font-semibold text-white no-underline hover:bg-primary-700"
    >
      👁 业务门户预览
    </a>
  </div>
</aside>
