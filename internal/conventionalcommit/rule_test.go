package conventionalcommit

import (
	"testing"

	"git-commit-sentinel/internal/config"
)

func TestEveryRuleHasSaneMetadata(t *testing.T) {
	for _, r := range Rules() {
		if r.Name() == "" {
			t.Errorf("rule %+v has an empty Name()", r)
		}
		if r.Description() == "" {
			t.Errorf("rule %q has an empty Description()", r.Name())
		}
		if !r.DefaultLevel().Valid() {
			t.Errorf("rule %q has an invalid DefaultLevel() = %q", r.Name(), r.DefaultLevel())
		}
	}
}

func TestRulesReturnsACopyOfTheRegistry(t *testing.T) {
	before := len(Rules())

	got := Rules()
	got[0] = ruleSpec{name: "corrupted", defaultLevel: config.LevelOff}

	after := len(Rules())
	if after != before {
		t.Fatalf("Rules() count changed from %d to %d after mutating a previous result", before, after)
	}
	for _, r := range Rules() {
		if r.Name() == "corrupted" {
			t.Fatal("mutating a slice returned by Rules() corrupted the shared registry")
		}
	}
}
