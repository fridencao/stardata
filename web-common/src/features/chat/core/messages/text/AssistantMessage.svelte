<!-- Renders assistant responses from router_agent. -->
<script lang="ts">
  import { enhanceCitationLinks } from "@rilldata/web-common/features/chat/core/messages/text/enhance-citation-links.ts";
  import { eventBus } from "@rilldata/web-common/lib/event-bus/event-bus.ts";
  import Markdown from "../../../../../components/markdown/Markdown.svelte";
  import type { Conversation } from "../../conversation";
  import FeedbackButtons from "../../feedback/FeedbackButtons.svelte";
  import { extractFollowUps, extractMessageText } from "../../utils";
  import type { TextBlock } from "./text-block";

  export let block: TextBlock;
  export let conversation: Conversation;
  export let onDownvote: (messageId: string) => void;

  $: message = block.message;
  $: messageId = message.id ?? "";

  // Safety net: strip wrapper if the LLM wraps the entire response in ```markdown fences
  $: content = extractMessageText(message).replace(
    /^```markdown\n([\s\S]*)\n```$/,
    "$1",
  );

  // Follow-up suggestion questions from the structured analyst answer.
  $: followUps = extractFollowUps(message);

  function askFollowUp(question: string) {
    eventBus.emit("start-chat", question);
  }
</script>

<div class="chat-message">
  <div class="chat-message-content" use:enhanceCitationLinks={conversation}>
    <Markdown {content} />
  </div>
  {#if followUps.length > 0}
    <div class="chat-follow-ups">
      {#each followUps as question}
        <button
          type="button"
          class="follow-up-chip"
          title={question}
          onclick={() => askFollowUp(question)}
        >
          {question}
        </button>
      {/each}
    </div>
  {/if}
  <div class="chat-message-actions">
    <FeedbackButtons
      {messageId}
      {conversation}
      feedback={block.feedback}
      {onDownvote}
    />
  </div>
</div>

<style lang="postcss">
  .chat-message {
    @apply max-w-full;
  }

  .chat-message-content {
    @apply py-2;
    @apply text-sm leading-relaxed break-words;
    @apply text-fg-primary;
  }

  .chat-follow-ups {
    @apply flex flex-wrap gap-2 py-1;
  }

  .follow-up-chip {
    @apply text-xs px-3 py-1.5 rounded-full border bg-input text-fg-muted;
    @apply transition-colors;
    cursor: pointer;
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .follow-up-chip:hover {
    @apply text-fg-primary border-ring-focus;
  }

  .chat-message-actions {
    @apply pb-2;
  }
</style>
