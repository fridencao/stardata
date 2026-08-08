package admin

import (
	"context"
	"errors"
	"fmt"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/fridencao/stardata/admin/billing"
	"github.com/fridencao/stardata/admin/billing/payment"
	"github.com/fridencao/stardata/admin/database"
	"github.com/fridencao/stardata/admin/jobs"
	"github.com/fridencao/stardata/admin/pkg/assetstore"
	"github.com/fridencao/stardata/admin/provisioner"
	"github.com/fridencao/stardata/cli/pkg/version"
	"github.com/fridencao/stardata/runtime/drivers"
	"github.com/fridencao/stardata/runtime/pkg/email"
	"github.com/fridencao/stardata/runtime/server/auth"
	"go.uber.org/zap"
)

type Options struct {
	DatabaseDriver             string
	DatabaseDSN                string
	DatabaseEncryptionKeyring  string
	ExternalURL                string
	ExternalGRPCURL            string // Defaults to ExternalURL. Set separately when gRPC is served on a different host/port (e.g. behind an HTTP-only reverse proxy).
	FrontendURL                string
	ProvisionerSetJSON         string
	DefaultProvisioner         string
	Version                    version.Version
	MetricsProjectOrg          string
	MetricsProjectName         string
	AutoscalerCron             string
	ScaleDownConstraint        int
	AllowMockBilling           bool
	StoppedDeploymentRetention time.Duration
	// AIDriver is the deployment-wide LLM driver name resolved from env vars. Kept
	// for display only: it tells an org admin what applies when no org override exists.
	AIDriver string
}

type Service struct {
	DB                         database.DB
	Jobs                       jobs.Client
	URLs                       *URLs
	ProvisionerSet             map[string]provisioner.Provisioner
	Email                      *email.Client
	AI                         drivers.AIService
	Assets                     assetstore.Store
	Used                       *usedFlusher
	Logger                     *zap.Logger
	opts                       *Options
	issuer                     *auth.Issuer
	authCache                  *lru.Cache
	Version                    version.Version
	MetricsProjectID           string
	AutoscalerCron             string
	ScaleDownConstraint        int
	AllowMockBilling           bool
	StoppedDeploymentRetention time.Duration
	Biller                     billing.Biller
	PaymentProvider            payment.Provider
	aiResolver                 *aiResolver
}

func New(ctx context.Context, opts *Options, logger *zap.Logger, issuer *auth.Issuer, emailClient *email.Client, aiService drivers.AIService, assets assetstore.Store, biller billing.Biller, p payment.Provider) (*Service, error) {
	// Default the external gRPC URL to the external (HTTP) URL when not set separately.
	if opts.ExternalGRPCURL == "" {
		opts.ExternalGRPCURL = opts.ExternalURL
	}

	// Init db
	db, err := database.Open(opts.DatabaseDriver, opts.DatabaseDSN, opts.DatabaseEncryptionKeyring)
	if err != nil {
		logger.Fatal("error connecting to database", zap.Error(err))
	}

	// Init URLs
	urls, err := NewURLs(opts.ExternalURL, opts.FrontendURL)
	if err != nil {
		logger.Fatal("error parsing URLs", zap.Error(err))
	}

	// Auto-run migrations
	v1, err := db.FindMigrationVersion(ctx)
	if err != nil {
		logger.Fatal("error getting migration version", zap.Error(err))
	}
	err = db.Migrate(ctx)
	if err != nil {
		logger.Fatal("error migrating database", zap.Error(err))
	}
	v2, err := db.FindMigrationVersion(ctx)
	if err != nil {
		logger.Fatal("error getting migration version", zap.Error(err))
	}
	if v1 == v2 {
		logger.Info("database is up to date", zap.Int("version", v2))
	} else {
		logger.Info("database migrated", zap.Int("from_version", v1), zap.Int("to_version", v2))
	}

	// Create provisioner set
	provSet, err := provisioner.NewSet(opts.ProvisionerSetJSON, db, logger)
	if err != nil {
		return nil, err
	}

	// Verify that the specified default provisioner is in the provisioner set
	_, ok := provSet[opts.DefaultProvisioner]
	if !ok {
		return nil, fmt.Errorf("default provisioner %q is not in the provisioner set", opts.DefaultProvisioner)
	}

	// Look for the optional metrics project
	var metricsProjectID string
	if opts.MetricsProjectOrg != "" && opts.MetricsProjectName != "" {
		proj, err := db.FindProjectByName(ctx, opts.MetricsProjectOrg, opts.MetricsProjectName)
		if err != nil {
			logger.Error("error looking up metrics project", zap.Error(err))
			// Not returning the error since that causes a circular dependency. Remember that evening in Amsterdam?
		} else {
			metricsProjectID = proj.ID
		}
	}

	// Create the auth token cache. See auth_token.go for details.
	authCache, err := lru.New(_authCacheSize)
	if err != nil {
		return nil, fmt.Errorf("error creating auth token cache: %w", err)
	}

	return &Service{
		DB:                         db,
		URLs:                       urls,
		ProvisionerSet:             provSet,
		Email:                      emailClient,
		AI:                         aiService,
		Assets:                     assets,
		Used:                       newUsedFlusher(logger, db),
		Logger:                     logger,
		opts:                       opts,
		issuer:                     issuer,
		authCache:                  authCache,
		Version:                    opts.Version,
		MetricsProjectID:           metricsProjectID,
		AutoscalerCron:             opts.AutoscalerCron,
		ScaleDownConstraint:        opts.ScaleDownConstraint,
		AllowMockBilling:           opts.AllowMockBilling,
		StoppedDeploymentRetention: opts.StoppedDeploymentRetention,
		Biller:                     biller,
		PaymentProvider:            p,
		aiResolver:                 newAIResolver(),
	}, nil
}

func (s *Service) Close() error {
	s.closeAIResolver()

	var allErrs error
	for _, p := range s.ProvisionerSet {
		err := p.Close()
		if err != nil {
			allErrs = errors.Join(allErrs, err)
		}
	}

	s.Used.Close()

	err := s.DB.Close()
	if err != nil {
		allErrs = errors.Join(allErrs, err)
	}

	return allErrs
}
