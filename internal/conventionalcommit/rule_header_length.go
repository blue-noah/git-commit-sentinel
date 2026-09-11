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
		// Ignores ctx.Parsed on purpose: an overlong header is worth
		// reporting whether or not it also happens to match the grammar.
		//
		// Counts runes, not bytes: len(ctx.Header) counts UTF-8 bytes, so a
		// header with accented letters, CJK characters or emoji would be
		// flagged as "too long" well before it reaches the limit a human
		// (or the user's editor) actually perceives.
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
