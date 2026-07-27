<script lang="ts">
  export let href: string = "/";
  export let logoUrl: string | undefined = undefined;
</script>

<a
  {href}
  class="grid place-content-center rounded {logoUrl ? 'pl-2 pr-2' : 'p-2'}"
>
  {#if logoUrl}
    <img src={logoUrl} alt="logo" class="h-7" />
  {:else}
    <!-- StarData brand mark, consistent with PortalNav -->
    <span class="flex items-center gap-2 text-base font-bold text-gray-900">
      <span
        class="grid size-[26px] place-items-center rounded-lg bg-primary-600 text-sm text-white"
      >
        ✦
      </span>
      StarData
    </span>
  {/if}
</a>
