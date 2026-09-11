package conventionalcommit

import "testing"

func TestCloneForRuleIsolatesLines(t *testing.T) {
	original := Context{Lines: []string{"a", "b"}}

	clone := original.cloneForRule()
	clone.Lines[0] = "mutated"

	if original.Lines[0] != "a" {
		t.Errorf("mutating clone.Lines[0] affected the original: got %q, want \"a\"", original.Lines[0])
	}
}
