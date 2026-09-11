package hookfile

import (
	"strings"
	"testing"
)

func TestRenderIsManaged(t *testing.T) {
	content := Render()
	if !IsManaged(content) {
		t.Error("IsManaged(Render()) = false; want true")
	}
}

func TestRenderInvokesBinary(t *testing.T) {
	content := Render()
	want := `exec ` + BinaryName + ` hook "$1"`
	if !strings.Contains(content, want) {
		t.Errorf("Render() = %q; want it to contain %q", content, want)
	}
}

func TestIsManagedRejectsForeignHooks(t *testing.T) {
	foreign := "#!/bin/sh\nnpx husky run commit-msg \"$1\"\n"
	if IsManaged(foreign) {
		t.Error("IsManaged(foreign hook) = true; want false")
	}
}
