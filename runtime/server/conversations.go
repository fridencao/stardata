package server

import (
	"encoding/json"
	"net/http"

	"github.com/fridencao/stardata/runtime"
	"github.com/fridencao/stardata/runtime/server/auth"
)

// DeleteConversationHandler deletes a conversation (AI session) and all its messages.
// DELETE /v1/instances/{instance_id}/ai/conversations/{conversation_id}
// This is a raw HTTP handler (bypasses gRPC/proto) wrapped with auth middleware.
func (s *Server) DeleteConversationHandler(w http.ResponseWriter, r *http.Request) {
	instanceID := r.PathValue("instance_id")
	conversationID := r.PathValue("conversation_id")

	// Auth check: require UseAI permission.
	claims := auth.GetClaims(r.Context(), instanceID)
	if !claims.Can(runtime.UseAI) {
		writeJSONError(w, http.StatusForbidden, "access denied")
		return
	}

	if conversationID == "" {
		writeJSONError(w, http.StatusBadRequest, "conversation_id is required")
		return
	}

	// Open the instance catalog and delete the session + its messages.
	catalog, release, err := s.runtime.Catalog(r.Context(), instanceID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to open catalog: "+err.Error())
		return
	}
	defer release()

	if err := catalog.DeleteAISession(r.Context(), conversationID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to delete conversation: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}
