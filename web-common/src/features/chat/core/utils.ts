/**
 * Shared utilities for chat functionality
 *
 * Common functions used across ConversationManager and Conversation classes to avoid duplication
 * and maintain consistency in ID generation and message content extraction.
 */
import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient.ts";
import {
  getRuntimeServiceGetConversationQueryOptions,
  getRuntimeServiceListConversationsQueryKey,
  getRuntimeServiceListConversationsQueryOptions,
  type V1Message,
} from "@rilldata/web-common/runtime-client";
import { MessageContentType, ToolName } from "./types";
import type { RuntimeClient } from "@rilldata/web-common/runtime-client/v2";
import { derived } from "svelte/store";
import { createQuery } from "@tanstack/svelte-query";

// =============================================================================
// ID GENERATION
// =============================================================================

export const NEW_CONVERSATION_ID = "new";

const OPTIMISTIC_MESSAGE_ID_PREFIX = "optimistic-message-";

export function getOptimisticMessageId(): string {
  return `${OPTIMISTIC_MESSAGE_ID_PREFIX}${Date.now()}`;
}

// =============================================================================
// MESSAGE CONTENT EXTRACTION
// =============================================================================

/**
 * Parsed form of the analyst agent's structured JSON answer.
 * The analyst stores its final answer as a JSON document inside the
 * router_agent result's `response` string field.
 */
interface StructuredAnswer {
  summary?: string;
  body?: string;
  insights?: string[];
  follow_ups?: string[];
}

/**
 * Attempts to parse the router_agent result as a structured analyst answer.
 * Returns null if the message is not a router_agent JSON result, or if the
 * `response` field is not a structured JSON document (e.g. plain text).
 */
function parseStructuredAnswer(message: V1Message): StructuredAnswer | null {
  if (
    message.contentType !== MessageContentType.JSON ||
    message.tool !== ToolName.ROUTER_AGENT
  ) {
    return null;
  }
  try {
    const parsed = JSON.parse(message.contentData || "");
    const resp = parsed?.response;
    if (typeof resp !== "string") return null;
    const structured = JSON.parse(resp);
    if (structured && typeof structured === "object") {
      return structured as StructuredAnswer;
    }
  } catch {
    // response is plain text or not JSON — not a structured answer
  }
  return null;
}

/**
 * Extract text content from a message based on content type
 *
 * Handles all three content types (text, json, error) with special parsing
 * for router_agent JSON messages to extract the `body` field from the
 * structured analyst answer (falling back to the raw response for
 * non-structured / legacy answers).
 */
export function extractMessageText(message: V1Message): string {
  const rawContent = message.contentData || "";

  switch (message.contentType) {
    case MessageContentType.JSON:
      // For router_agent, extract the structured answer's `body` field.
      if (message.tool === ToolName.ROUTER_AGENT) {
        const structured = parseStructuredAnswer(message);
        if (structured) {
          // Structured answer — never leak raw JSON to the user.
          return structured.body || structured.summary || "";
        }
        // Non-structured (legacy / plain-text) response.
        try {
          const parsed = JSON.parse(rawContent);
          return parsed.prompt || parsed.response || rawContent;
        } catch {
          return rawContent;
        }
      }

      // For non-router_agent JSON messages, return raw content
      return rawContent;

    case MessageContentType.TEXT:
      return rawContent;

    case MessageContentType.ERROR:
      return rawContent;

    default:
      return rawContent;
  }
}

/**
 * Extract follow-up suggestion questions from a structured analyst answer.
 * Returns an empty array if the message has no follow-ups.
 */
export function extractFollowUps(message: V1Message): string[] {
  const structured = parseStructuredAnswer(message);
  if (!Array.isArray(structured?.follow_ups)) return [];
  return structured.follow_ups.filter(
    (q): q is string => typeof q === "string" && q.trim() !== "",
  );
}

export function invalidateConversationsList(instanceId: string) {
  const listConversationsKey = getRuntimeServiceListConversationsQueryKey(
    instanceId,
    {
      userAgentPattern: "rill%",
    },
  );
  return queryClient.invalidateQueries({ queryKey: listConversationsKey });
}

/**
 * Returns the last updated conversation ID.
 */
export function getLatestConversationQueryOptions(client: RuntimeClient) {
  const listConversationsQueryOptions =
    getRuntimeServiceListConversationsQueryOptions(client, {
      // Filter to only show Rill client conversations, excluding MCP conversations
      userAgentPattern: "rill%",
    });
  const lastConversationId = derived(
    createQuery(listConversationsQueryOptions, queryClient),
    (conversationsResp) => {
      const conversations = conversationsResp?.data?.conversations?.filter(
        (c) => c.userAgent !== "rill/report",
      );
      return conversations?.[0]?.id;
    },
  );

  return derived([lastConversationId], ([id]) => {
    return getRuntimeServiceGetConversationQueryOptions(
      client,
      { conversationId: id ?? "" },
      {
        query: {
          enabled: !!id,
        },
      },
    );
  });
}
