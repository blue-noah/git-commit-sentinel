package conventionalcommit

import "git-commit-sentinel/internal/config"

func init() {
	Register(ruleSpec{
		name:         "description-empty",
		description:  "the description is not empty",
		defaultLevel: config.LevelError,
		check: func(ctx Context) (bool, string) {
			if !ctx.Parsed || ctx.Description != "" {
				return false, ""
			}
			return true, "description must not be empty"
		},
	})
}
