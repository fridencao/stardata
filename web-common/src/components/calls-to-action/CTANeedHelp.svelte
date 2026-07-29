<script lang="ts">
  import { EmbedStore } from "@rilldata/web-common/features/embeds/embed-store";
  import { COMMUNITY_URL } from "@rilldata/web-common/lib/stardata-links";

  export let leading: string | undefined = undefined;

  $: isEmbedded = EmbedStore.isEmbedded();
</script>

<!-- 私有化部署无公开社区：COMMUNITY_URL 为空时不渲染求助入口 -->
{#if !isEmbedded && COMMUNITY_URL}
  <p class="text-sm text-fg-secondary text-center">
    {#if leading}{leading}{" "}{/if}Need help? Reach out to us on
    <a href={COMMUNITY_URL} target="_blank">our support channel</a>
  </p>
{/if}
