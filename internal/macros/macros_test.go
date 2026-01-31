package macros

import (
	"strings"
	"testing"
)

func TestFixContent(t *testing.T) {
	if FixContent("") == "" {
		t.Error("empty topic should return usage")
	}
	if !strings.Contains(FixContent("gas"), "APT") {
		t.Error("gas macro should mention APT")
	}
	if !strings.Contains(FixContent(TopicScam), "never DM") {
		t.Error("scam macro should mention never DM")
	}
	if FixContent("unknown_topic") == FixContent(TopicGas) {
		t.Error("unknown topic should not return gas content")
	}
}

func TestAllFixTopics(t *testing.T) {
	topics := AllFixTopics()
	if len(topics) != 4 {
		t.Errorf("expected 4 topics, got %d", len(topics))
	}
	seen := make(map[string]bool)
	for _, t := range topics {
		seen[t] = true
	}
	for _, name := range []string{TopicGas, TopicStaking, TopicSwitchRPC, TopicScam} {
		if !seen[name] {
			t.Errorf("missing topic %q", name)
		}
	}
}
