package version

import (
	"reflect"
	"testing"
)

func TestInfo(t *testing.T) {
	info := Info()
	if info["version"] == "" {
		t.Error("version should be set")
	}
	keys := []string{"version", "commit", "buildDate", "go"}
	for _, k := range keys {
		if _, ok := info[k]; !ok {
			t.Errorf("info missing key %q", k)
		}
	}
	if len(info) != len(keys) {
		t.Errorf("info has %d keys, want %d", len(info), len(keys))
	}
	// go version should look like "go1.x..."
	if info["go"] != "" && info["go"][:2] != "go" {
		t.Errorf("info[go] = %q", info["go"])
	}
	_ = reflect.DeepEqual(info, info)
}
