package conventionalcommit

import (
	"strings"

	"github.com/blue-noah/git-commit-sentinel/internal/config"
)

func init() {
	Register(ruleSpec{
		name:         "body-blank-line",
		description:  "a blank line separates the header from the body, if any",
		defaultLevel: config.LevelWarn,
		check: func(ctx Context) (bool, string) {
			if len(ctx.Lines) <= 1 || strings.TrimSpace(ctx.Lines[1]) == "" {
				return false, ""
			}
			return true, "second line must be blank to separate header from body"
		},
	})
}
