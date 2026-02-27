package slugify

import (
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// removeDiacritics normalizes a string and removes Unicode nonspacing marks.
//
// This function exists because simple rune mapping does not handle combining
// mark sequences such as "e" + "\u0301".
func removeDiacritics(s string) string {
	transformer := transform.Chain(
		norm.NFKD,
		transform.RemoveFunc(func(r rune) bool {
			// Mn is the Unicode category for nonspacing marks, which includes most combining diacritics.
			return unicode.Is(unicode.Mn, r)
		}),
		norm.NFKC,
	)

	result, _, err := transform.String(transformer, s)
	if err != nil {
		// transform.String only fails on internal transformer issues, but returning the original
		// string is a safer fallback than returning partial output.
		return s
	}
	return result
}
