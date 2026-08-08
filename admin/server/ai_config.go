package server

import (
	"context"
	"errors"

	admin "github.com/fridencao/stardata/admin"
	"github.com/fridencao/stardata/admin/database"
	"github.com/fridencao/stardata/admin/server/auth"
	adminv1 "github.com/fridencao/stardata/proto/gen/stardata/admin/v1"
	aiv1 "github.com/fridencao/stardata/proto/gen/stardata/ai/v1"
	"github.com/fridencao/stardata/runtime/drivers"
	"github.com/fridencao/stardata/runtime/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// orgIDForAIRequest derives the org whose LLM config applies to the caller.
// ChatBI completions arrive with deployment claims, so the deployment's project
// determines the org. Anything else falls back to the deployment-wide config.
func (s *Server) orgIDForAIRequest(ctx context.Context) string {
	claims := auth.GetClaims(ctx)
	if claims == nil || claims.OwnerType() != auth.OwnerTypeDeployment {
		return ""
	}
	depl, err := s.admin.DB.FindDeployment(ctx, claims.OwnerID())
	if err != nil {
		return ""
	}
	proj, err := s.admin.DB.FindProject(ctx, depl.ProjectID)
	if err != nil {
		return ""
	}
	return proj.OrganizationID
}

func orgAIConfigToPB(cfg *database.OrgAIConfig) *adminv1.OrgAIConfig {
	if cfg == nil {
		return nil
	}
	return &adminv1.OrgAIConfig{
		OrgId:     cfg.OrgID,
		Driver:    cfg.Driver,
		BaseUrl:   cfg.BaseURL,
		Model:     cfg.Model,
		HasApiKey: len(cfg.APIKey) > 0,
		UpdatedOn: timestamppb.New(cfg.UpdatedOn),
	}
}

// requireOrgAIAdmin resolves the org and asserts the caller may manage its AI config.
func (s *Server) requireOrgAIAdmin(ctx context.Context, orgName string) (*database.Organization, error) {
	org, err := s.admin.DB.FindOrganizationByName(ctx, orgName)
	if err != nil {
		return nil, err
	}
	claims := auth.GetClaims(ctx)
	if !claims.OrganizationPermissions(ctx, org.ID).ManageOrg {
		return nil, status.Error(codes.PermissionDenied, "not allowed to manage AI configuration")
	}
	return org, nil
}

func (s *Server) GetOrgAIConfig(ctx context.Context, req *adminv1.GetOrgAIConfigRequest) (*adminv1.GetOrgAIConfigResponse, error) {
	observability.AddRequestAttributes(ctx, attribute.String("args.org", req.Org))

	org, err := s.requireOrgAIAdmin(ctx, req.Org)
	if err != nil {
		return nil, err
	}

	resp := &adminv1.GetOrgAIConfigResponse{
		// Reflect the deployment env config so the UI can show "current default"
		// when the org has no override yet. Value comes from admin startup opts.
		DefaultDriver: s.admin.DefaultAIDriver(),
	}

	cfg, err := s.admin.DB.FindOrgAIConfig(ctx, org.ID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return resp, nil
		}
		return nil, err
	}
	resp.Config = orgAIConfigToPB(cfg)
	return resp, nil
}

func (s *Server) SetOrgAIConfig(ctx context.Context, req *adminv1.SetOrgAIConfigRequest) (*adminv1.SetOrgAIConfigResponse, error) {
	observability.AddRequestAttributes(ctx,
		attribute.String("args.org", req.Org),
		attribute.String("args.driver", req.Driver),
	)

	org, err := s.requireOrgAIAdmin(ctx, req.Org)
	if err != nil {
		return nil, err
	}
	if !admin.ValidAIDriver(req.Driver) {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported AI driver %q", req.Driver)
	}

	claims := auth.GetClaims(ctx)
	opts := &database.UpsertOrgAIConfigOptions{
		OrgID:           org.ID,
		Driver:          req.Driver,
		BaseURL:         req.BaseUrl,
		Model:           req.Model,
		UpdatedByUserID: auditActor(claims),
	}
	if req.ApiKey == "" {
		// An empty key means "leave the stored secret alone" so the form can be
		// re-saved without the browser ever holding the key.
		existing, err := s.admin.DB.FindOrgAIConfig(ctx, org.ID)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return nil, status.Error(codes.InvalidArgument, "api_key is required when no key is stored yet")
			}
			return nil, err
		}
		if len(existing.APIKey) == 0 {
			return nil, status.Error(codes.InvalidArgument, "api_key is required when no key is stored yet")
		}
		opts.KeepExistingKey = true
	} else {
		opts.APIKey = []byte(req.ApiKey)
	}

	cfg, err := s.admin.DB.UpsertOrgAIConfig(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Drop the cached handle so the next completion uses the new config without a restart.
	s.admin.InvalidateAIForOrg(org.ID)

	s.admin.RecordAudit(ctx, &admin.AuditEventOptions{
		OrgID:       org.ID,
		ActorUserID: auditActor(claims),
		EventType:   admin.AuditEventOrgAIConfigSet,
		TargetID:    org.ID,
		Payload: map[string]any{
			"driver":   cfg.Driver,
			"base_url": cfg.BaseURL,
			"model":    cfg.Model,
			// Never the key itself; only whether it was rotated in this call.
			"api_key_rotated": req.ApiKey != "",
		},
	})

	return &adminv1.SetOrgAIConfigResponse{Config: orgAIConfigToPB(cfg)}, nil
}

func (s *Server) DeleteOrgAIConfig(ctx context.Context, req *adminv1.DeleteOrgAIConfigRequest) (*adminv1.DeleteOrgAIConfigResponse, error) {
	observability.AddRequestAttributes(ctx, attribute.String("args.org", req.Org))

	org, err := s.requireOrgAIAdmin(ctx, req.Org)
	if err != nil {
		return nil, err
	}
	if err := s.admin.DB.DeleteOrgAIConfig(ctx, org.ID); err != nil {
		return nil, err
	}
	s.admin.InvalidateAIForOrg(org.ID)

	s.admin.RecordAudit(ctx, &admin.AuditEventOptions{
		OrgID:       org.ID,
		ActorUserID: auditActor(auth.GetClaims(ctx)),
		EventType:   admin.AuditEventOrgAIConfigSet,
		TargetID:    org.ID,
		Payload:     map[string]any{"cleared": true},
	})

	return &adminv1.DeleteOrgAIConfigResponse{}, nil
}

// TestOrgAIConfig runs one minimal completion so an admin can validate credentials
// before saving them. Nothing is persisted and the resolver cache is untouched.
func (s *Server) TestOrgAIConfig(ctx context.Context, req *adminv1.TestOrgAIConfigRequest) (*adminv1.TestOrgAIConfigResponse, error) {
	observability.AddRequestAttributes(ctx, attribute.String("args.org", req.Org))

	org, err := s.requireOrgAIAdmin(ctx, req.Org)
	if err != nil {
		return nil, err
	}

	driver, baseURL, model := req.Driver, req.BaseUrl, req.Model
	apiKey := []byte(req.ApiKey)

	// Fill any blanks from the stored config so "test what's saved" works too.
	if driver == "" || len(apiKey) == 0 {
		stored, err := s.admin.DB.FindOrgAIConfig(ctx, org.ID)
		if err != nil {
			if errors.Is(err, database.ErrNotFound) {
				return &adminv1.TestOrgAIConfigResponse{
					Ok:      false,
					Message: "no AI configuration to test",
				}, nil
			}
			return nil, err
		}
		if driver == "" {
			driver = stored.Driver
		}
		if baseURL == "" {
			baseURL = stored.BaseURL
		}
		if model == "" {
			model = stored.Model
		}
		if len(apiKey) == 0 {
			apiKey = stored.APIKey
		}
	}

	if !admin.ValidAIDriver(driver) {
		return &adminv1.TestOrgAIConfigResponse{Ok: false, Message: "unsupported AI driver: " + driver}, nil
	}

	svc, closeFn, err := s.admin.OpenAIService(driver, baseURL, model, apiKey)
	if err != nil {
		return &adminv1.TestOrgAIConfigResponse{Ok: false, Message: err.Error()}, nil
	}
	defer closeFn()

	res, err := svc.Complete(ctx, &drivers.CompleteOptions{
		Messages: []*aiv1.CompletionMessage{{
			Role:    "user",
			Content: []*aiv1.ContentBlock{{BlockType: &aiv1.ContentBlock_Text{Text: "ping"}}},
		}},
	})
	if err != nil {
		return &adminv1.TestOrgAIConfigResponse{Ok: false, Message: err.Error()}, nil
	}
	if res == nil || res.Message == nil || len(res.Message.Content) == 0 {
		return &adminv1.TestOrgAIConfigResponse{Ok: false, Message: "the model returned an empty response"}, nil
	}

	return &adminv1.TestOrgAIConfigResponse{Ok: true, Provider: res.Provider}, nil
}
