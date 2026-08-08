package postgres

import (
	"context"
	"testing"

	"github.com/fridencao/stardata/admin/database"
	"github.com/stretchr/testify/require"
)

// testProjectVersions covers the atomic-publish contract: a version freezes the
// draft set at snapshot time, later edits do not disturb it, the lifecycle only
// reaches 'published' explicitly, and the project pointer can move forward and
// backward (which is what rollback is).
func testProjectVersions(t *testing.T, db database.DB) {
	ctx := context.Background()
	_, proj := semanticTestProject(t, db, "ver")

	// Seed two draft resources.
	mv, err := db.InsertSemanticResource(ctx, &database.InsertSemanticResourceOptions{
		ProjectID:    proj.ID,
		ResourceKind: "metrics_view",
		ResourceName: "revenue_mv",
		Definition:   []byte(`{"raw":"type: metrics_view\nmodel: orders\n"}`),
	})
	require.NoError(t, err)
	_, err = db.InsertSemanticResource(ctx, &database.InsertSemanticResourceOptions{
		ProjectID:    proj.ID,
		ResourceKind: "model",
		ResourceName: "orders",
		Definition:   []byte(`{"raw":"SELECT 1","format":"sql"}`),
	})
	require.NoError(t, err)

	// A new version opens as 'validating' — never straight to published, because the
	// dry-run gate has not run yet.
	v1, err := db.InsertProjectVersion(ctx, &database.InsertProjectVersionOptions{
		ProjectID: proj.ID,
		Note:      "first publish",
	})
	require.NoError(t, err)
	require.Equal(t, 1, v1.Version)
	require.Equal(t, database.ProjectVersionStatusValidating, v1.Status)
	require.Nil(t, v1.PublishedOn)

	// The snapshot captures one row per resource, not the whole history.
	n, err := db.SnapshotDraftResources(ctx, v1.ID, proj.ID)
	require.NoError(t, err)
	require.Equal(t, 2, n)

	// Snapshotting twice must not double-insert.
	n, err = db.SnapshotDraftResources(ctx, v1.ID, proj.ID)
	require.NoError(t, err)
	require.Equal(t, 0, n)

	// Promote to published; published_on gets stamped on that transition only.
	require.NoError(t, db.UpdateProjectVersionStatus(ctx, v1.ID, database.ProjectVersionStatusPublished, nil))
	got, err := db.FindProjectVersion(ctx, v1.ID)
	require.NoError(t, err)
	require.Equal(t, database.ProjectVersionStatusPublished, got.Status)
	require.NotNil(t, got.PublishedOn)

	require.NoError(t, db.SetProjectCurrentPublishedVersion(ctx, proj.ID, v1.ID))
	reloaded, err := db.FindProject(ctx, proj.ID)
	require.NoError(t, err)
	require.NotNil(t, reloaded.CurrentPublishedVersionID)
	require.Equal(t, v1.ID, *reloaded.CurrentPublishedVersionID)

	// Editing a resource after publish creates a new draft version. v1's snapshot
	// must still point at the OLD row — this is the guarantee that a published
	// metric keeps its definition while the governor edits the next one.
	mv2, err := db.InsertSemanticResource(ctx, &database.InsertSemanticResourceOptions{
		ProjectID:    proj.ID,
		ResourceKind: "metrics_view",
		ResourceName: "revenue_mv",
		Definition:   []byte(`{"raw":"type: metrics_view\nmodel: orders\n# edited\n"}`),
	})
	require.NoError(t, err)
	require.Equal(t, 2, mv2.Version)
	require.NotEqual(t, mv.ID, mv2.ID)

	// A second version snapshots the NEW row set.
	v2, err := db.InsertProjectVersion(ctx, &database.InsertProjectVersionOptions{
		ProjectID: proj.ID,
		Note:      "second publish",
	})
	require.NoError(t, err)
	require.Equal(t, 2, v2.Version)
	n, err = db.SnapshotDraftResources(ctx, v2.ID, proj.ID)
	require.NoError(t, err)
	require.Equal(t, 2, n)

	// A rejected dry-run records its report and leaves the version unpublished.
	v3, err := db.InsertProjectVersion(ctx, &database.InsertProjectVersionOptions{
		ProjectID: proj.ID,
		Note:      "bad publish",
	})
	require.NoError(t, err)
	report := []byte(`{"errors":["model orders failed to reconcile"]}`)
	require.NoError(t, db.UpdateProjectVersionStatus(ctx, v3.ID, database.ProjectVersionStatusRejected, report))
	got3, err := db.FindProjectVersion(ctx, v3.ID)
	require.NoError(t, err)
	require.Equal(t, database.ProjectVersionStatusRejected, got3.Status)
	require.JSONEq(t, string(report), string(got3.ValidationReport))
	require.Nil(t, got3.PublishedOn)

	// Latest-by-status ignores the rejected one.
	latestPublished, err := db.FindLatestProjectVersion(ctx, proj.ID, database.ProjectVersionStatusPublished)
	require.NoError(t, err)
	require.Equal(t, v1.Version, latestPublished.Version)

	// History is newest-first and includes every status.
	list, err := db.ListProjectVersions(ctx, proj.ID, 10)
	require.NoError(t, err)
	require.Len(t, list, 3)
	require.Equal(t, 3, list[0].Version)
	require.Equal(t, 1, list[2].Version)

	// Rollback is just repointing at an older version.
	require.NoError(t, db.UpdateProjectVersionStatus(ctx, v2.ID, database.ProjectVersionStatusPublished, nil))
	require.NoError(t, db.SetProjectCurrentPublishedVersion(ctx, proj.ID, v2.ID))
	require.NoError(t, db.SetProjectCurrentPublishedVersion(ctx, proj.ID, v1.ID))
	reloaded, err = db.FindProject(ctx, proj.ID)
	require.NoError(t, err)
	require.Equal(t, v1.ID, *reloaded.CurrentPublishedVersionID)

	// A resource row referenced by a version must not be deletable, otherwise
	// rollback would find a hole where a definition used to be.
	err = db.DeleteSemanticResource(ctx, proj.ID, "metrics_view", "revenue_mv")
	require.Error(t, err, "deleting a version-referenced resource must be refused")
}
