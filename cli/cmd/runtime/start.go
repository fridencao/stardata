package runtime

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/redis/go-redis/v9"
	"github.com/fridencao/stardata/cli/pkg/cmdutil"
	runtimev1 "github.com/fridencao/stardata/proto/gen/stardata/runtime/v1"
	"github.com/fridencao/stardata/runtime"
	"github.com/fridencao/stardata/runtime/pkg/activity"
	"github.com/fridencao/stardata/runtime/pkg/debugserver"
	"github.com/fridencao/stardata/runtime/pkg/email"
	"github.com/fridencao/stardata/runtime/pkg/graceful"
	"github.com/fridencao/stardata/runtime/pkg/observability"
	"github.com/fridencao/stardata/runtime/pkg/ratelimit"
	"github.com/fridencao/stardata/runtime/server"
	"github.com/fridencao/stardata/runtime/server/auth"
	"github.com/fridencao/stardata/runtime/storage"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3"

	// Load connectors and reconcilers for runtime
	_ "github.com/fridencao/stardata/runtime/drivers/admin"
	_ "github.com/fridencao/stardata/runtime/drivers/athena"
	_ "github.com/fridencao/stardata/runtime/drivers/azure"
	_ "github.com/fridencao/stardata/runtime/drivers/bigquery"
	_ "github.com/fridencao/stardata/runtime/drivers/claude"
	_ "github.com/fridencao/stardata/runtime/drivers/clickhouse"
	_ "github.com/fridencao/stardata/runtime/drivers/databricks"
	_ "github.com/fridencao/stardata/runtime/drivers/druid"
	_ "github.com/fridencao/stardata/runtime/drivers/duckdb"
	_ "github.com/fridencao/stardata/runtime/drivers/file"
	_ "github.com/fridencao/stardata/runtime/drivers/gcs"
	_ "github.com/fridencao/stardata/runtime/drivers/gemini"
	_ "github.com/fridencao/stardata/runtime/drivers/https"
	_ "github.com/fridencao/stardata/runtime/drivers/mock/ai"
	_ "github.com/fridencao/stardata/runtime/drivers/mysql"
	_ "github.com/fridencao/stardata/runtime/drivers/openai"
	_ "github.com/fridencao/stardata/runtime/drivers/deepseek"
	_ "github.com/fridencao/stardata/runtime/drivers/pinot"
	_ "github.com/fridencao/stardata/runtime/drivers/postgres"
	_ "github.com/fridencao/stardata/runtime/drivers/redshift"
	_ "github.com/fridencao/stardata/runtime/drivers/s3"
	_ "github.com/fridencao/stardata/runtime/drivers/salesforce"
	_ "github.com/fridencao/stardata/runtime/drivers/slack"
	_ "github.com/fridencao/stardata/runtime/drivers/snowflake"
	_ "github.com/fridencao/stardata/runtime/drivers/sqlite"
	_ "github.com/fridencao/stardata/runtime/drivers/starrocks"
	_ "github.com/fridencao/stardata/runtime/reconcilers"
	_ "github.com/fridencao/stardata/runtime/resolvers"
)

// Config describes runtime server config derived from environment variables.
// Env var keys must be prefixed with RILL_RUNTIME_ and are converted from snake_case to CamelCase.
// For example RILL_RUNTIME_HTTP_PORT is mapped to Config.HTTPPort.
type Config struct {
	MetastoreDriver         string                 `default:"sqlite" split_words:"true"`
	MetastoreURL            string                 `default:"file:rill?mode=memory&cache=shared" split_words:"true"`
	MetastoreID             string                 `split_words:"true"`
	RedisURL                string                 `default:"" split_words:"true"`
	MetricsExporter         observability.Exporter `default:"prometheus" split_words:"true"`
	TracesExporter          observability.Exporter `default:"" split_words:"true"`
	LogLevel                zapcore.Level          `default:"info" split_words:"true"`
	HTTPPort                int                    `default:"8080" split_words:"true"`
	GRPCPort                int                    `default:"8080" split_words:"true"`
	DebugPort               int                    `default:"6060" split_words:"true"`
	AllowedOrigins          []string               `default:"*" split_words:"true"`
	SessionKeyPairs         []string               `split_words:"true"`
	AuthEnable              bool                   `default:"false" split_words:"true"`
	AuthIssuerURL           string                 `default:"" split_words:"true"`
	AuthAudienceURL         string                 `default:"" split_words:"true"`
	EmailSMTPHost           string                 `split_words:"true"`
	EmailSMTPPort           int                    `split_words:"true"`
	EmailSMTPUsername       string                 `split_words:"true"`
	EmailSMTPPassword       string                 `split_words:"true"`
	EmailSenderEmail        string                 `split_words:"true"`
	EmailSenderName         string                 `split_words:"true"`
	EmailBCC                string                 `split_words:"true"`
	ConnectionCacheSize     int                    `default:"100" split_words:"true"`
	QueryCacheSizeBytes     int64                  `default:"104857600" split_words:"true"` // 100MB by default
	SecurityEngineCacheSize int                    `default:"1000" split_words:"true"`
	LogBufferCapacity       int                    `default:"10000" split_words:"true"`    // 10k log lines
	LogBufferSizeBytes      int64                  `default:"16777216" split_words:"true"` // 16MB by default
	// AllowHostAccess controls whether instance can use host credentials and
	// local_file sources can access directory outside repo
	AllowHostAccess bool `default:"false" split_words:"true"`
	// ExternalURL is the public base URL of this server. Used as the JWT audience claim
	// and the OIDC redirect base. Falls back to http://localhost:<http_port> when empty.
	ExternalURL string `split_words:"true" yaml:"external_url"`
	// Auth is the self-hosted authentication configuration (see config.example.yaml `auth:`).
	Auth *auth.AuthConfig `yaml:"auth"`
	// DataDir stores data for all instances like duckdb file, temporary downloaded file etc.
	// The data for each instance is stored in a child directory named instance_id
	DataDir string `split_words:"true"`
	// DataBucket is a common GCS bucket to store data for all instances. This data is expected to be persisted across resets.
	DataBucket                string `split_words:"true"`
	DataBucketCredentialsJSON string `split_words:"true"`
	// Sink type of activity client: noop (or empty string), kafka
	ActivitySinkType string `default:"" split_words:"true"`
	// Kafka brokers of an activity client's sink
	ActivitySinkKafkaBrokers string `default:"" split_words:"true"`
	// Kafka topic of an activity client's sink
	ActivitySinkKafkaTopic string `default:"" split_words:"true"`
}

// StartCmd starts a stand-alone runtime server. It only allows configuration using environment variables.
func StartCmd(ch *cmdutil.Helper) *cobra.Command {
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start stand-alone runtime server",
		Run: func(cmd *cobra.Command, args []string) {
			// Load .env (note: fails silently if .env has errors)
			_ = godotenv.Load()

			// Init config
			var conf Config
			err := envconfig.Process("rill_runtime", &conf)
			if err != nil {
				fmt.Printf("failed to load config: %s\n", err.Error())
				os.Exit(1)
			}

			// envconfig allocates conf.Auth even when no auth env vars are set;
			// treat an all-zero config as "not set" so the YAML overlay (and the
			// auth-disabled default) work as intended.
			if !conf.Auth.IsConfigured() {
				conf.Auth = nil
			}

			// Overlay YAML config file (STARDATA_CONFIG) when present.
			// Environment variables take precedence: only fill values that were not already set via env.
			if cfgPath := os.Getenv("STARDATA_CONFIG"); cfgPath != "" {
				if data, rerr := os.ReadFile(cfgPath); rerr == nil {
					var yc struct {
						Server struct {
							ExternalURL string `yaml:"external_url"`
						} `yaml:"server"`
						Auth *auth.AuthConfig `yaml:"auth"`
					}
					if yerr := yaml.Unmarshal(data, &yc); yerr != nil {
						fmt.Printf("failed to parse %s: %s\n", cfgPath, yerr.Error())
						os.Exit(1)
					}
					if conf.ExternalURL == "" && yc.Server.ExternalURL != "" {
						conf.ExternalURL = yc.Server.ExternalURL
					}
					if conf.Auth == nil && yc.Auth != nil {
						conf.Auth = yc.Auth
					}
				} else {
					fmt.Printf("warning: could not read STARDATA_CONFIG=%s: %s\n", cfgPath, rerr.Error())
				}
			}

			// Init logger
			cfg := zap.NewProductionConfig()
			cfg.Level.SetLevel(conf.LogLevel)
			cfg.EncoderConfig.NameKey = zapcore.OmitKey
			cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
			logger, err := cfg.Build()
			if err != nil {
				fmt.Printf("error: failed to create logger: %s\n", err.Error())
				os.Exit(1)
			}

			// Init email client
			var sender email.Sender
			if conf.EmailSMTPHost != "" {
				sender, err = email.NewSMTPSender(&email.SMTPOptions{
					SMTPHost:     conf.EmailSMTPHost,
					SMTPPort:     conf.EmailSMTPPort,
					SMTPUsername: conf.EmailSMTPUsername,
					SMTPPassword: conf.EmailSMTPPassword,
					FromEmail:    conf.EmailSenderEmail,
					FromName:     conf.EmailSenderName,
					BCC:          conf.EmailBCC,
				})
			} else {
				sender, err = email.NewConsoleSender(logger, conf.EmailSenderEmail, conf.EmailSenderName)
			}
			if err != nil {
				logger.Fatal("error creating email sender", zap.Error(err))
			}
			emailClient := email.New(sender)

			// Parse session keys as hex strings
			keyPairs := make([][]byte, len(conf.SessionKeyPairs))
			for idx, keyHex := range conf.SessionKeyPairs {
				key, err := hex.DecodeString(keyHex)
				if err != nil {
					logger.Fatal("failed to parse session key from hex string to bytes")
				}
				keyPairs[idx] = key
			}

			// Init observability
			shutdown, err := observability.Start(cmd.Context(), logger, &observability.Options{
				MetricsExporter: conf.MetricsExporter,
				TracesExporter:  conf.TracesExporter,
				ServiceName:     "runtime-server",
				ServiceVersion:  ch.Version.String(),
			})
			if err != nil {
				logger.Fatal("error starting observability", zap.Error(err))
			}
			defer func() {
				// Allow 10 seconds to gracefully shutdown observability
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				err := shutdown(ctx)
				if err != nil {
					logger.Error("observability shutdown failed", zap.Error(err))
				}
			}()

			// Init activity client
			var activityClient *activity.Client
			switch conf.ActivitySinkType {
			case "", "noop":
				activityClient = activity.NewNoopClient()
			case "kafka":
				sink, err := activity.NewKafkaSink(conf.ActivitySinkKafkaBrokers, conf.ActivitySinkKafkaTopic, logger)
				if err != nil {
					logger.Fatal("error creating kafka sink", zap.Error(err))
				}
				activityClient = activity.NewClient(sink, logger)
			case "console":
				sink := activity.NewLoggerSink(logger, zapcore.InfoLevel)
				activityClient = activity.NewClient(sink, logger)
			default:
				logger.Fatal("unknown activity sink type", zap.String("type", conf.ActivitySinkType))
			}
			defer activityClient.Close(context.Background())

			// Add service info to the activity client
			activityClient = activityClient.WithServiceName("runtime-server")
			if ch.Version.Number != "" || ch.Version.Commit != "" {
				activityClient = activityClient.WithServiceVersion(ch.Version.Number, ch.Version.Commit)
			}
			if ch.Version.IsDev() {
				activityClient = activityClient.WithIsDev()
			}

			// storage client
			var bucketConfig map[string]interface{}
			if conf.DataBucket != "" {
				bucketConfig = map[string]interface{}{
					"bucket":                              conf.DataBucket,
					"google_application_credentials_json": conf.DataBucketCredentialsJSON,
				}
			}
			storage, err := storage.New(conf.DataDir, bucketConfig)
			if err != nil {
				logger.Fatal("error: could not create storage client", zap.Error(err))
			}

			// Create ctx that cancels on termination signals
			ctx := graceful.WithCancelOnTerminate(context.Background())
			// Init runtime
			metastoreConfig, err := structpb.NewStruct(map[string]any{
				"dsn": conf.MetastoreURL,
				"id":  conf.MetastoreID,
			})
			if err != nil {
				logger.Fatal("error: could not creat metastore metastore config", zap.Error(err))
			}
			opts := &runtime.Options{
				ConnectionCacheSize:          conf.ConnectionCacheSize,
				MetastoreConnector:           "metastore",
				QueryCacheSizeBytes:          conf.QueryCacheSizeBytes,
				SecurityEngineCacheSize:      conf.SecurityEngineCacheSize,
				ControllerLogBufferCapacity:  conf.LogBufferCapacity,
				ControllerLogBufferSizeBytes: conf.LogBufferSizeBytes,
				AllowHostAccess:              conf.AllowHostAccess,
				SystemConnectors: []*runtimev1.Connector{
					{
						Type:   conf.MetastoreDriver,
						Name:   "metastore",
						Config: metastoreConfig,
					},
				},
				Version:              ch.Version,
				EnableConfigReloader: true,
			}
			rt, err := runtime.New(ctx, opts, logger, storage, activityClient, emailClient)
			if err != nil {
				logger.Fatal("error: could not create runtime", zap.Error(err))
			}
			defer rt.Close()

			var limiter ratelimit.Limiter
			if conf.RedisURL == "" {
				limiter = ratelimit.NewNoop()
			} else {
				opts, err := redis.ParseURL(conf.RedisURL)
				if err != nil {
					logger.Fatal("failed to parse redis url", zap.Error(err))
				}
				limiter = ratelimit.NewRedis(redis.NewClient(opts))
			}

			// Init server
			externalURL := conf.ExternalURL
			if externalURL == "" {
				externalURL = fmt.Sprintf("http://localhost:%d", conf.HTTPPort)
			}
			srvOpts := &server.Options{
				HTTPPort:        conf.HTTPPort,
				GRPCPort:        conf.GRPCPort,
				AllowedOrigins:  conf.AllowedOrigins,
				ServePrometheus: conf.MetricsExporter == observability.PrometheusExporter,
				SessionKeyPairs: keyPairs,
				AuthEnable:      conf.AuthEnable,
				AuthIssuerURL:   conf.AuthIssuerURL,
				AuthAudienceURL: conf.AuthAudienceURL,
				ExternalURL:     externalURL,
				Auth:            conf.Auth,
			}
			s, err := server.NewServer(ctx, srvOpts, rt, logger, limiter, activityClient)
			if err != nil {
				logger.Fatal("error: could not create server", zap.Error(err))
			}

			// Run server
			group, cctx := errgroup.WithContext(ctx)
			group.Go(func() error { return s.ServeHTTP(cctx, nil, false) })
			if conf.DebugPort != 0 {
				group.Go(func() error { return debugserver.ServeHTTP(cctx, conf.DebugPort) })
			}
			err = group.Wait()
			if err != nil && !errors.Is(err, context.Canceled) {
				logger.Error("server crashed", zap.Error(err))
				return
			}

			logger.Info("server shutdown gracefully")
		},
	}
	return startCmd
}
