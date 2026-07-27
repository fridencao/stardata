<script lang="ts">
  import DeveloperChat from "@rilldata/web-common/features/chat/DeveloperChat.svelte";
  import Navigation from "@rilldata/web-common/layout/navigation/Navigation.svelte";
  import { editorRoutePrefix } from "@rilldata/web-common/layout/navigation/editor-routing";
  import PortalNav from "../../../features/portal/PortalNav.svelte";
  import StudioTabs from "../../../features/studio/StudioTabs.svelte";
  import type { LayoutData } from "../$types";

  export let data: LayoutData;

  // web-local uses top-level routes; no prefix needed
  $editorRoutePrefix = "";
</script>

<div class="flex size-full min-h-0 flex-col overflow-hidden">
  <PortalNav />
  <StudioTabs />

  <div class="flex w-full min-h-0 flex-1 overflow-hidden">
    {#if data.initialized}
      <Navigation showFooterLinks={false} />
    {/if}

    <section class="flex size-full overflow-hidden">
      <div class="flex-1 overflow-hidden">
        <slot />
      </div>
      <DeveloperChat />
    </section>
  </div>
</div>
