package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	runtimev1 "github.com/fridencao/stardata/proto/gen/stardata/runtime/v1"
	"gopkg.in/yaml.v3"
)

// publishGateFilePath is the project-relative path of the StarData publish gate file.
const publishGateFilePath = "/publish.yaml"

// publishGateTTL is how long a parsed publish gate is cached per instance.
// The gate only changes when a governor publishes, so a short TTL keeps the
// hot path (resource listing loops) cheap while bounding staleness.
const publishGateTTL = 2 * time.Second

// ErrNotPublished is returned when a resource is hidden by the publish gate.
var ErrNotPublished = errors.New("resource is not published")

// publishGateKinds are the resource kinds subject to the publish gate.
// Everything a business user can reach from the portal is covered: the semantic
// layer itself plus the two dashboard kinds built on top of it.
var publishGateKinds = map[string]bool{
	ResourceKindMetricsView: true,
	ResourceKindExplore:     true,
	ResourceKindCanvas:      true,
}

// PublishGate is the parsed contents of /publish.yaml.
type PublishGate struct {
	// Gated is false when the project has not opted into publish gating,
	// i.e. /publish.yaml is absent. In that case everything is visible.
	Gated bool
	// Published holds the metrics view names that have been published.
	Published map[string]bool
}

// Allows reports whether the named metrics view is visible through the gate.
func (g *PublishGate) Allows(metricsView string) bool {
	if g == nil || !g.Gated {
		return true
	}
	return g.Published[metricsView]
}

type publishGateEntry struct {
	gate     *PublishGate
	cachedAt time.Time
}

type publishGateCache struct {
	mu      sync.Mutex
	entries map[string]publishGateEntry
}

func newPublishGateCache() *publishGateCache {
	return &publishGateCache{entries: make(map[string]publishGateEntry)}
}

// PublishGate returns the publish gate for an instance.
//
// Unlike the previous AI-only implementation, a malformed /publish.yaml is an error
// rather than an implicit "allow all": once a project has opted into gating, failures
// must deny rather than leak unpublished content (fail-closed).
func (r *Runtime) PublishGate(ctx context.Context, instanceID string) (*PublishGate, error) {
	c := r.publishGates
	c.mu.Lock()
	if e, ok := c.entries[instanceID]; ok && time.Since(e.cachedAt) < publishGateTTL {
		c.mu.Unlock()
		return e.gate, nil
	}
	c.mu.Unlock()

	gate, err := r.loadPublishGate(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.entries[instanceID] = publishGateEntry{gate: gate, cachedAt: time.Now()}
	c.mu.Unlock()
	return gate, nil
}

func (r *Runtime) loadPublishGate(ctx context.Context, instanceID string) (*PublishGate, error) {
	repo, release, err := r.Repo(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("publish gate: failed to open repo: %w", err)
	}
	defer release()

	data, err := repo.Get(ctx, publishGateFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// The project has not opted into publish gating.
			return &PublishGate{}, nil
		}
		return nil, fmt.Errorf("publish gate: failed to read %s: %w", publishGateFilePath, err)
	}

	var doc struct {
		Published []string `yaml:"published"`
	}
	if err := yaml.Unmarshal([]byte(data), &doc); err != nil {
		return nil, fmt.Errorf("publish gate: %s is malformed: %w", publishGateFilePath, err)
	}

	gate := &PublishGate{Gated: true, Published: make(map[string]bool, len(doc.Published))}
	for _, name := range doc.Published {
		gate.Published[name] = true
	}
	return gate, nil
}

// CheckPublishGate reports whether the caller may see the given resource.
//
// The gate is scoped to the environment that serves business users: callers holding
// EditRepo (the Studio / dev editing context) are exempt so governors can work on
// drafts. Production deployments do not grant EditRepo, so production only ever
// serves published content — to business users and governors alike.
func (r *Runtime) CheckPublishGate(ctx context.Context, instanceID string, claims *SecurityClaims, res *runtimev1.Resource) (bool, error) {
	if res == nil {
		return true, nil
	}
	if claims == nil || claims.SkipChecks || claims.Can(EditRepo) {
		return true, nil
	}
	if !publishGateKinds[res.Meta.GetName().GetKind()] {
		return true, nil
	}

	gate, err := r.PublishGate(ctx, instanceID)
	if err != nil {
		return false, err
	}
	if !gate.Gated {
		return true, nil
	}

	for _, name := range publishGateSubjects(res) {
		if !gate.Allows(name) {
			return false, nil
		}
	}
	return true, nil
}

// publishGateSubjects returns the metrics view names that must be published for the
// resource to be visible. A metrics view gates on itself; dashboards gate on every
// metrics view they reference, so a dashboard is hidden until all of its data is published.
func publishGateSubjects(res *runtimev1.Resource) []string {
	name := res.Meta.GetName()
	if name.GetKind() == ResourceKindMetricsView {
		return []string{name.GetName()}
	}

	var subjects []string
	for _, ref := range res.Meta.GetRefs() {
		if ref.GetKind() == ResourceKindMetricsView {
			subjects = append(subjects, ref.GetName())
		}
	}
	return subjects
}
