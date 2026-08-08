package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	admin "github.com/fridencao/stardata/admin"
	"github.com/fridencao/stardata/admin/database"
	"github.com/fridencao/stardata/admin/server/auth"
	adminv1 "github.com/fridencao/stardata/proto/gen/stardata/admin/v1"
	"github.com/fridencao/stardata/runtime/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/yaml.v3"
)

// StarData Phase 5: semantic resource CRUD.
//
// Saving is validate-then-store. Per the Phase 5 design (Q19), the draft layer does
// syntax and reference checking only — it does not materialize data. So a governor
// gets immediate feedback that a definition is well-formed and that everything it
// references exists, while real numbers come from the Preview dry-run in 5.2. This
// keeps the draft layer cheap: no second catalog, no staging schema.

// validSemanticResourceKinds mirrors the semantic_resource_kind enum. Validating
// here rather than relying on the DB's enum lets us return a helpful message
// instead of a raw constraint violation.
var validSemanticResourceKinds = map[string]bool{
	"source": true, "model": true, "metrics_view": true, "explore": true,
	"canvas": true, "report": true, "alert": true, "theme": true,
	"api": true, "config": true,
}

func semanticResourceToPB(r *database.SemanticResource) *adminv1.SemanticResourceInfo {
	if r == nil {
		return nil
	}
	out := &adminv1.SemanticResourceInfo{
		Id:           r.ID,
		ProjectId:    r.ProjectID,
		ResourceKind: r.ResourceKind,
		ResourceName: r.ResourceName,
		Version:      int32(r.Version),
		Status:       string(r.Status),
		CreatedOn:    timestamppb.New(r.CreatedOn),
		UpdatedOn:    timestamppb.New(r.UpdatedOn),
	}
	if r.CreatedByUserID != nil {
		out.CreatedByUserId = *r.CreatedByUserID
	}
	// Surface the raw body so the editor can round-trip exactly what was saved.
	var def map[string]any
	if json.Unmarshal(r.Definition, &def) == nil {
		if raw, ok := def["raw"].(string); ok {
			out.DefinitionRaw = raw
		}
	}
	return out
}

func (s *Server) ListSemanticResources(ctx context.Context, req *adminv1.ListSemanticResourcesRequest) (*adminv1.ListSemanticResourcesResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
	)

	proj, err := s.requireProjectEditor(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	rows, err := s.admin.DB.FindSemanticResources(ctx, proj.ID, database.SemanticResourceStatusDraft)
	if err != nil {
		return nil, err
	}

	out := make([]*adminv1.SemanticResourceInfo, 0, len(rows))
	for _, r := range rows {
		out = append(out, semanticResourceToPB(r))
	}
	return &adminv1.ListSemanticResourcesResponse{Resources: out}, nil
}

func (s *Server) GetSemanticResource(ctx context.Context, req *adminv1.GetSemanticResourceRequest) (*adminv1.GetSemanticResourceResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
		attribute.String("args.resource_kind", req.ResourceKind),
		attribute.String("args.resource_name", req.ResourceName),
	)

	proj, err := s.requireProjectEditor(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	r, err := s.admin.DB.FindSemanticResource(ctx, proj.ID, req.ResourceKind, req.ResourceName, database.SemanticResourceStatusDraft)
	if err != nil {
		return nil, err
	}
	return &adminv1.GetSemanticResourceResponse{Resource: semanticResourceToPB(r)}, nil
}

// SaveSemanticResource appends a new draft version. It requires the caller to hold
// the project's editing lock: without that check the lock would be advisory only and
// two governors could still clobber each other through the API.
func (s *Server) SaveSemanticResource(ctx context.Context, req *adminv1.SaveSemanticResourceRequest) (*adminv1.SaveSemanticResourceResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
		attribute.String("args.resource_kind", req.ResourceKind),
		attribute.String("args.resource_name", req.ResourceName),
	)

	proj, err := s.requireProjectEditor(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}
	if !validSemanticResourceKinds[req.ResourceKind] {
		return nil, status.Errorf(codes.InvalidArgument, "unknown resource_kind %q", req.ResourceKind)
	}

	claims := auth.GetClaims(ctx)
	if err := s.assertHoldsEditLock(ctx, proj.ID, claims); err != nil {
		return nil, err
	}

	// Validate before writing. A rejected save leaves the previous version as the
	// latest, so a bad edit cannot break what the runtime is currently reading.
	validationErrs, err := s.validateSemanticResource(ctx, proj.ID, req)
	if err != nil {
		return nil, err
	}
	if len(validationErrs) > 0 {
		return &adminv1.SaveSemanticResourceResponse{ValidationErrors: validationErrs}, nil
	}

	def := map[string]any{"raw": req.DefinitionRaw}
	if req.Format != "" {
		def["format"] = req.Format
	}
	defJSON, err := json.Marshal(def)
	if err != nil {
		return nil, err
	}

	actor := auditActor(claims)
	row, err := s.admin.DB.InsertSemanticResource(ctx, &database.InsertSemanticResourceOptions{
		ProjectID:       proj.ID,
		ResourceKind:    req.ResourceKind,
		ResourceName:    req.ResourceName,
		Definition:      defJSON,
		CreatedByUserID: actor,
	})
	if err != nil {
		return nil, err
	}

	s.admin.RecordAudit(ctx, &admin.AuditEventOptions{
		OrgID:       proj.OrganizationID,
		ProjectID:   &proj.ID,
		ActorUserID: actor,
		EventType:   admin.AuditEventSemanticResourceSave,
		TargetID:    row.ID,
		Payload: map[string]any{
			"kind":    row.ResourceKind,
			"name":    row.ResourceName,
			"version": row.Version,
		},
	})

	return &adminv1.SaveSemanticResourceResponse{Resource: semanticResourceToPB(row)}, nil
}

func (s *Server) DeleteSemanticResource(ctx context.Context, req *adminv1.DeleteSemanticResourceRequest) (*adminv1.DeleteSemanticResourceResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.project", req.Project),
		attribute.String("args.resource_kind", req.ResourceKind),
		attribute.String("args.resource_name", req.ResourceName),
	)

	proj, err := s.requireProjectEditor(ctx, req.Org, req.Project)
	if err != nil {
		return nil, err
	}

	claims := auth.GetClaims(ctx)
	if err := s.assertHoldsEditLock(ctx, proj.ID, claims); err != nil {
		return nil, err
	}

	// Refuse to delete something other resources still point at, otherwise the next
	// parse would fail on dangling references instead of here where we can explain it.
	dependents, err := s.findSemanticDependents(ctx, proj.ID, req.ResourceName)
	if err != nil {
		return nil, err
	}
	if len(dependents) > 0 {
		return nil, status.Errorf(codes.FailedPrecondition,
			"cannot delete %q: still referenced by %s", req.ResourceName, strings.Join(dependents, ", "))
	}

	if err := s.admin.DB.DeleteSemanticResource(ctx, proj.ID, req.ResourceKind, req.ResourceName); err != nil {
		return nil, err
	}

	s.admin.RecordAudit(ctx, &admin.AuditEventOptions{
		OrgID:       proj.OrganizationID,
		ProjectID:   &proj.ID,
		ActorUserID: auditActor(claims),
		EventType:   admin.AuditEventSemanticResourceDelete,
		TargetID:    req.ResourceName,
		Payload:     map[string]any{"kind": req.ResourceKind, "name": req.ResourceName},
	})

	return &adminv1.DeleteSemanticResourceResponse{}, nil
}

// assertHoldsEditLock fails the request unless the caller currently holds the
// project's editing lock.
func (s *Server) assertHoldsEditLock(ctx context.Context, projectID string, claims auth.Claims) error {
	if claims.OwnerType() != auth.OwnerTypeUser {
		return status.Error(codes.PermissionDenied, "only users can edit semantic resources")
	}
	lock, err := s.admin.DB.FindEditingLock(ctx, projectID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return status.Error(codes.FailedPrecondition, "acquire the editing lock before saving")
		}
		return err
	}
	if lock.LockedByUserID != claims.OwnerID() {
		return status.Error(codes.FailedPrecondition, "another user holds the editing lock for this project")
	}
	return nil
}

// validateSemanticResource performs the draft-layer checks: the body must be
// well-formed YAML, and anything it references must already exist in the project.
// It returns human-readable messages rather than an error so the editor can show
// them inline; a non-nil error means the check itself failed.
func (s *Server) validateSemanticResource(ctx context.Context, projectID string, req *adminv1.SaveSemanticResourceRequest) ([]string, error) {
	var errs []string

	// A model saved as bare SQL is not YAML, so only structural resources are parsed.
	isBareSQL := req.ResourceKind == "model" && strings.EqualFold(req.Format, "sql")

	var body map[string]any
	if !isBareSQL {
		if err := yaml.Unmarshal([]byte(req.DefinitionRaw), &body); err != nil {
			// Syntax errors stop here: reference checks on an unparseable body would
			// only produce noise on top of the real problem.
			return []string{fmt.Sprintf("YAML 语法错误：%v", err)}, nil
		}
		if body == nil {
			return []string{"定义内容为空"}, nil
		}
	}

	// Reference completeness. Only the references we can resolve cheaply are checked;
	// full graph validation happens during the publish dry-run.
	refs := collectSemanticRefs(req.ResourceKind, body)
	if len(refs) > 0 {
		existing, err := s.admin.DB.FindSemanticResources(ctx, projectID, database.SemanticResourceStatusDraft)
		if err != nil {
			return nil, err
		}
		known := make(map[string]bool, len(existing))
		for _, r := range existing {
			known[strings.ToLower(r.ResourceName)] = true
		}
		// A resource may legitimately reference itself by name across versions.
		known[strings.ToLower(req.ResourceName)] = true

		for _, ref := range refs {
			if !known[strings.ToLower(ref)] {
				errs = append(errs, fmt.Sprintf("引用的资源 %q 不存在", ref))
			}
		}
	}

	return errs, nil
}

// collectSemanticRefs extracts the resource names a definition depends on. It is
// intentionally shallow — the fields that matter for the metrics_view tracer bullet
// plus the obvious dashboard case — and returns nothing for kinds it does not know.
func collectSemanticRefs(kind string, body map[string]any) []string {
	if body == nil {
		return nil
	}
	var refs []string
	add := func(v any) {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			refs = append(refs, s)
		}
	}

	switch kind {
	case "metrics_view":
		// A metrics view reads from either a model or a physical table.
		add(body["model"])
	case "explore", "canvas":
		add(body["metrics_view"])
		if mvs, ok := body["metrics_views"].([]any); ok {
			for _, mv := range mvs {
				add(mv)
			}
		}
	case "report", "alert":
		add(body["metrics_view"])
	}
	return refs
}

// findSemanticDependents returns display labels for resources that reference name.
func (s *Server) findSemanticDependents(ctx context.Context, projectID, name string) ([]string, error) {
	rows, err := s.admin.DB.FindSemanticResources(ctx, projectID, database.SemanticResourceStatusDraft)
	if err != nil {
		return nil, err
	}

	var out []string
	for _, r := range rows {
		if strings.EqualFold(r.ResourceName, name) {
			continue
		}
		var def map[string]any
		if json.Unmarshal(r.Definition, &def) != nil {
			continue
		}
		raw, _ := def["raw"].(string)
		if raw == "" {
			continue
		}
		var body map[string]any
		if yaml.Unmarshal([]byte(raw), &body) != nil {
			continue
		}
		for _, ref := range collectSemanticRefs(r.ResourceKind, body) {
			if strings.EqualFold(ref, name) {
				out = append(out, fmt.Sprintf("%s/%s", r.ResourceKind, r.ResourceName))
				break
			}
		}
	}
	return out, nil
}
