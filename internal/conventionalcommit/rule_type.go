package conventionalcommit

import (
	"fmt"
	"slices"
	"strings"

	"git-commit-sentinel/internal/config"
)

func init() {
	Register(ruleSpec{
		name:         "type",
		description:  "the commit type is one of the configured allowed types",
		defaultLevel: config.LevelError,
		check: func(ctx Context) (bool, string) {
			if !ctx.Parsed {
				return false, ""
			}
			allowed := ctx.Cfg.AllowedTypes()
			if slices.Contains(allowed, ctx.Type) {
				return false, ""
			}
			return true, fmt.Sprintf("type %q is not in the allowed list: %s", ctx.Type, strings.Join(allowed, ", "))
		},
	})
}
