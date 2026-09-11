package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git-commit-sentinel/internal/hookfile"
)

func TestCheckHookFileManaged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "commit-msg")
	if err := os.WriteFile(path, []byte(hookfile.Render()), 0o755); err != nil {
		t.Fatal(err)
	}
	r := checkHookFile(path)
	if r.Level != levelOK {
		t.Errorf("checkHookFile(managed) = %+v, want level OK", r)
	}
}

func TestCheckHookFileForeign(t *testing.T) {
	path := filepath.Join(t.TempDir(), "commit-msg")
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho hi\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r := checkHookFile(path)
	if r.Level != levelWarn {
		t.Errorf("checkHookFile(foreign) = %+v, want level WARN", r)
	}
}

func TestCheckHookFileMissing(t *testing.T) {
	r := checkHookFile(filepath.Join(t.TempDir(), "does-not-exist"))
	if r.Level != levelFail {
		t.Errorf("checkHookFile(missing) = %+v, want level FAIL", r)
	}
}

func TestCheckHookFileNotExecutable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "commit-msg")
	if err := os.WriteFile(path, []byte(hookfile.Render()), 0o644); err != nil {
		t.Fatal(err)
	}
	r := checkHookFile(path)
	if r.Level != levelFail {
		t.Errorf("checkHookFile(not executable) = %+v, want level FAIL", r)
	}
}

func TestCheckBinaryOnPathMissing(t *testing.T) {
	original := lookPath
	lookPath = func(string) (string, error) { return "", errors.New("not found") }
	defer func() { lookPath = original }()

	results := checkBinaryOnPath()
	if len(results) != 1 || results[0].Level != levelFail {
		t.Errorf("checkBinaryOnPath() = %+v, want a single FAIL", results)
	}
}

func TestCheckBinaryOnPathFound(t *testing.T) {
	original := lookPath
	lookPath = func(string) (string, error) { return "/usr/local/bin/git-commit-sentinel", nil }
	defer func() { lookPath = original }()

	results := checkBinaryOnPath()
	if len(results) != 1 || results[0].Level != levelOK {
		t.Errorf("checkBinaryOnPath() = %+v, want a single OK", results)
	}
}

func TestPrintReportHealthy(t *testing.T) {
	var buf bytes.Buffer
	healthy := printReport(&buf, []section{
		{"a", []checkResult{ok("fine")}},
		{"b", []checkResult{warn("meh")}},
	})
	if !healthy {
		t.Error("printReport() healthy = false, want true (no FAIL results)")
	}
	if !strings.Contains(buf.String(), "[WARN] meh") {
		t.Errorf("printReport() output = %q, want it to contain the warning", buf.String())
	}
}

func TestPrintReportUnhealthy(t *testing.T) {
	var buf bytes.Buffer
	healthy := printReport(&buf, []section{
		{"a", []checkResult{fail("broken")}},
	})
	if healthy {
		t.Error("printReport() healthy = true, want false (a FAIL result is present)")
	}
}
