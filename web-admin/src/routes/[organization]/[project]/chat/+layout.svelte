<script lang="ts">
  import { page } from "$app/stores";
  import PortalFullPageChat from "@rilldata/web-common/features/portal/chat/PortalFullPageChat.svelte";

  $: ({
    params: { organization, project },
  } = $page);
</script>

<!-- The portal chrome (PortalNav + PortalTabs) is rendered by the project layout. -->
<PortalFullPageChat
  basePath={`/${organization}/${project}/chat`}
  chatRouteId="/[organization]/[project]/chat"
/>
<!-- Keep slot to satisfy SvelteKit nested layout contract without rendering extra DOM. -->
<slot />
