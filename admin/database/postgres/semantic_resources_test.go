package postgres

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/fridencao/stardata/admin/database"
	"github.com/stretchr/testify/require"
)

func semanticTestProject(t *testing.T, db database.DB, suffix string) (*database.Organization, *database.Project) {
	t.Helper()
	ctx := context.Background()

	org, err := db.InsertOrganization(ctx, &database.InsertOrganizationOptions{
		Name:                                "sem_org_" + suffix,
		QuotaProjects:                       10,
		QuotaDeployments:                    10,
		QuotaSlotsTotal:                     10,
		QuotaSlotsPerDeployment:             10,
		QuotaOutstandingInvites:             10,
		QuotaStorageLimitBytesPerDeployment: -1,
	})
	require.NoError(t, err)

	proj, err := db.InsertProject(ctx, &database.InsertProjectOptions{
		OrganizationID: org.ID,
		Name:           "sem_project_" + suffix,
		ProdSlots:      1,
	})
	require.NoError(t, err)

	return org, proj
}

// testSemanticResources covers the append-only version chain that replaces yaml
// files: inserts bump the version, reads collapse to the newest version, name
// matching is case-insensitive, and the fingerprint moves whenever content does.
func testSemanticResources(t *testing.T, db database.DB) {
	ctx := context.Background()
	_, proj := semanticTestProject(t, db, "res")

	def1, err := json.Marshal(map[string]any{"model": "orders", "measures": []any{map[string]any{"name": "revenue"}}})
	require.NoError(t, err)

	// A brand new resource starts at version 1.
	r1, err := db.InsertSemanticResource(ctx, &database.InsertSemanticResourceOptions{
		ProjectID:    proj.ID,
		ResourceKind: "metrics_view",
		ResourceName: "Revenue_MV",
		Definition:   def1,
	})
	require.NoError(t, err)
	require.Equal(t, 1, r1.Version)
	require.Equal(t, database.SemanticResourceStatusDraft, r1.Status)

	// Saving again appends version 2 rather than overwriting version 1.
	def2, err := json.Marshal(map[string]any{"model": "orders", "measures": []any{map[string]any{"name": "revenue_net"}}})
	require.NoError(t, err)
	r2, err := db.InsertSemanticResource(ctx, &database.InsertSemanticResourceOptions{
		ProjectID:    proj.ID,
		ResourceKind: "metrics_view",
		ResourceName: "Revenue_MV",
		Definition:   def2,
	})
	require.NoError(t, err)
	require.Equal(t, 2, r2.Version)

	// Reads resolve to the newest version, and match the name case-insensitively.
	got, err := db.FindSemanticResource(ctx, proj.ID, "metrics_view", "revenue_mv", database.SemanticResourceStatusDraft)
	require.NoError(t, err)
	require.Equal(t, 2, got.Version)
	require.JSONEq(t, string(def2), string(got.Definition))

	// A second resource should not disturb the first one's version chain.
	_, err = db.InsertSemanticResource(ctx, &database.InsertSemanticResourceOptions{
		ProjectID:    proj.ID,
		ResourceKind: "model",
		ResourceName: "orders",
		Definition:   []byte(`{"sql":"SELECT 1"}`),
	})
	require.NoError(t, err)

	// Listing returns one row per resource — the newest — not the whole history.
	list, err := db.FindSemanticResources(ctx, proj.ID, database.SemanticResourceStatusDraft)
	require.NoError(t, err)
	require.Len(t, list, 2)
	byName := map[string]*database.SemanticResource{}
	for _, r := range list {
		byName[r.ResourceName] = r
	}
	require.Equal(t, 2, byName["Revenue_MV"].Version)
	require.Equal(t, 1, byName["orders"].Version)

	// The fingerprint must change when content changes, otherwise the runtime would
	// not notice edits.
	fp1, err := db.FindSemanticResourceFingerprint(ctx, proj.ID, database.SemanticResourceStatusDraft)
	require.NoError(t, err)
	require.NotEmpty(t, fp1)

	_, err = db.InsertSemanticResource(ctx, &database.InsertSemanticResourceOptions{
		ProjectID:    proj.ID,
		ResourceKind: "model",
		ResourceName: "orders",
		Definition:   []byte(`{"sql":"SELECT 2"}`),
	})
	require.NoError(t, err)

	fp2, err := db.FindSemanticResourceFingerprint(ctx, proj.ID, database.SemanticResourceStatusDraft)
	require.NoError(t, err)
	require.NotEqual(t, fp1, fp2)

	// Deleting drops the entire version chain for that resource only.
	require.NoError(t, db.DeleteSemanticResource(ctx, proj.ID, "metrics_view", "revenue_mv"))
	list, err = db.FindSemanticResources(ctx, proj.ID, database.SemanticResourceStatusDraft)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "orders", list[0].ResourceName)

	_, err = db.FindSemanticResource(ctx, proj.ID, "metrics_view", "revenue_mv", database.SemanticResourceStatusDraft)
	require.ErrorIs(t, err, database.ErrNotFound)
}

// testEditingLocks covers the contention rules that named branches used to make
// unnecessary: one holder at a time, re-entrant for that holder, stealable only
// once expired, and forcibly releasable by an admin.
func testEditingLocks(t *testing.T, db database.DB) {
	ctx := context.Background()
	org, proj := semanticTestProject(t, db, "lock")

	userA, err := db.InsertUser(ctx, &database.InsertUserOptions{
		Email:               "lock_a@example.com",
		DisplayName:         "Lock A",
		QuotaSingleuserOrgs: -1,
		QuotaTrialOrgs:      -1,
	})
	require.NoError(t, err)
	userB, err := db.InsertUser(ctx, &database.InsertUserOptions{
		Email:               "lock_b@example.com",
		DisplayName:         "Lock B",
		QuotaSingleuserOrgs: -1,
		QuotaTrialOrgs:      -1,
	})
	require.NoError(t, err)
	_ = org

	// A free project has no lock.
	_, err = db.FindEditingLock(ctx, proj.ID)
	require.ErrorIs(t, err, database.ErrNotFound)

	// A acquires.
	lock, err := db.AcquireEditingLock(ctx, proj.ID, userA.ID, time.Hour)
	require.NoError(t, err)
	require.Equal(t, userA.ID, lock.LockedByUserID)

	// Re-acquiring as the same holder is allowed (page reload, reconnect).
	lock, err = db.AcquireEditingLock(ctx, proj.ID, userA.ID, time.Hour)
	require.NoError(t, err)
	require.Equal(t, userA.ID, lock.LockedByUserID)

	// B cannot steal a live lock, and learns who holds it.
	held, err := db.AcquireEditingLock(ctx, proj.ID, userB.ID, time.Hour)
	require.ErrorIs(t, err, database.ErrNotUnique)
	require.NotNil(t, held)
	require.Equal(t, userA.ID, held.LockedByUserID)

	// Heartbeat extends A's expiry; B's heartbeat is rejected outright.
	beat, err := db.HeartbeatEditingLock(ctx, proj.ID, userA.ID, 2*time.Hour)
	require.NoError(t, err)
	require.True(t, beat.ExpiresAt.After(lock.ExpiresAt))

	_, err = db.HeartbeatEditingLock(ctx, proj.ID, userB.ID, time.Hour)
	require.ErrorIs(t, err, database.ErrNotFound)

	// B releasing someone else's lock is a no-op, not a hijack.
	require.NoError(t, db.ReleaseEditingLock(ctx, proj.ID, userB.ID))
	still, err := db.FindEditingLock(ctx, proj.ID)
	require.NoError(t, err)
	require.Equal(t, userA.ID, still.LockedByUserID)

	// An expired lock is invisible to readers and freely stealable.
	_, err = db.AcquireEditingLock(ctx, proj.ID, userA.ID, -time.Minute)
	require.NoError(t, err)
	_, err = db.FindEditingLock(ctx, proj.ID)
	require.ErrorIs(t, err, database.ErrNotFound)

	taken, err := db.AcquireEditingLock(ctx, proj.ID, userB.ID, time.Hour)
	require.NoError(t, err)
	require.Equal(t, userB.ID, taken.LockedByUserID)

	// Force release ignores the holder — the admin recovery path.
	require.NoError(t, db.ForceReleaseEditingLock(ctx, proj.ID))
	_, err = db.FindEditingLock(ctx, proj.ID)
	require.ErrorIs(t, err, database.ErrNotFound)

	// The sweeper reclaims expired rows.
	_, err = db.AcquireEditingLock(ctx, proj.ID, userA.ID, -time.Minute)
	require.NoError(t, err)
	n, err := db.DeleteExpiredEditingLocks(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, n, 1)
}
