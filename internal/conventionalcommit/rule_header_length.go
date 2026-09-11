package conventionalcommit

import (
	"fmt"
	"unicode/utf8"

	"git-commit-sentinel/internal/config"
)

func init() {
	Register(ruleSpec{
		name:         "header-length",
		description:  "the header does not exceed headerMaxLength characters",
		defaultLevel: config.LevelWarn,
		// Counts runes, not bytes (utf8.RuneCountInString): len() would
		// over-count multi-byte characters.
		check: func(ctx Context) (bool, string) {
			max := ctx.Cfg.MaxHeaderLength()
			length := utf8.RuneCountInString(ctx.Header)
			if length <= max {
				return false, ""
			}
			return true, fmt.Sprintf("header is %d characters long, exceeds the limit of %d", length, max)
		},
	})
}
