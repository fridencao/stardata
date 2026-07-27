<!-- Renders assistant responses from router_agent. -->
<script lang="ts">
  import { enhanceCitationLinks } from "@rilldata/web-common/features/chat/core/messages/text/enhance-citation-links.ts";
  import { eventBus } from "@rilldata/web-common/lib/event-bus/event-bus.ts";
  import { page } from "$app/stores";
  import type { V1Message } from "@rilldata/web-common/runtime-client";
  import Markdown from "../../../../../components/markdown/Markdown.svelte";
  import type { Conversation } from "../../conversation";
  import FeedbackButtons from "../../feedback/FeedbackButtons.svelte";
  import RequestDialog from "../../../requests/RequestDialog.svelte";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
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

  // Pre-fill: most recent user question before this AI message
  let requestOpen = false;
  $: convQuery = conversation.getConversationQuery();
  $: defaultQuestion = sanitize(
    findPrecedingUserQuestion(
      $convQuery.data?.messages ?? [],
      messageId,
    ),
  );

  // Only show in local (web-local), not cloud (organization param present)
  $: canRequest = !$page.params.organization;

  function findPrecedingUserQuestion(
    messages: V1Message[],
    assistantId: string,
  ): string {
    let idx = messages.findIndex((m) => m.id === assistantId);
    if (idx < 0) idx = messages.length;
    for (let i = idx - 1; i >= 0; i--) {
      if (messages[i].role === "user") return extractMessageText(messages[i]);
    }
    return "";
  }

  // Strip HTML tags — belt-and-suspenders defense-in-depth. The value goes into a
  // Svelte textarea (auto-escaped), but stripping removes the risk if any future
  // consumer renders it unsafely. Also keeps follow-up-chip `title` attribute clean.
  function sanitize(text: string): string {
    return text.replace(/<[^>]*>/g, "");
  }

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
  <div class="chat-message-actions flex items-center gap-2">
    <FeedbackButtons
      {messageId}
      {conversation}
      feedback={block.feedback}
      {onDownvote}
    />
    {#if canRequest}
      <button
        type="button"
        class="follow-up-chip"
        onclick={() => (requestOpen = true)}
      >
        {m.chat_request_cta()}
      </button>
      <RequestDialog bind:open={requestOpen} {defaultQuestion} />
    {/if}
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
