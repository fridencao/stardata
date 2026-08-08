package admin

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/fridencao/stardata/admin/database"
	"github.com/fridencao/stardata/runtime/drivers"
	"github.com/fridencao/stardata/runtime/pkg/activity"
	rillstorage "github.com/fridencao/stardata/runtime/storage"
	"go.uber.org/zap"
	"os"
)

// aiDrivers are the LLM drivers an org admin may select from the UI. Deliberately
// narrower than the set the admin process can be started with: these are the ones
// whose config surface is fully expressible as (base_url, model, api_key).
var aiDrivers = []string{"openai", "deepseek"}

// ValidAIDriver reports whether driver is selectable as an org-level AI provider.
func ValidAIDriver(driver string) bool {
	for _, d := range aiDrivers {
		if d == driver {
			return true
		}
	}
	return false
}

// AIDrivers returns the selectable org-level AI provider names.
func AIDrivers() []string {
	out := make([]string, len(aiDrivers))
	copy(out, aiDrivers)
	return out
}

// DefaultAIDriver returns the deployment-wide AI driver name resolved from env
// at startup. Used by the UI to show what an org is currently using when no
// override is set.
func (s *Service) DefaultAIDriver() string {
	if s.opts == nil {
		return ""
	}
	return s.opts.AIDriver
}

// aiCacheEntry holds an opened driver handle plus the config fingerprint it was
// opened with, so a config change invalidates it.
type aiCacheEntry struct {
	fingerprint string
	handle      drivers.Handle
	service     drivers.AIService
}

// aiResolver caches per-org AI services. Opening a driver is cheap but not free,
// and Complete() is on the ChatBI hot path, so handles are reused until the org's
// stored config changes.
type aiResolver struct {
	mu      sync.Mutex
	entries map[string]*aiCacheEntry
}

func newAIResolver() *aiResolver {
	return &aiResolver{entries: make(map[string]*aiCacheEntry)}
}

func aiFingerprint(driver, baseURL, model string, apiKey []byte) string {
	return fmt.Sprintf("%s|%s|%s|%d", driver, baseURL, model, len(apiKey))
}

// OpenAIService opens a standalone AI service for an explicit config. The caller
// owns the returned closer and must call it. Used by the connectivity test, which
// must not touch or poison the resolver cache.
func (s *Service) OpenAIService(driver, baseURL, model string, apiKey []byte) (drivers.AIService, func(), error) {
	if !ValidAIDriver(driver) {
		return nil, nil, fmt.Errorf("unsupported AI driver %q", driver)
	}
	cfg := aiDriverConfig(baseURL, model, apiKey)
	handle, err := drivers.Open(driver, "", "", cfg, rillstorage.MustNew(os.TempDir(), nil), activity.NewNoopClient(), s.Logger)
	if err != nil {
		return nil, nil, err
	}
	svc, ok := handle.AsAI("")
	if !ok {
		_ = handle.Close()
		return nil, nil, fmt.Errorf("driver %q does not implement the AI interface", driver)
	}
	return svc, func() { _ = handle.Close() }, nil
}

func aiDriverConfig(baseURL, model string, apiKey []byte) map[string]any {
	cfg := map[string]any{"api_key": string(apiKey)}
	if baseURL != "" {
		cfg["base_url"] = baseURL
	}
	if model != "" {
		cfg["model"] = model
	}
	return cfg
}

// AIForOrg returns the AI service to use for the given org. When the org has a
// stored config it is used; otherwise the deployment-wide env-var service applies.
// A failure to open the org's configured driver falls back to the deployment
// service rather than breaking chat outright — a bad saved config should degrade
// to the platform default, not take the org's ChatBI offline.
func (s *Service) AIForOrg(ctx context.Context, orgID string) drivers.AIService {
	if orgID == "" {
		return s.AI
	}

	cfg, err := s.DB.FindOrgAIConfig(ctx, orgID)
	if err != nil {
		if !errors.Is(err, database.ErrNotFound) {
			s.Logger.Warn("ai: failed to load org config, using deployment default",
				zap.String("org_id", orgID), zap.Error(err))
		}
		return s.AI
	}

	fp := aiFingerprint(cfg.Driver, cfg.BaseURL, cfg.Model, cfg.APIKey)

	s.aiResolver.mu.Lock()
	defer s.aiResolver.mu.Unlock()

	if e, ok := s.aiResolver.entries[orgID]; ok {
		if e.fingerprint == fp {
			return e.service
		}
		// Config changed — drop the stale handle before opening a new one.
		_ = e.handle.Close()
		delete(s.aiResolver.entries, orgID)
	}

	if !ValidAIDriver(cfg.Driver) {
		s.Logger.Warn("ai: org config has unsupported driver, using deployment default",
			zap.String("org_id", orgID), zap.String("driver", cfg.Driver))
		return s.AI
	}

	handle, err := drivers.Open(cfg.Driver, "", "", aiDriverConfig(cfg.BaseURL, cfg.Model, cfg.APIKey),
		rillstorage.MustNew(os.TempDir(), nil), activity.NewNoopClient(), s.Logger)
	if err != nil {
		s.Logger.Warn("ai: failed to open org driver, using deployment default",
			zap.String("org_id", orgID), zap.String("driver", cfg.Driver), zap.Error(err))
		return s.AI
	}
	svc, ok := handle.AsAI("")
	if !ok {
		_ = handle.Close()
		s.Logger.Warn("ai: org driver does not implement AI interface, using deployment default",
			zap.String("org_id", orgID), zap.String("driver", cfg.Driver))
		return s.AI
	}

	s.aiResolver.entries[orgID] = &aiCacheEntry{fingerprint: fp, handle: handle, service: svc}
	return svc
}

// InvalidateAIForOrg drops any cached AI service for the org. Called after a
// config write so the next completion picks up the change without a restart.
func (s *Service) InvalidateAIForOrg(orgID string) {
	s.aiResolver.mu.Lock()
	defer s.aiResolver.mu.Unlock()
	if e, ok := s.aiResolver.entries[orgID]; ok {
		_ = e.handle.Close()
		delete(s.aiResolver.entries, orgID)
	}
}

// closeAIResolver releases all cached org AI handles.
func (s *Service) closeAIResolver() {
	s.aiResolver.mu.Lock()
	defer s.aiResolver.mu.Unlock()
	for id, e := range s.aiResolver.entries {
		_ = e.handle.Close()
		delete(s.aiResolver.entries, id)
	}
}
