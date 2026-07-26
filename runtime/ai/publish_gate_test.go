package ai

import (
	"testing"
)

func TestPublishGateParse(t *testing.T) {
	cases := []struct {
		name      string
		data      string
		wantGated bool
		wantHas   []string
		wantMiss  []string
	}{
		{name: "normal list", data: "published:\n  - sales_metrics\n  - orders_metrics\n", wantGated: true, wantHas: []string{"sales_metrics", "orders_metrics"}, wantMiss: []string{"other"}},
		{name: "empty file", data: "", wantGated: false},
		{name: "empty list", data: "published: []\n", wantGated: false},
		{name: "null list", data: "published:\n", wantGated: false},
		{name: "invalid yaml", data: "published: [unclosed\n", wantGated: false},
		{name: "unrelated keys only", data: "foo: bar\n", wantGated: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			published, gated := parsePublishedList(c.data)
			if gated != c.wantGated {
				t.Fatalf("gated = %v, want %v", gated, c.wantGated)
			}
			for _, n := range c.wantHas {
				if !published[n] {
					t.Errorf("expected %q to be published", n)
				}
			}
			for _, n := range c.wantMiss {
				if published[n] {
					t.Errorf("expected %q to NOT be published", n)
				}
			}
		})
	}
}
