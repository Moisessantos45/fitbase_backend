package utils

import (
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/text/unicode/norm"
)

func NormalizeText(s string) string {
	nfd := norm.NFD.String(s)
	var b strings.Builder

	for _, r := range nfd {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(r)
	}
	withoutAccents := b.String()

	lower := cases.Lower(language.Spanish).String(withoutAccents)

	return strings.TrimSpace(lower)
}
