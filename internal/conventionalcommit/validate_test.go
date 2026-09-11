package conventionalcommit

import "testing"

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		msg     string
		types   []string
		wantOK  bool
	}{
		{"valid feat", "feat: add login endpoint", nil, true},
		{"valid fix with scope", "fix(auth): handle expired tokens", nil, true},
		{"valid breaking change", "feat(api)!: remove deprecated field", nil, true},
		{"valid with body", "fix: correct off-by-one error\n\nThis fixes issue #42.", nil, true},
		{"missing colon", "feat add login endpoint", nil, false},
		{"unknown type", "wip: work in progress", nil, false},
		{"empty description", "feat: ", nil, false},
		{"trailing period", "fix: correct the bug.", nil, false},
		{"missing blank line before body", "fix: correct bug\nsee also #1", nil, false},
		{"empty message", "", nil, false},
		{"custom types allow it", "task: do something", []string{"task"}, true},
		{"custom types reject default", "feat: add thing", []string{"task"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Validate(tt.msg, tt.types)
			if got.Valid != tt.wantOK {
				t.Errorf("Validate(%q) valid = %v, errors = %v; want %v", tt.msg, got.Valid, got.Errors, tt.wantOK)
			}
		})
	}
}
