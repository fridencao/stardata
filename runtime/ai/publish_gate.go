package ai

import (
	"context"

	"github.com/fridencao/stardata/runtime"
	"gopkg.in/yaml.v3"
)

// publishFilePath is the project-relative path of the StarData publish gate file.
const publishFilePath = "/publish.yaml"

type publishYAML struct {
	Published []string `yaml:"published"`
}

// parsePublishedList parses the contents of publish.yaml.
// gated=false when the list is empty or the file fails to parse
// (parse failures must never block Q&A).
func parsePublishedList(data string) (map[string]bool, bool) {
	var p publishYAML
	if err := yaml.Unmarshal([]byte(data), &p); err != nil {
		return nil, false
	}
	if len(p.Published) == 0 {
		return nil, false
	}
	published := make(map[string]bool, len(p.Published))
	for _, name := range p.Published {
		published[name] = true
	}
	return published, true
}

// publishedMetricsViews reads /publish.yaml from the project repo.
// gated=false (allow all) when the file does not exist or cannot be read.
func publishedMetricsViews(ctx context.Context, rt *runtime.Runtime, instanceID string) (map[string]bool, bool) {
	repo, release, err := rt.Repo(ctx, instanceID)
	if err != nil {
		return nil, false
	}
	defer release()
	data, err := repo.Get(ctx, publishFilePath)
	if err != nil {
		return nil, false
	}
	return parsePublishedList(data)
}
