<script lang="ts">
  import { beforeNavigate } from "$app/navigation";
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";
  import { onMount, tick } from "svelte";
  import { get } from "svelte/store";
  import { writable } from "svelte/store";
  import { useRuntimeClient } from "@rilldata/web-common/runtime-client/v2";
  import {
    getConversationManager,
    cleanupConversationManager,
  } from "@rilldata/web-common/features/chat/core/conversation-manager";
  import Messages from "@rilldata/web-common/features/chat/core/messages/Messages.svelte";
  import ChatInput from "@rilldata/web-common/features/chat/core/input/ChatInput.svelte";
  import { projectChat } from "@rilldata/web-common/features/project/chat-context";
  import { eventBus } from "@rilldata/web-common/lib/event-bus/event-bus.ts";
  import { waitUntil } from "@rilldata/web-common/lib/waitUtils.ts";
  import { chatMounted } from "@rilldata/web-common/features/chat/layouts/sidebar/sidebar-store";
  import { getStardataToken } from "@rilldata/web-common/runtime-client/auth-token";
  import PlusIcon from "@rilldata/web-common/components/icons/PlusIcon.svelte";
  import Pin from "@rilldata/web-common/components/icons/Pin.svelte";
  import Trash from "@rilldata/web-common/components/icons/Trash.svelte";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  /** 会话页路由前缀(web-local 为 "/chat";web-admin 为 "/[org]/[project]/chat") */
  export let basePath = "/chat";
  /** SvelteKit route id 前缀,用于判断是否仍在 chat 上下文内 */
  export let chatRouteId = "/(portal)/chat";

  const runtimeClient = useRuntimeClient();

  $: conversationManager = getConversationManager(runtimeClient, {
    conversationState: "url",
    basePath: () => basePath,
  });

  let chatInputComponent: ChatInput;

  function onMessageSend() {
    chatInputComponent?.focusInput();
  }

  // Focus on mount after the component tree settles
  onMount(async () => {
    await tick();
    chatInputComponent?.focusInput();

    // StarData: 首页推荐问题经 /chat?new=true&q=... 跳入时自动发送
    const q = $page.url.searchParams.get("q");
    if (q) {
      const url = new URL($page.url);
      url.searchParams.delete("q");
      window.history.replaceState(window.history.state, "", url);
      await waitUntil(() => get(chatMounted));
      eventBus.emit("start-chat", q);
    }
  });

  // Clean up conversation manager resources when leaving the chat context entirely
  beforeNavigate(({ to }) => {
    const isChatRoute = to?.route?.id?.startsWith(chatRouteId);
    if (!isChatRoute) {
      cleanupConversationManager(runtimeClient.instanceId);
    }
  });

  // ----- Portal-styled conversation list (data from conversation-manager) -----
  $: currentConversation = conversationManager.getCurrentConversation();
  $: getConversationQuery = $currentConversation?.getConversationQuery();
  $: currentConversationDto = $getConversationQuery?.data?.conversation ?? null;

  $: listConversationsQuery = conversationManager.listConversationsQuery();
  $: conversations = ($listConversationsQuery.data?.conversations ?? []).filter(
    (c) => c.userAgent !== "rill/report",
  );

  function handleNewConversation() {
    conversationManager.enterNewConversationMode();
    chatInputComponent?.focusInput();
  }

  // ----- Pinned conversations (client-side, persisted per instance) -----
  const pinKey = `stardata.pinned.${runtimeClient.instanceId}`;
  function loadPinned(): string[] {
    try {
      const raw = localStorage.getItem(pinKey);
      return raw ? (JSON.parse(raw) as string[]) : [];
    } catch {
      return [];
    }
  }
  const pinnedIds = writable<string[]>(loadPinned());
  pinnedIds.subscribe((ids) => {
    try {
      localStorage.setItem(pinKey, JSON.stringify(ids));
    } catch {
      /* ignore */
    }
  });

  $: pinnedSet = new Set($pinnedIds);

  // Pinned first, otherwise keep original (stable) order.
  $: sortedConversations = [...conversations].sort((a, b) => {
    const pa = pinnedSet.has(a.id ?? "") ? 0 : 1;
    const pb = pinnedSet.has(b.id ?? "") ? 0 : 1;
    return pa - pb;
  });

  function togglePin(id: string | undefined) {
    if (!id) return;
    pinnedIds.update((ids) =>
      ids.includes(id) ? ids.filter((x) => x !== id) : [...ids, id],
    );
  }

  async function handleDelete(id: string | undefined) {
    if (!id) return;
    if (!confirm(m.portal_chat_confirm_delete())) return;
    // Prefer the runtime client's JWT (web-admin); fall back to the StarData
    // local token (web-local) when no JWT is attached.
    const token = runtimeClient.getJwt() ?? getStardataToken();
    try {
      const res = await fetch(
        `${runtimeClient.host}/v1/instances/${runtimeClient.instanceId}/ai/conversations/${id}`,
        {
          method: "DELETE",
          headers: token ? { Authorization: `Bearer ${token}` } : {},
        },
      );
      if (!res.ok) {
        const body = await res.text();
        alert(m.portal_chat_delete_failed({ error: body || String(res.status) }));
        return;
      }
    } catch (e) {
      alert(m.portal_chat_delete_failed({ error: (e as Error).message }));
      return;
    }

    // If we deleted the currently open conversation, go back to a fresh one.
    if (currentConversationDto?.id === id) {
      goto(`${basePath}?new=true`);
    }
    // Refresh the list (svelte-query refetch)
    await $listConversationsQuery.refetch();
  }
</script>

<div class="portal-chat">
  <!-- Portal-styled conversation sidebar -->
  <aside class="portal-sidebar">
    <div class="sidebar-header">
      <span class="sidebar-title">{m.portal_tabs_chat()}</span>
      <a
        class="new-conversation-btn"
        href={`${basePath}?new=true`}
        onclick={handleNewConversation}
      >
        <PlusIcon size="13px" />
        <span>{m.portal_chat_new_conversation()}</span>
      </a>
    </div>

    <div class="sidebar-list">
      {#if $listConversationsQuery.isLoading}
        <div class="sidebar-hint">{m.portal_chat_loading()}</div>
      {:else if $listConversationsQuery.isError}
        <div class="sidebar-hint sidebar-hint--error">{m.portal_chat_load_failed()}</div>
      {:else if sortedConversations.length}
        {#each sortedConversations as conversation (conversation.id)}
          {@const cid = conversation.id ?? ""}
          <div
            class="conversation-item"
            class:active={cid === currentConversationDto?.id}
            class:pinned={pinnedSet.has(cid)}
          >
            <a href={`${basePath}/${cid}`} class="conversation-link">
              {#if pinnedSet.has(cid)}
                <span class="pin-badge" title={m.portal_chat_pinned()}>
                  <Pin size="12px" />
                </span>
              {/if}
              <span class="conversation-title">{conversation.title || m.portal_chat_new_conversation()}</span>
            </a>

            <div class="conv-actions">
              <button
                type="button"
                class="conv-action"
                class:active={pinnedSet.has(cid)}
                title={pinnedSet.has(cid) ? m.portal_chat_unpin() : m.portal_chat_pin()}
                onclick={() => togglePin(cid)}
              >
                <Pin size="13px" />
              </button>
              <button
                type="button"
                class="conv-action conv-action--danger"
                title={m.portal_chat_delete_conversation()}
                onclick={() => handleDelete(cid)}
              >
                <Trash size="13px" />
              </button>
            </div>
          </div>
        {/each}
      {:else}
        <div class="sidebar-hint">{m.portal_chat_empty()}</div>
      {/if}
    </div>
  </aside>

  <!-- Main chat area: reuse Rill's Messages + ChatInput (typography preserved) -->
  <div class="portal-main">
    <div class="portal-messages">
      <Messages {conversationManager} layout="fullpage" config={projectChat} />
    </div>

    <div class="portal-input-section">
      <div class="portal-input-wrapper">
        <ChatInput
          {conversationManager}
          onSend={onMessageSend}
          bind:this={chatInputComponent}
          config={projectChat}
        />
      </div>
    </div>
  </div>
</div>

<style lang="postcss">
  /*
   * Portal-styled chat shell.
   * Recolors Rill's inner components (Messages / ChatInput) to the StarData
   * Portal palette by overriding the semantic CSS variables they consume.
   * Typography (font sizes) is intentionally left untouched — it is shared
   * globally via web-common/src/app.css (Rill's compact enterprise scale).
   */
  .portal-chat {
    /* Portal palette remap (consumed by Rill's Messages/ChatInput) */
    --surface-base: var(--color-gray-50);
    --surface: var(--color-gray-50);
    --surface-subtle: var(--color-gray-100);
    --surface-muted: var(--color-gray-100);
    --surface-hover: var(--color-gray-100);
    --surface-card: #ffffff;
    --surface-overlay: #ffffff;
    --border: var(--color-gray-200);
    --fg-primary: var(--color-gray-900);
    --fg-secondary: var(--color-gray-600);
    --fg-tertiary: var(--color-gray-500);
    --fg-muted: var(--color-gray-500);
    --accent-primary: var(--color-primary-600);
    --accent-primary-action: var(--color-primary-600);
    --icon-accent: var(--color-primary-600);

    display: flex;
    height: 100%;
    width: 100%;
    overflow: hidden;
    background: var(--color-gray-50);
  }

  /* ---------- Sidebar (Portal-styled, built here) ---------- */
  .portal-sidebar {
    flex-shrink: 0;
    width: 280px;
    display: flex;
    flex-direction: column;
    min-height: 0;
    overflow: hidden;
    background: #ffffff;
    border-right: 1px solid var(--color-gray-200);
  }

  .sidebar-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
    padding: 1rem 1rem 0.75rem;
  }

  .sidebar-title {
    font-size: 12px;
    font-weight: 600;
    color: var(--color-gray-900);
  }

  .new-conversation-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    border-radius: 9999px;
    border: 1px solid var(--color-gray-200);
    background: #ffffff;
    padding: 0.375rem 0.875rem;
    font-size: 12px;
    line-height: 1.4;
    color: var(--color-gray-600);
    text-decoration: none;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
    transition:
      border-color 150ms,
      color 150ms;
  }
  .new-conversation-btn:hover {
    border-color: var(--color-primary-300);
    color: var(--color-primary-700);
  }

  .sidebar-list {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding: 0.25rem 0.75rem 0.75rem;
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .conversation-item {
    position: relative;
    display: flex;
    align-items: center;
    width: 100%;
    padding: 0.625rem 0.875rem;
    border-radius: 0.75rem;
    border: 1px solid var(--color-gray-200);
    background: #ffffff;
    transition:
      border-color 150ms,
      background-color 150ms;
  }
  .conversation-item:hover {
    border-color: var(--color-primary-300);
  }
  .conversation-item.active {
    border-color: var(--color-primary-300);
    background: var(--color-primary-50);
  }
  .conversation-item.pinned {
    border-color: var(--color-primary-200);
  }

  .conversation-link {
    flex: 1;
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 0.375rem;
    color: var(--color-gray-700);
    text-decoration: none;
    overflow: hidden;
  }
  .conversation-item.active .conversation-link {
    color: var(--color-primary-700);
  }

  .pin-badge {
    flex-shrink: 0;
    display: inline-flex;
    color: var(--color-primary-600);
  }

  .conversation-title {
    font-size: 12px;
    line-height: 1.4;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  /* Hover actions (pin / delete) */
  .conv-actions {
    flex-shrink: 0;
    display: none;
    align-items: center;
    gap: 0.125rem;
    margin-left: 0.375rem;
  }
  .conversation-item:hover .conv-actions {
    display: flex;
  }
  .conv-action {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    border-radius: 0.5rem;
    border: none;
    background: transparent;
    color: var(--color-gray-400);
    cursor: pointer;
    transition:
      background-color 120ms,
      color 120ms;
  }
  .conv-action:hover {
    background: var(--color-gray-100);
    color: var(--color-gray-700);
  }
  .conv-action.active {
    color: var(--color-primary-600);
  }
  .conv-action--danger:hover {
    background: var(--color-red-50);
    color: var(--color-red-600);
  }

  .sidebar-hint {
    padding: 1rem 0.5rem;
    text-align: center;
    font-size: 12px;
    color: var(--color-gray-500);
  }
  .sidebar-hint--error {
    color: var(--color-red-500);
  }

  /* ---------- Main chat area ---------- */
  .portal-main {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    background: var(--color-gray-50);
  }

  .portal-messages {
    flex: 1;
    min-height: 0;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .portal-input-section {
    flex-shrink: 0;
    padding: 1rem;
    display: flex;
    justify-content: center;
    background: var(--color-gray-50);
  }

  .portal-input-wrapper {
    width: 100%;
    max-width: 680px;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  @media (max-width: 640px) {
    .portal-chat {
      flex-direction: column;
    }
    .portal-sidebar {
      width: 100%;
      height: 200px;
      border-right: none;
      border-bottom: 1px solid var(--color-gray-200);
    }
    .sidebar-list {
      flex-direction: row;
      overflow-x: auto;
    }
    .conversation-item {
      flex-shrink: 0;
      min-width: 150px;
    }
  }
</style>
