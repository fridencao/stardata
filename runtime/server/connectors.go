package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	runtimev1 "github.com/fridencao/stardata/proto/gen/stardata/runtime/v1"
	"github.com/fridencao/stardata/runtime"
	"github.com/fridencao/stardata/runtime/drivers"
	"github.com/fridencao/stardata/runtime/parser"
	"github.com/fridencao/stardata/runtime/server/auth"
	"github.com/fridencao/stardata/runtime/storage"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/types/known/structpb"
)

func (s *Server) ListConnectorDrivers(ctx context.Context, req *runtimev1.ListConnectorDriversRequest) (*runtimev1.ListConnectorDriversResponse, error) {
	var pbs []*runtimev1.ConnectorDriver
	for name, driver := range drivers.Connectors {
		spec := driver.Spec()
		pbs = append(pbs, driverSpecToPB(name, &spec))
	}
	return &runtimev1.ListConnectorDriversResponse{Connectors: pbs}, nil
}

func (s *Server) AnalyzeConnectors(ctx context.Context, req *runtimev1.AnalyzeConnectorsRequest) (*runtimev1.AnalyzeConnectorsResponse, error) {
	if !auth.GetClaims(ctx, req.InstanceId).Can(runtime.ReadInstance) {
		return nil, ErrForbidden
	}

	inst, err := s.runtime.Instance(ctx, req.InstanceId)
	if err != nil {
		return nil, err
	}

	repo, release, err := s.runtime.Repo(ctx, req.InstanceId)
	if err != nil {
		return nil, err
	}
	defer release()

	p, err := parser.Parse(ctx, repo, req.InstanceId, inst.Environment, inst.OLAPConnector, true)
	if err != nil {
		return nil, err
	}

	connectors := p.AnalyzeConnectors(ctx)

	res := make(map[string]*runtimev1.AnalyzedConnector)

	for _, connector := range connectors {
		if connector.Err != nil {
			res[connector.Name] = &runtimev1.AnalyzedConnector{
				Name:         connector.Name,
				ErrorMessage: connector.Err.Error(),
			}
			continue
		}

		cfg, err := s.runtime.ConnectorConfig(ctx, req.InstanceId, connector.Name)
		if err != nil {
			res[connector.Name] = &runtimev1.AnalyzedConnector{
				Name:         connector.Name,
				ErrorMessage: err.Error(),
			}
			continue
		}

		var provisionArgsPB *structpb.Struct
		if len(cfg.ProvisionArgs) > 0 {
			provisionArgsPB, err = structpb.NewStruct(cfg.ProvisionArgs)
			if err != nil {
				return nil, err
			}
		}

		cfgConfig := cfg.Resolve()
		var cfgConfigPB *structpb.Struct
		if len(cfgConfig) > 0 {
			cfgConfigPB, err = structpb.NewStruct(cfgConfig)
			if err != nil {
				return nil, err
			}
		}

		var presetConfigPB *structpb.Struct
		if len(cfg.Preset) > 0 {
			presetConfigPB, err = structpb.NewStruct(cfg.Preset)
			if err != nil {
				return nil, err
			}
		}

		projectConfig := connector.DefaultConfig
		var projectConfigPB *structpb.Struct
		if len(projectConfig) > 0 {
			projectConfigPB, err = structpb.NewStruct(projectConfig)
			if err != nil {
				return nil, err
			}
		}

		c := &runtimev1.AnalyzedConnector{
			Name:               connector.Name,
			Driver:             driverSpecToPB(connector.Driver, connector.Spec),
			Config:             cfgConfigPB,
			PresetConfig:       presetConfigPB,
			ProjectConfig:      projectConfigPB, // NOTE: Could also use cfg.Project, but connector.DefaultConfig might be slightly more up-to-date
			EnvConfig:          cfg.Env,
			Provision:          cfg.Provision,
			ProvisionArgs:      provisionArgsPB,
			HasAnonymousAccess: connector.AnonymousAccess,
			UsedBy:             nil,
		}

		for _, r := range connector.Resources {
			c.UsedBy = append(c.UsedBy, runtime.ResourceNameFromParser(r.Name))
		}

		res[connector.Name] = c
	}

	return &runtimev1.AnalyzeConnectorsResponse{
		Connectors: maps.Values(res),
	}, nil
}

func (s *Server) ListNotifierConnectors(ctx context.Context, req *runtimev1.ListNotifierConnectorsRequest) (*runtimev1.ListNotifierConnectorsResponse, error) {
	if !auth.GetClaims(ctx, req.InstanceId).Can(runtime.ReadObjects) {
		return nil, ErrForbidden
	}

	inst, err := s.runtime.Instance(ctx, req.InstanceId)
	if err != nil {
		return nil, err
	}

	res := make(map[string]*runtimev1.Connector)

	for _, c := range inst.Connectors {
		if driverIsNotifier(c.Type) {
			res[c.Name] = &runtimev1.Connector{
				Type: c.Type,
				Name: c.Name,
			}
		}
	}

	for _, c := range inst.ProjectConnectors {
		if driverIsNotifier(c.Type) {
			res[c.Name] = &runtimev1.Connector{
				Type: c.Type,
				Name: c.Name,
			}
		}
	}

	// Connectors may be implicitly defined just by adding variables in the format "connector.<name>.<property>".
	// NOTE: We can remove this if we move to explicitly defined connectors.
	for k := range inst.ResolveVariables(true) {
		if !strings.HasPrefix(k, "connector.") {
			continue
		}

		parts := strings.Split(k, ".")
		if len(parts) <= 2 {
			continue
		}

		// Implicitly defined connectors always have the same name as the driver
		name := parts[1]
		if driverIsNotifier(name) {
			res[name] = &runtimev1.Connector{
				Type: name,
				Name: name,
			}
		}
	}

	return &runtimev1.ListNotifierConnectorsResponse{
		Connectors: maps.Values(res),
	}, nil
}

func driverSpecToPB(name string, spec *drivers.Spec) *runtimev1.ConnectorDriver {
	pb := &runtimev1.ConnectorDriver{
		Name:                  name,
		ConfigProperties:      nil,
		SourceProperties:      nil,
		DisplayName:           spec.DisplayName,
		Description:           spec.Description,
		DocsUrl:               spec.DocsURL,
		ImplementsRegistry:    spec.ImplementsRegistry,
		ImplementsCatalog:     spec.ImplementsCatalog,
		ImplementsRepo:        spec.ImplementsRepo,
		ImplementsAdmin:       spec.ImplementsAdmin,
		ImplementsAi:          spec.ImplementsAI,
		ImplementsSqlStore:    spec.ImplementsSQLStore,
		ImplementsOlap:        spec.ImplementsOLAP,
		ImplementsObjectStore: spec.ImplementsObjectStore,
		ImplementsFileStore:   spec.ImplementsFileStore,
		ImplementsNotifier:    spec.ImplementsNotifier,
		ImplementsWarehouse:   spec.ImplementsWarehouse,
	}

	for _, prop := range spec.ConfigProperties {
		pb.ConfigProperties = append(pb.ConfigProperties, driverPropertySpecToPB(prop))
	}

	for _, prop := range spec.SourceProperties {
		pb.SourceProperties = append(pb.SourceProperties, driverPropertySpecToPB(prop))
	}

	return pb
}

func driverPropertySpecToPB(spec *drivers.PropertySpec) *runtimev1.ConnectorDriver_Property {
	var t runtimev1.ConnectorDriver_Property_Type
	switch spec.Type {
	case drivers.NumberPropertyType:
		t = runtimev1.ConnectorDriver_Property_TYPE_NUMBER
	case drivers.BooleanPropertyType:
		t = runtimev1.ConnectorDriver_Property_TYPE_BOOLEAN
	case drivers.StringPropertyType:
		t = runtimev1.ConnectorDriver_Property_TYPE_STRING
	case drivers.FilePropertyType:
		t = runtimev1.ConnectorDriver_Property_TYPE_FILE
	case drivers.InformationalPropertyType:
		t = runtimev1.ConnectorDriver_Property_TYPE_INFORMATIONAL
	}

	return &runtimev1.ConnectorDriver_Property{
		Key:         spec.Key,
		Type:        t,
		Required:    spec.Required,
		DisplayName: spec.DisplayName,
		Description: spec.Description,
		DocsUrl:     spec.DocsURL,
		Hint:        spec.Hint,
		Default:     spec.Default,
		Placeholder: spec.Placeholder,
		Secret:      spec.Secret,
		NoPrompt:    spec.NoPrompt,
	}
}

func driverIsNotifier(driver string) bool {
	connector, ok := drivers.Connectors[driver]
	if !ok {
		return false
	}

	return connector.Spec().ImplementsNotifier
}

// TestConnectionHandler tests a data source connection without saving it.
// POST /v1/instances/{instance_id}/connectors:testconnection
// Body: {"driver":"clickhouse","config":{"host":"...","port":"9000",...}}
// This is a raw HTTP handler (bypasses gRPC/proto) wrapped with auth middleware.
func (s *Server) TestConnectionHandler(w http.ResponseWriter, r *http.Request) {
	instanceID := r.PathValue("instance_id")

	// Auth check: require ReadInstance permission.
	claims := auth.GetClaims(r.Context(), instanceID)
	if !claims.Can(runtime.ReadInstance) {
		writeJSONError(w, http.StatusForbidden, "access denied")
		return
	}

	var req struct {
		Driver string         `json:"driver"`
		Config map[string]any `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %s", err))
		return
	}

	if req.Driver == "" {
		writeJSONError(w, http.StatusBadRequest, "driver is required")
		return
	}

	// Look up the driver.
	driver, ok := drivers.Connectors[req.Driver]
	if !ok {
		writeJSONResult(w, false, fmt.Sprintf("driver %q not found", req.Driver))
		return
	}

	// Create a minimal storage client (some drivers like DuckDB need it for Open).
	st, stErr := storage.New(os.TempDir(), nil)
	if stErr != nil {
		writeJSONResult(w, false, fmt.Sprintf("failed to create storage client: %s", stErr))
		return
	}

	// Open a connection with the provided config (not saved anywhere).
	// Wrapped in recover because some drivers may panic on nil dependencies.
	var handle drivers.Handle
	var openErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				openErr = fmt.Errorf("driver panicked: %v", r)
			}
		}()
		handle, openErr = driver.Open("test-connection", instanceID, req.Config, st, nil, s.logger)
	}()
	if openErr != nil {
		writeJSONResult(w, false, fmt.Sprintf("failed to open connection: %s", openErr))
		return
	}
	defer handle.Close()

	// Ping with a timeout.
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	if err := handle.Ping(ctx); err != nil {
		writeJSONResult(w, false, fmt.Sprintf("connection test failed: %s", err))
		return
	}

	writeJSONResult(w, true, "connection successful")
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": message})
}

func writeJSONResult(w http.ResponseWriter, ok bool, message string) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": ok, "message": message})
}
