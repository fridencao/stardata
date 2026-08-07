package postgres

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/fridencao/stardata/admin/database"
	"github.com/stretchr/testify/require"
)

// testAuditEvents covers the audit log used for compliance in enterprise
// deployments: insert, org-scoped listing (newest first), and the project /
// event-type filters.
func testAuditEvents(t *testing.T, db database.DB) {
	ctx := context.Background()

	org, err := db.InsertOrganization(ctx, &database.InsertOrganizationOptions{
		Name:                                "audit_org",
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
		Name:           "audit_project",
		ProdSlots:      1,
	})
	require.NoError(t, err)

	// Empty to start.
	events, err := db.ListAuditEventsForOrg(ctx, org.ID, nil, 100)
	require.NoError(t, err)
	require.Empty(t, events)

	// Org-scoped event (no project).
	require.NoError(t, db.InsertAuditEvent(ctx, &database.InsertAuditEventOptions{
		OrgID:     org.ID,
		EventType: "member_add",
		TargetID:  "user-1",
		Payload:   map[string]any{"scope": "org", "role": "admin"},
	}))

	// Project-scoped events.
	require.NoError(t, db.InsertAuditEvent(ctx, &database.InsertAuditEventOptions{
		OrgID:     org.ID,
		ProjectID: &proj.ID,
		EventType: "project_publish",
		TargetID:  "asset-1",
		Payload:   map[string]any{"version": 1},
	}))
	require.NoError(t, db.InsertAuditEvent(ctx, &database.InsertAuditEventOptions{
		OrgID:     org.ID,
		ProjectID: &proj.ID,
		EventType: "project_rollback",
		TargetID:  "asset-1",
	}))

	// All three, newest first.
	events, err = db.ListAuditEventsForOrg(ctx, org.ID, nil, 100)
	require.NoError(t, err)
	require.Len(t, events, 3)
	require.Equal(t, "project_rollback", events[0].EventType)
	require.Equal(t, org.ID, events[0].OrgID)

	// Payload round-trips as JSON. A nil payload defaults to an empty object.
	var publishPayload map[string]any
	for _, e := range events {
		if e.EventType == "project_publish" {
			require.NoError(t, json.Unmarshal(e.Payload, &publishPayload))
		}
		if e.EventType == "project_rollback" {
			require.JSONEq(t, `{}`, string(e.Payload))
		}
	}
	require.Equal(t, float64(1), publishPayload["version"])

	// Project filter excludes the org-scoped row.
	events, err = db.ListAuditEventsForOrg(ctx, org.ID, &database.AuditEventFilter{ProjectID: &proj.ID}, 100)
	require.NoError(t, err)
	require.Len(t, events, 2)
	for _, e := range events {
		require.NotNil(t, e.ProjectID)
		require.Equal(t, proj.ID, *e.ProjectID)
	}

	// Event-type filter.
	events, err = db.ListAuditEventsForOrg(ctx, org.ID, &database.AuditEventFilter{EventType: "project_publish"}, 100)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "asset-1", events[0].TargetID)

	// Limit is honored.
	events, err = db.ListAuditEventsForOrg(ctx, org.ID, nil, 1)
	require.NoError(t, err)
	require.Len(t, events, 1)

	// Deleting the project keeps the audit trail (project_id is set to NULL, the
	// event is not cascaded away) — the record must survive for compliance.
	require.NoError(t, db.DeleteProject(ctx, proj.ID))
	events, err = db.ListAuditEventsForOrg(ctx, org.ID, nil, 100)
	require.NoError(t, err)
	require.Len(t, events, 3)
}
