package slugify

import (
	"regexp"
	"strings"
	"sync"
	"unicode"
)

// Version of the library.
const Version = "v1.0.0"

// Options controls slug behavior.
type Options struct {
	Lowercase        bool
	Strict           bool
	MaxLength        int
	SmartTruncate    bool
	Separator        string
	RemoveStopwords  bool
	Stopwords        map[string]struct{}
	Replacements     map[string]string
	Transliterate    bool
	DeterministicAI  bool
	NormalizeTag     bool
}

var cache sync.Map

var defaultStopwords = map[string]struct{}{
	"a": {}, "an": {}, "the": {}, "and": {}, "or": {}, "but": {},
	"in": {}, "on": {}, "at": {}, "to": {}, "for": {}, "of": {},
}

var transliterationMap = map[rune]string{
	'á': "a", 'à': "a", 'ä': "a", 'â': "a",
	'é': "e", 'è': "e", 'ë': "e", 'ê': "e",
	'í': "i", 'ì': "i", 'ï': "i", 'î': "i",
	'ó': "o", 'ò': "o", 'ö': "o", 'ô': "o",
	'ú': "u", 'ù': "u", 'ü': "u", 'û': "u",
	'ñ': "n", 'ç': "c",
}

// DefaultOptions returns production defaults.
func DefaultOptions() Options {
	return Options{
		Lowercase:       true,
		Strict:          true,
		MaxLength:       0,
		SmartTruncate:   true,
		Separator:       "-",
		RemoveStopwords: false,
		Stopwords:       defaultStopwords,
		Replacements:    map[string]string{},
		Transliterate:   true,
		DeterministicAI: false,
		NormalizeTag:    false,
	}
}

// Slugify converts input into a URL-safe slug.
func Slugify(input string, opts *Options) string {
	if input == "" {
		return ""
	}

	if opts == nil {
		defaults := DefaultOptions()
		opts = &defaults
	}

	cacheKey := buildCacheKey(input, opts)
	if val, ok := cache.Load(cacheKey); ok {
		return val.(string)
	}

	s := input

	if opts.Lowercase {
		s = strings.ToLower(s)
	}

	for old, newVal := range opts.Replacements {
		s = strings.ReplaceAll(s, old, newVal)
	}

	if opts.Transliterate {
		s = transliterate(s)
	}

	var builder strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		} else {
			builder.WriteRune(' ')
		}
	}
	s = builder.String()

	words := strings.Fields(s)

	if opts.RemoveStopwords {
		filtered := make([]string, 0, len(words))
		for _, w := range words {
			if _, exists := opts.Stopwords[w]; !exists {
				filtered = append(filtered, w)
			}
		}
		words = filtered
	}

	slug := strings.Join(words, opts.Separator)

	if opts.Strict {
		re := regexp.MustCompile("[^a-zA-Z0-9" + regexp.QuoteMeta(opts.Separator) + "]")
		slug = re.ReplaceAllString(slug, "")
	}

	if opts.MaxLength > 0 && len(slug) > opts.MaxLength {
		if opts.SmartTruncate {
			slug = smartTrim(slug, opts.MaxLength, opts.Separator)
		} else {
			slug = slug[:opts.MaxLength]
		}
	}

	if opts.DeterministicAI {
		slug = normalizeDeterministic(slug, opts.Separator)
	}

	if opts.NormalizeTag {
		slug = normalizeTagID(slug, opts.Separator)
	}

	cache.Store(cacheKey, slug)
	return slug
}

// Deslugify converts a slug back into readable text.
func Deslugify(slug string, separator string) string {
	if slug == "" {
		return ""
	}
	if separator == "" {
		separator = "-"
	}
	return strings.ReplaceAll(slug, separator, " ")
}

func transliterate(s string) string {
	var builder strings.Builder
	for _, r := range s {
		if replacement, ok := transliterationMap[r]; ok {
			builder.WriteString(replacement)
		} else {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func smartTrim(s string, max int, sep string) string {
	if len(s) <= max {
		return s
	}
	cut := s[:max]
	lastSep := strings.LastIndex(cut, sep)
	if lastSep > 0 {
		return cut[:lastSep]
	}
	return cut
}

func normalizeDeterministic(s string, sep string) string {
	re := regexp.MustCompile(regexp.QuoteMeta(sep) + "+")
	return re.ReplaceAllString(s, sep)
}

func normalizeTagID(s string, sep string) string {
	s = strings.Trim(s, sep)
	re := regexp.MustCompile(regexp.QuoteMeta(sep) + "+")
	return re.ReplaceAllString(s, sep)
}

func buildCacheKey(input string, opts *Options) string {
	return input +
		opts.Separator +
		string(rune(opts.MaxLength)) +
		boolToStr(opts.Strict) +
		boolToStr(opts.Transliterate) +
		boolToStr(opts.DeterministicAI)
}

func boolToStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
