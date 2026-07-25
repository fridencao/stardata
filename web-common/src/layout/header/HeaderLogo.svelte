<script lang="ts">
  import StarDataLogo from "@rilldata/web-common/components/icons/StarDataLogo.svelte";

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
    <StarDataLogo />
  {/if}
</a>
