package conventionalcommit

import (
	"fmt"

	"git-commit-sentinel/internal/config"
)

func init() {
	Register(ruleSpec{
		name:         "format",
		description:  "the header matches <type>(<scope>)!: <description>",
		defaultLevel: config.LevelError,
		check: func(ctx Context) (bool, string) {
			if ctx.Parsed {
				return false, ""
			}
			return true, fmt.Sprintf(
				"header %q does not match the expected format: <type>(<scope>)!: <description>",
				ctx.Header,
			)
		},
	})
}
