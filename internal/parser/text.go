// Package parser provides utilities for extracting clean plain text from raw
// HTML content retrieved from Wikipedia pages.  It uses only the Go standard
// library so that the module has zero external dependencies.
package parser

import (
	"regexp"
	"strings"
	"unicode"
)

// Pre-compiled regexps used by StripHTML.  Compiling once at package init is
// far cheaper than recompiling on every call.
var (
	scriptOrStyle    = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	htmlTag          = regexp.MustCompile(`<[^>]+>`)
	htmlEntity       = regexp.MustCompile(`&[a-zA-Z0-9#]+;`)
	multipleSpaces   = regexp.MustCompile(`[ \t]{2,}`)
	multipleNewlines = regexp.MustCompile(`\n{3,}`)
)

// htmlEntities maps the most common HTML character entities to their UTF-8
// equivalents.
var htmlEntities = map[string]string{
	"&amp;":    "&",
	"&lt;":     "<",
	"&gt;":     ">",
	"&quot;":   `"`,
	"&apos;":   "'",
	"&nbsp;":   " ",
	"&ndash;":  "–",
	"&mdash;":  "—",
	"&hellip;": "…",
	"&copy;":   "©",
	"&reg;":    "®",
}

// StripHTML removes all HTML markup from raw HTML content and returns plain
// text suitable for storage in the knowledge base.
func StripHTML(raw string) string {
	// Remove script and style blocks with their inner content.
	clean := scriptOrStyle.ReplaceAllString(raw, " ")

	// Replace block-level closing tags with newlines to preserve paragraph breaks.
	blockTags := []string{
		"</p>", "</div>", "</h1>", "</h2>", "</h3>",
		"</h4>", "</h5>", "</li>", "</tr>", "<br>", "<br/>",
	}
	for _, tag := range blockTags {
		clean = strings.ReplaceAll(clean, tag, "\n")
		clean = strings.ReplaceAll(clean, strings.ToUpper(tag), "\n")
	}

	// Strip remaining HTML tags.
	clean = htmlTag.ReplaceAllString(clean, " ")

	// Decode common HTML entities.
	clean = htmlEntity.ReplaceAllStringFunc(clean, func(entity string) string {
		if decoded, ok := htmlEntities[strings.ToLower(entity)]; ok {
			return decoded
		}
		return " "
	})

	return CleanText(clean)
}

// CleanText normalises raw text by collapsing excess whitespace and removing
// non-printable control characters.
func CleanText(raw string) string {
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' && r != '\r' {
			return -1
		}
		return r
	}, raw)

	cleaned = strings.ReplaceAll(cleaned, "\r\n", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\r", "\n")
	cleaned = multipleSpaces.ReplaceAllString(cleaned, " ")
	cleaned = multipleNewlines.ReplaceAllString(cleaned, "\n\n")

	return strings.TrimSpace(cleaned)
}

// IsUsefulParagraph returns true when the text contains at least 5 words,
// filtering out short navigation labels and image captions.
func IsUsefulParagraph(text string) bool {
	const minWordCount = 5
	return len(strings.Fields(text)) >= minWordCount
}
