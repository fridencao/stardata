import {
  getLastConversationId,
  setLastConversationId,
} from "@rilldata/web-common/features/chat/layouts/fullpage/fullpage-store";
import { getFeatureFlags } from "@rilldata/web-common/features/feature-flags.js";
import { getCloudRuntimeClient } from "@rilldata/web-admin/lib/runtime-client";
import { redirect } from "@sveltejs/kit";

export const load = async ({
  params: { organization, project, conversationId },
  route,
  url,
  parent,
}) => {
  const { runtime, user } = await parent();
  const client = getCloudRuntimeClient(runtime);
  // Scope the persisted conversation ID by user so switching accounts in the
  // same tab never replays another user's conversation (runtime rejects it
  // with "action not allowed").
  const userId = user?.id;

  const fetchedFeatureFlags = await getFeatureFlags(client);

  // Redirect to the portal home if the chat feature is disabled
  const chatEnabled = Boolean(fetchedFeatureFlags.chat);
  if (!chatEnabled) {
    throw redirect(307, `/${organization}/${project}`);
  }

  switch (route.id) {
    case "/[organization]/[project]/chat": {
      // If user explicitly wants a new conversation, clear stored ID and skip redirect logic
      const isExplicitNewConversation = url.searchParams.get("new") === "true";
      if (isExplicitNewConversation) {
        setLastConversationId(organization, project, null, userId);
        return;
      }

      // Try to redirect to the last conversation
      const lastConversationId = getLastConversationId(
        organization,
        project,
        userId,
      );
      if (lastConversationId) {
        throw redirect(
          307,
          `/${organization}/${project}/chat/${lastConversationId}`,
        );
      }

      // No existing conversation found, show new conversation interface
      return;
    }

    case "/[organization]/[project]/chat/[conversationId]": {
      // If conversation ID is missing or empty, redirect to base chat
      if (!conversationId?.trim()) {
        throw redirect(307, `/${organization}/${project}/chat`);
      }

      // Store this conversation ID as the last accessed conversation
      setLastConversationId(organization, project, conversationId, userId);

      // Go to the conversation
      return;
    }

    default: {
      // Allow unknown routes to pass through
      return;
    }
  }
};
