<script lang="ts">
  import type { Readable } from "svelte/store";
  import Button from "../../../../components/button/Button.svelte";
  import * as Tooltip from "../../../../components/tooltip-v2";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
  import type { ChatActions } from "./sidebar-store";

  export let open: Readable<boolean>;
  export let actions: ChatActions;

  const isMac = window.navigator.userAgent.includes("Macintosh");
</script>

<svelte:window
  onkeydown={(e) => {
    if (e[isMac ? "metaKey" : "ctrlKey"] && e.key === "j") {
      e.preventDefault();
      actions.toggleChat();
    }
  }}
/>

<Tooltip.Root>
  <Tooltip.Trigger>
    {#snippet child({ props })}
      <Button
        {...props}
        compact
        type="secondary"
        onClick={actions.toggleChat}
        active={$open}
      >
        AI
      </Button>
    {/snippet}
  </Tooltip.Trigger>
  <Tooltip.Content side="bottom">
    {m.chat_toggle_tooltip({ shortcut: isMac ? "⌘" : "Ctrl" })}
  </Tooltip.Content>
</Tooltip.Root>
