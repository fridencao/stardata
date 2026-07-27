<script lang="ts">
  import PortalFullPageChat from "../../../features/chat/PortalFullPageChat.svelte";
</script>

<!-- Reuse the PortalNav + page shell provided by the parent (portal)/+layout.svelte. -->
<!-- This layout only supplies the chat content so the top navigation is not duplicated. -->
<PortalFullPageChat />
<!-- Keep slot to satisfy SvelteKit nested layout contract without rendering extra DOM. -->
<slot />
