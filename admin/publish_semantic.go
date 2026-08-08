package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fridencao/stardata/admin/database"
	"go.uber.org/zap"
)

// PublishProject executes the full publish pipeline for DB-mode projects:
//  1. Opens a new version in 'validating' status
//  2. Snapshots the current draft resources into that version
//  3. (TODO 5.2-T2: dry-run gate — for now, skip to step 4)
//  4. Promotes the version to 'published'
//  5. Points the project at the new version
//  6. Triggers the parser on the prod deployment so the runtime picks up the change
//
// The dry-run gate (step 3) is the most complex piece of 5.2 and is tracked as T2.
// For the Phase 5.2 MVP, publishing proceeds without dry-run so the full tracer
// bullet (save → publish → runtime reload → business sees it) can be exercised. The
// gate will be wired in once T2 lands, and the status lifecycle will remain unchanged.
func (s *Service) PublishProject(ctx context.Context, projectID string, note string, actorUserID *string) (*database.ProjectVersion, error) {
	proj, err := s.DB.FindProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if proj.SemanticLayerMode != "db" {
		return nil, fmt.Errorf("PublishProject: project %q is not in DB semantic layer mode", proj.Name)
	}

	// Step 1: Open a new version.
	ver, err := s.DB.InsertProjectVersion(ctx, &database.InsertProjectVersionOptions{
		ProjectID:         projectID,
		Note:              note,
		PublishedByUserID: actorUserID,
	})
	if err != nil {
		return nil, fmt.Errorf("publish: create version: %w", err)
	}
	s.Logger.Info("publish: version opened",
		zap.String("project_id", projectID),
		zap.Int("version", ver.Version),
	)

	// Step 2: Snapshot current drafts.
	n, err := s.DB.SnapshotDraftResources(ctx, ver.ID, projectID)
	if err != nil {
		// Mark rejected so the UI doesn't show a dangling 'validating' version.
		report, _ := json.Marshal(map[string]any{"error": err.Error()})
		_ = s.DB.UpdateProjectVersionStatus(ctx, ver.ID, database.ProjectVersionStatusRejected, report)
		return nil, fmt.Errorf("publish: snapshot: %w", err)
	}
	if n == 0 {
		report, _ := json.Marshal(map[string]any{"error": "project has no draft resources to publish"})
		_ = s.DB.UpdateProjectVersionStatus(ctx, ver.ID, database.ProjectVersionStatusRejected, report)
		return nil, fmt.Errorf("publish: no draft resources")
	}
	s.Logger.Info("publish: resources snapshotted",
		zap.String("project_id", projectID),
		zap.Int("version", ver.Version),
		zap.Int("count", n),
	)

	// Step 3: dry-run gate (placeholder — will be implemented in 5.2-T2).
	// For now: skip validation and promote directly.

	// Step 4: Promote.
	if err := s.DB.UpdateProjectVersionStatus(ctx, ver.ID, database.ProjectVersionStatusPublished, nil); err != nil {
		return nil, fmt.Errorf("publish: promote: %w", err)
	}

	// Step 5: Point project at the new version.
	if err := s.DB.SetProjectCurrentPublishedVersion(ctx, projectID, ver.ID); err != nil {
		return nil, fmt.Errorf("publish: set current version: %w", err)
	}

	// Step 6: Notify the runtime. For DB-mode projects the prod deployment is the
	// only one that exists. TriggerParser makes it do a pull → PullVirtualRepo sees
	// the new fingerprint → re-renders resources → parser fires → reconcile runs.
	if err := s.notifyRuntimeVersionChange(ctx, proj); err != nil {
		// Non-fatal: the publish succeeded in the DB, and the next poll cycle (or a
		// manual TriggerReconcile) will pick it up. Log rather than fail so the
		// governor does not think the publish was reverted.
		s.Logger.Warn("publish: failed to notify runtime (will catch up on next poll)",
			zap.String("project_id", projectID),
			zap.Int("version", ver.Version),
			zap.Error(err),
		)
	}

	// Audit
	s.RecordAudit(ctx, &AuditEventOptions{
		OrgID:       proj.OrganizationID,
		ProjectID:   &projectID,
		ActorUserID: actorUserID,
		EventType:   AuditEventProjectPublish,
		TargetID:    ver.ID,
		Payload:     map[string]any{"version": ver.Version, "note": note, "resources": n},
	})

	// Refetch for the published_on timestamp.
	ver, _ = s.DB.FindProjectVersion(ctx, ver.ID)
	return ver, nil
}

// notifyRuntimeVersionChange triggers the prod deployment's parser so it picks up
// the new virtual-file set. This is the Q18=A webhook: admin→runtime push on publish.
func (s *Service) notifyRuntimeVersionChange(ctx context.Context, proj *database.Project) error {
	if proj.PrimaryDeploymentID == nil || *proj.PrimaryDeploymentID == "" {
		return nil // No deployment to notify (project never deployed — shouldn't happen, but safe).
	}

	depl, err := s.DB.FindDeployment(ctx, *proj.PrimaryDeploymentID)
	if err != nil {
		return err
	}

	// Use a short timeout: the publish is already committed, and failing to
	// notify is not worth blocking the governor's UI.
	notifyCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	return s.TriggerParser(notifyCtx, depl)
}
