package conventionalcommit

import (
	"strings"

	"git-commit-sentinel/internal/config"
)

func init() {
	Register(ruleSpec{
		name:         "description-period",
		description:  "the description does not end with a period",
		defaultLevel: config.LevelWarn,
		check: func(ctx Context) (bool, string) {
			if !ctx.Parsed || !strings.HasSuffix(ctx.Description, ".") {
				return false, ""
			}
			return true, "description must not end with a period"
		},
	})
}
