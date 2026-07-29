package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/fridencao/stardata/admin/database"
	"github.com/fridencao/stardata/admin/server/auth"
	runtimev1 "github.com/fridencao/stardata/proto/gen/stardata/runtime/v1"
	"github.com/fridencao/stardata/runtime/pkg/archive"
	"github.com/fridencao/stardata/runtime/pkg/httputil"
	"github.com/google/uuid"
)

// maxPublishNoteLen limits the length of a publish note.
const maxPublishNoteLen = 500

// maxPublishHistory caps how many history entries are returned.
const maxPublishHistory = 200

// publishItem is the JSON representation of a publish history entry (StarData).
type publishItem struct {
	Version     int    `json:"version"`
	Note        string `json:"note,omitempty"`
	PublishedBy string `json:"published_by,omitempty"`
	CreatedAt   string `json:"created_at"`
	Current     bool   `json:"current"`
}

// publishesForOrgAndProject serves the publish model endpoint (StarData):
//   - GET:  list the publish history (release versions).
//   - POST: publish the current Studio (dev) draft as a new release and point production at it.
//
// Both require ManageProject (technical users). Publishing packages the dev deployment's
// current project files into a new archive asset and switches production to it.
func (s *Server) publishesForOrgAndProject(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	org := r.PathValue("org")
	project := r.PathValue("project")

	proj, err := s.admin.DB.FindProjectByName(ctx, org, project)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return httputil.Errorf(http.StatusNotFound, "project not found")
		}
		return httputil.Error(http.StatusInternalServerError, err)
	}

	claims := auth.GetClaims(ctx)
	perms := claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID)
	if !perms.ManageProject {
		return httputil.Errorf(http.StatusForbidden, "does not have permission to manage publishes")
	}

	switch r.Method {
	case http.MethodGet:
		items, err := s.listPublishes(ctx, proj)
		if err != nil {
			return err
		}
		return writeJSON(w, map[string]any{"publishes": items})
	case http.MethodPost:
		return s.publishProject(w, r, proj)
	default:
		return httputil.Errorf(http.StatusMethodNotAllowed, "method %s not allowed", r.Method)
	}
}

// rollbackForOrgAndProject re-deploys a previous release to production (StarData). Requires ManageProject.
func (s *Server) rollbackForOrgAndProject(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	org := r.PathValue("org")
	project := r.PathValue("project")

	proj, err := s.admin.DB.FindProjectByName(ctx, org, project)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return httputil.Errorf(http.StatusNotFound, "project not found")
		}
		return httputil.Error(http.StatusInternalServerError, err)
	}

	claims := auth.GetClaims(ctx)
	perms := claims.ProjectPermissions(ctx, proj.OrganizationID, proj.ID)
	if !perms.ManageProject {
		return httputil.Errorf(http.StatusForbidden, "does not have permission to manage publishes")
	}
	if r.Method != http.MethodPost {
		return httputil.Errorf(http.StatusMethodNotAllowed, "method %s not allowed", r.Method)
	}

	versionStr := r.PathValue("version")
	version, err := strconv.Atoi(versionStr)
	if err != nil || version <= 0 {
		return httputil.Errorf(http.StatusBadRequest, "invalid version")
	}

	target, err := s.admin.DB.FindProjectPublish(ctx, proj.ID, version)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return httputil.Errorf(http.StatusNotFound, "publish version %d not found", version)
		}
		return httputil.Error(http.StatusInternalServerError, err)
	}

	// Point the project at the historical asset and reconcile deployments.
	if err := s.setProjectArchiveAsset(ctx, proj, target.AssetID); err != nil {
		return httputil.Error(http.StatusInternalServerError, err)
	}

	// Record the rollback as a new publish entry so history stays append-only.
	pub, err := s.admin.DB.InsertProjectPublish(ctx, &database.InsertProjectPublishOptions{
		ProjectID:   proj.ID,
		AssetID:     target.AssetID,
		Note:        fmt.Sprintf("Rolled back to v%d", version),
		PublishedBy: s.publisherLabel(ctx, claims),
	})
	if err != nil {
		return httputil.Error(http.StatusInternalServerError, err)
	}

	return writeJSON(w, publishItemFromDB(pub, true))
}

// publishProject snapshots the dev deployment's current files into a new archive asset,
// switches production to it, and records a publish history entry.
func (s *Server) publishProject(w http.ResponseWriter, r *http.Request, proj *database.Project) error {
	ctx := r.Context()

	var body struct {
		Note string `json:"note"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10))
	if err := dec.Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		return httputil.Errorf(http.StatusBadRequest, "invalid request body: %s", err.Error())
	}
	note := strings.TrimSpace(body.Note)
	if runes := []rune(note); len(runes) > maxPublishNoteLen {
		note = string(runes[:maxPublishNoteLen])
	}

	if s.admin.Assets == nil {
		return httputil.Errorf(http.StatusPreconditionFailed, "assets storage is not configured")
	}

	// Package the current dev draft into an archive.
	buf, err := s.packageDevDraft(ctx, proj)
	if err != nil {
		return err
	}

	// Create the archive asset via a signed upload URL (works for both local and GCS stores).
	assetID := uuid.New().String()
	objectPath := path.Join("deploy", fmt.Sprintf("%s__%s__%s.tar.gz", proj.OrganizationID, proj.Name, assetID))
	objectURL := s.admin.Assets.ObjectURL(objectPath)
	signedURL, headers, err := s.admin.Assets.GenerateUploadURL(objectPath, maxAssetSizeForType["deploy"])
	if err != nil {
		return httputil.Error(http.StatusInternalServerError, err)
	}
	if err := archive.Upload(ctx, signedURL, buf, headers); err != nil {
		return httputil.Error(http.StatusInternalServerError, err)
	}
	if _, err := s.admin.DB.InsertAsset(ctx, assetID, proj.OrganizationID, objectURL, auth.GetClaims(ctx).OwnerID(), false); err != nil {
		return httputil.Error(http.StatusInternalServerError, err)
	}

	// Point the project at the new asset and reconcile deployments.
	if err := s.setProjectArchiveAsset(ctx, proj, assetID); err != nil {
		return httputil.Error(http.StatusInternalServerError, err)
	}

	// Record the publish history entry.
	pub, err := s.admin.DB.InsertProjectPublish(ctx, &database.InsertProjectPublishOptions{
		ProjectID:   proj.ID,
		AssetID:     assetID,
		Note:        note,
		PublishedBy: s.publisherLabel(ctx, auth.GetClaims(ctx)),
	})
	if err != nil {
		return httputil.Error(http.StatusInternalServerError, err)
	}

	return writeJSON(w, publishItemFromDB(pub, true))
}

// packageDevDraft fetches the dev deployment's current project files and packages them into a tar.gz.
func (s *Server) packageDevDraft(ctx context.Context, proj *database.Project) (*bytes.Buffer, error) {
	depl, err := s.findReadyDevDeployment(ctx, proj)
	if err != nil {
		return nil, err
	}

	rt, err := s.admin.OpenRuntimeClient(depl)
	if err != nil {
		return nil, httputil.Errorf(http.StatusPreconditionFailed, "could not connect to the Studio runtime: %s", err.Error())
	}
	defer rt.Close()

	list, err := rt.ListFiles(ctx, &runtimev1.ListFilesRequest{InstanceId: depl.RuntimeInstanceID, Glob: "**"})
	if err != nil {
		return nil, httputil.Error(http.StatusInternalServerError, err)
	}

	var entries []archive.BlobEntry
	for _, f := range list.Files {
		if f.IsDir {
			continue
		}
		// Skip the dev-only virtual files overlay (e.g. the data-request backlog).
		if strings.HasPrefix(strings.TrimPrefix(f.Path, "/"), "__virtual__") {
			continue
		}
		res, err := rt.GetFile(ctx, &runtimev1.GetFileRequest{InstanceId: depl.RuntimeInstanceID, Path: f.Path})
		if err != nil {
			return nil, httputil.Error(http.StatusInternalServerError, fmt.Errorf("failed to read project file %q: %w", f.Path, err))
		}
		entries = append(entries, archive.BlobEntry{Path: f.Path, Data: []byte(res.Blob)})
	}
	if len(entries) == 0 {
		return nil, httputil.Errorf(http.StatusPreconditionFailed, "there are no project files to publish")
	}

	buf, err := archive.CreateFromBlobs(ctx, entries)
	if err != nil {
		return nil, httputil.Error(http.StatusInternalServerError, err)
	}
	return buf, nil
}

// findReadyDevDeployment returns the dev deployment that has a running runtime, or an error.
func (s *Server) findReadyDevDeployment(ctx context.Context, proj *database.Project) (*database.Deployment, error) {
	depls, err := s.admin.DB.FindDeploymentsForProject(ctx, proj.ID, "dev", "")
	if err != nil {
		return nil, httputil.Error(http.StatusInternalServerError, err)
	}
	for _, d := range depls {
		if d.RuntimeHost != "" {
			return d, nil
		}
	}
	return nil, httputil.Errorf(http.StatusPreconditionFailed, "start Studio (the dev environment) before publishing")
}

// setProjectArchiveAsset points the project at an archive asset and reconciles its deployments.
// Changing ArchiveAssetID makes UpdateProject re-pull the archive into production automatically.
func (s *Server) setProjectArchiveAsset(ctx context.Context, proj *database.Project, assetID string) error {
	_, err := s.admin.UpdateProject(ctx, proj, &database.UpdateProjectOptions{
		Name:                 proj.Name,
		Description:          proj.Description,
		Public:               proj.Public,
		DirectoryName:        proj.DirectoryName,
		Provisioner:          proj.Provisioner,
		ArchiveAssetID:       &assetID,
		GitRemote:            proj.GitRemote,
		GithubInstallationID: proj.GithubInstallationID,
		GithubRepoID:         proj.GithubRepoID,
		ManagedGitRepoID:     proj.ManagedGitRepoID,
		ProdVersion:          proj.ProdVersion,
		PrimaryBranch:        proj.PrimaryBranch,
		Subpath:              proj.Subpath,
		PrimaryDeploymentID:  proj.PrimaryDeploymentID,
		ProdSlots:            proj.ProdSlots,
		ProdTTLSeconds:       proj.ProdTTLSeconds,
		DevSlots:             proj.DevSlots,
		DevTTLSeconds:        proj.DevTTLSeconds,
		OverrideDiskGB:       proj.OverrideDiskGB,
		Annotations:          proj.Annotations,
	})
	return err
}

// listPublishes returns the publish history with the current release flagged.
func (s *Server) listPublishes(ctx context.Context, proj *database.Project) ([]publishItem, error) {
	pubs, err := s.admin.DB.FindProjectPublishes(ctx, proj.ID, maxPublishHistory)
	if err != nil {
		return nil, httputil.Error(http.StatusInternalServerError, err)
	}
	items := make([]publishItem, 0, len(pubs))
	// Entries are sorted by version DESC; only the newest entry for the active asset is "current"
	// (a rollback re-uses the asset of an older version).
	flagged := false
	for _, p := range pubs {
		current := !flagged && proj.ArchiveAssetID != nil && *proj.ArchiveAssetID == p.AssetID
		if current {
			flagged = true
		}
		items = append(items, publishItemFromDB(p, current))
	}
	return items, nil
}

// publisherLabel resolves a human-friendly label for who performed a publish.
func (s *Server) publisherLabel(ctx context.Context, claims auth.Claims) string {
	if claims.OwnerType() != auth.OwnerTypeUser {
		return string(claims.OwnerType())
	}
	user, err := s.admin.DB.FindUser(ctx, claims.OwnerID())
	if err != nil {
		return ""
	}
	if user.DisplayName != "" {
		return user.DisplayName
	}
	return user.Email
}

func publishItemFromDB(p *database.ProjectPublish, current bool) publishItem {
	return publishItem{
		Version:     p.Version,
		Note:        p.Note,
		PublishedBy: p.PublishedBy,
		CreatedAt:   p.CreatedOn.UTC().Format(time.RFC3339),
		Current:     current,
	}
}
