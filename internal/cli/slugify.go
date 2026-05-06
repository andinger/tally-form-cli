package cli

import (
	"strings"
	"unicode"
)

// slugify produces a filesystem-safe ASCII slug for a question label.
//
// German umlauts and ß are transliterated (ä→ae, ö→oe, ü→ue, ß→ss), the
// result is lowercased, all non-[a-z0-9-] runes are dropped, runs of "-" are
// collapsed, and the slug is truncated to max characters at the last word
// boundary (last "-") so words are not cut mid-syllable.
//
// Returns "" if no usable characters remain — callers should fall back to a
// stable identifier like the question ID.
func slugify(s string, max int) string {
	if max <= 0 {
		return ""
	}

	var b strings.Builder
	for _, r := range s {
		switch r {
		case 'ä', 'Ä':
			b.WriteString("ae")
		case 'ö', 'Ö':
			b.WriteString("oe")
		case 'ü', 'Ü':
			b.WriteString("ue")
		case 'ß':
			b.WriteString("ss")
		default:
			b.WriteRune(unicode.ToLower(r))
		}
	}
	lower := b.String()

	var out strings.Builder
	prevDash := true // suppress leading dash
	for _, r := range lower {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out.WriteRune(r)
			prevDash = false
		case unicode.IsSpace(r), r == '-', r == '_':
			if !prevDash {
				out.WriteByte('-')
				prevDash = true
			}
		}
	}
	slug := strings.TrimRight(out.String(), "-")

	if len(slug) <= max {
		return slug
	}

	// Truncate at the last "-" within the limit, so we cut on word boundaries.
	cut := slug[:max]
	if i := strings.LastIndex(cut, "-"); i > 0 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, "-")
}
