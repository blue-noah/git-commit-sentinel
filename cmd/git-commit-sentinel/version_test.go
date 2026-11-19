package main

import (
	"runtime/debug"
	"testing"
)

func TestResolveVersion(t *testing.T) {
	buildInfo := func(v string) *debug.BuildInfo {
		return &debug.BuildInfo{Main: debug.Module{Path: "github.com/blue-noah/git-commit-sentinel", Version: v}}
	}

	tests := []struct {
		name   string
		linked string
		info   *debug.BuildInfo
		want   string
	}{
		{"release build: -X wins over build info", "0.0.6", buildInfo("v0.0.6-0.20261103131323-30efcfc27136"), "0.0.6"},
		{"go install @tag", "dev", buildInfo("v0.0.6"), "0.0.6"},
		{"go install @main or go build from a clone", "dev", buildInfo("v0.0.6-0.20261103131323-30efcfc27136"), "0.0.6-0.20261103131323-30efcfc27136"},
		{"go build with uncommitted changes", "dev", buildInfo("v0.0.6-0.20261103131323-30efcfc27136+dirty"), "0.0.6-0.20261103131323-30efcfc27136+dirty"},
		{"go run or go test", "dev", buildInfo("(devel)"), "dev"},
		{"empty module version", "dev", buildInfo(""), "dev"},
		{"no build info", "dev", nil, "dev"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveVersion(tt.linked, tt.info); got != tt.want {
				t.Errorf("resolveVersion(%q, ...) = %q, want %q", tt.linked, got, tt.want)
			}
		})
	}
}
