package server

import (
	"testing"

	"github.com/fridencao/stardata/admin"
	"github.com/fridencao/stardata/admin/database"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGenerateSignedDownloadURLNotConfigured(t *testing.T) {
	// Assets is nil (no GCS bucket configured in private deployments).
	s := &Server{admin: &admin.Service{}}

	_, err := s.generateSignedDownloadURL(&database.Asset{Path: "gs://bucket/path"})
	if status.Code(err) != codes.Unimplemented {
		t.Fatalf("expected codes.Unimplemented when assets storage is not configured, got %v", err)
	}
}
