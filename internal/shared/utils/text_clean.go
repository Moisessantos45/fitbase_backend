package utils

import (
	"html"
	"regexp"
	"strings"
	"unicode"
)

var (
	reFrontMatter = regexp.MustCompile(`(?ms)^---\s*\n.*?\n---\s*\n?`)
	reCodeFence   = regexp.MustCompile("(?ms)```.*?```|~~~.*?~~~")
	reInlineCode  = regexp.MustCompile("`([^`]*)`")
	reImageLink   = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	reMdLink      = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	reAutoURL     = regexp.MustCompile(`https?://[^\s)]+|www\.[^\s)]+`)
	reHTMLTags    = regexp.MustCompile(`(?s)<[^>]*>`)
	reHeader      = regexp.MustCompile(`(?m)^\s{0,3}#{1,6}\s*`)
	reBlockquote  = regexp.MustCompile(`(?m)^\s{0,3}>\s?`)
	reULItem      = regexp.MustCompile(`(?m)^\s*[-*+]\s+`)
	reOLItem      = regexp.MustCompile(`(?m)^\s*\d+\.\s+`)
	reHr          = regexp.MustCompile(`(?m)^\s*([-*_]\s*){3,}\s*$`)
	reStrike      = regexp.MustCompile(`~~(.*?)~~`)
	reTablePipes  = regexp.MustCompile(`\|`)
	reRefLinks    = regexp.MustCompile(`(?m)^\s*\[[^\]]+\]:\s+\S+\s*$`)
	reMultiSpace  = regexp.MustCompile(`[ \t]+`)
	reMultiNL     = regexp.MustCompile(`\n{3,}`)
	reMdSymbols   = regexp.MustCompile(`[>#*_~]+`)

	reBoldStar    = regexp.MustCompile(`\*\*([^*\n]+?)\*\*`)
	reBoldUnder   = regexp.MustCompile(`__([^_\n]+?)__`)
	reItalicStar  = regexp.MustCompile(`\*([^*\n]+?)\*`)
	reItalicUnder = regexp.MustCompile(`_([^_\n]+?)_`)
)

func CleanMarkdownForSearch(input string, keepCode bool) string {
	s := strings.ReplaceAll(input, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	s = reFrontMatter.ReplaceAllString(s, "\n")
	s = html.UnescapeString(s)

	if keepCode {
		s = reCodeFence.ReplaceAllStringFunc(s, func(m string) string {
			m = strings.TrimSpace(m)
			lines := strings.Split(m, "\n")
			if len(lines) <= 1 {
				return "\n"
			}

			first := strings.TrimSpace(lines[0])
			if strings.HasPrefix(first, "```") || strings.HasPrefix(first, "~~~") {
				lines = lines[1:]
			}

			if n := len(lines); n > 0 {
				last := strings.TrimSpace(lines[n-1])
				if last == "```" || last == "~~~" {
					lines = lines[:n-1]
				}
			}

			return "\n" + strings.Join(lines, "\n") + "\n"
		})

		s = reInlineCode.ReplaceAllString(s, "$1")
	} else {
		s = reCodeFence.ReplaceAllString(s, "\n")
		s = reInlineCode.ReplaceAllString(s, " ")
	}

	s = reImageLink.ReplaceAllString(s, " $1 ")
	s = reMdLink.ReplaceAllString(s, " $1 ")
	s = reRefLinks.ReplaceAllString(s, " ")

	s = reAutoURL.ReplaceAllString(s, " ")
	s = reHTMLTags.ReplaceAllString(s, " ")

	s = reHeader.ReplaceAllString(s, "")
	s = reBlockquote.ReplaceAllString(s, "")
	s = reULItem.ReplaceAllString(s, "")
	s = reOLItem.ReplaceAllString(s, "")
	s = reHr.ReplaceAllString(s, "\n")
	s = reStrike.ReplaceAllString(s, "$1")

	s = reBoldStar.ReplaceAllString(s, "$1")
	s = reBoldUnder.ReplaceAllString(s, "$1")
	s = reItalicStar.ReplaceAllString(s, "$1")
	s = reItalicUnder.ReplaceAllString(s, "$1")

	s = reTablePipes.ReplaceAllString(s, " ")
	s = reMdSymbols.ReplaceAllString(s, " ")

	lines := strings.Split(s, "\n")
	cleaned := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.Map(func(r rune) rune {
			if unicode.IsControl(r) && r != '\n' && r != '\t' {
				return -1
			}
			return r
		}, line)

		line = reMultiSpace.ReplaceAllString(strings.TrimSpace(line), " ")
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}

	s = strings.Join(cleaned, "\n")
	s = reMultiNL.ReplaceAllString(s, "\n\n")
	s = strings.TrimSpace(s)

	return s
}
