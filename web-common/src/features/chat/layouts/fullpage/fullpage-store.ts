import { localStorageStore } from "../../../../lib/store-utils/local-storage";

// =============================================================================
// SIDEBAR STATE
// =============================================================================

// Whether the conversation sidebar is collapsed (icon-only mode)
export const conversationSidebarCollapsed = localStorageStore<boolean>(
  "conversation-sidebar-collapsed",
  false, // default to expanded
);

export function toggleConversationSidebar() {
  conversationSidebarCollapsed.update((collapsed) => !collapsed);
}

// =============================================================================
// CONVERSATION ID PERSISTENCE
// =============================================================================

function getConversationIdStorageKey(
  organization: string,
  project: string,
  userId?: string,
) {
  // Scope by user when known so a conversation ID left behind by a previous
  // login in the same tab is never replayed for a different user (the runtime
  // rejects foreign conversations with "action not allowed").
  const userSuffix = userId ? `-${userId}` : "";
  return `project-chat-conversation-id-${organization}-${project}${userSuffix}`;
}

/**
 * Retrieves the last conversation ID for the given project from sessionStorage.
 * Handles both JSON-stringified and raw string formats for backwards compatibility.
 */
export function getLastConversationId(
  organization: string,
  project: string,
  userId?: string,
): string | null {
  const storageKey = getConversationIdStorageKey(organization, project, userId);
  const storedValue = sessionStorage.getItem(storageKey);

  if (!storedValue) {
    return null;
  }

  try {
    // Try to parse as JSON first (new format)
    const parsed = JSON.parse(storedValue);
    return parsed === "null" ? null : parsed;
  } catch {
    // Fall back to raw string value (legacy format)
    return storedValue === "null" ? null : storedValue;
  }
}

/**
 * Stores the conversation ID for the given project in sessionStorage.
 * Uses JSON serialization for consistency with the getter function.
 */
export function setLastConversationId(
  organization: string,
  project: string,
  conversationId: string | null,
  userId?: string,
): void {
  const storageKey = getConversationIdStorageKey(organization, project, userId);

  if (conversationId === null) {
    sessionStorage.removeItem(storageKey);
  } else {
    sessionStorage.setItem(storageKey, JSON.stringify(conversationId));
  }
}
