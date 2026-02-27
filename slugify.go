package slugify

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// Version is the current semantic version of this library.
//
// This value is intended for diagnostics and does not affect behavior.
const Version = "v1.0.0"

// Options controls how Slugify transforms input text.
//
// You should usually start from DefaultOptions and then override the fields you
// care about, because DefaultOptions also protects you from shared mutable
// defaults.
//
// All fields are optional; Slugify applies DefaultOptions when a nil pointer is
// provided.
type Options struct {
	// Lowercase controls whether the input is converted to lower case before further processing.
	Lowercase bool

	// Strict controls whether the final slug is restricted to ASCII letters, ASCII digits, and the separator.
	//
	// When Strict is false, the slug may include non-ASCII letters and digits.
	Strict bool

	// MaxLength limits the number of Unicode code points (runes) in the output slug.
	//
	// A value of zero or less means no maximum length is applied.
	MaxLength int

	// SmartTruncate controls whether the slug is truncated at a separator boundary when MaxLength is set.
	//
	// When SmartTruncate is false, the slug is truncated exactly at MaxLength runes.
	SmartTruncate bool

	// Separator is the string used to join words.
	//
	// If Separator is empty or whitespace-only, Slugify replaces it with "-".
	Separator string

	// RemoveStopwords controls whether common stopwords are removed from the output.
	RemoveStopwords bool

	// Stopwords is the set of words that may be removed when RemoveStopwords is true.
	//
	// This map is treated as read-only by the library.
	Stopwords map[string]struct{}

	// Replacements applies literal string replacements before tokenization.
	//
	// Replacements are applied deterministically and in overlap-safe order.
	Replacements map[string]string

	// Transliterate controls whether Unicode diacritics are removed and a small set of additional
	// character transliterations are applied.
	Transliterate bool

	// DeterministicAI controls whether repeated separators are collapsed.
	//
	// This option exists to produce stable identifiers in pipelines where different inputs may
	// otherwise introduce repeated separators.
	DeterministicAI bool

	// NormalizeTag controls whether leading and trailing separators are trimmed and repeated
	// separators are collapsed.
	//
	// This option is useful when slugs are used as tag identifiers and consumers expect no
	// leading or trailing separators.
	NormalizeTag bool
}

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

// defaultSlugCache is a bounded cache to avoid unbounded memory growth.
//
// The cache is intentionally package-scoped so repeated calls do not repeatedly allocate
// cache structures.
var defaultSlugCache = newLRUCache[string, string](10_000)

// DefaultOptions returns production-oriented defaults.
//
// The returned Stopwords map is a defensive copy so that callers can safely mutate their own
// options without affecting other callers.
func DefaultOptions() Options {
	stopwordsCopy := make(map[string]struct{}, len(defaultStopwords))
	for key := range defaultStopwords {
		stopwordsCopy[key] = struct{}{}
	}

	return Options{
		Lowercase:       true,
		Strict:          true,
		MaxLength:       0,
		SmartTruncate:   true,
		Separator:       "-",
		RemoveStopwords: false,
		Stopwords:       stopwordsCopy,
		Replacements:    map[string]string{},
		Transliterate:   true,
		DeterministicAI: false,
		NormalizeTag:    false,
	}
}

// Slugify converts input into a URL-safe slug.
//
// The transformation is designed to be stable and deterministic for the same input and options.
// For performance, results are cached in a bounded in-memory cache keyed by the full set of
// behavior-affecting options.
func Slugify(input string, opts *Options) string {
	if input == "" {
		return ""
	}

	if opts == nil {
		defaults := DefaultOptions()
		opts = &defaults
	}

	separator := sanitizeSeparator(opts.Separator)

	cacheKey := buildCacheKey(input, opts, separator)
	if cached, ok := defaultSlugCache.Get(cacheKey); ok {
		return cached
	}

	s := input

	if opts.Lowercase {
		s = strings.ToLower(s)
	}

	s = applyReplacementsDeterministic(s, opts.Replacements)

	if opts.Transliterate {
		// Removing diacritics with Unicode normalization handles both precomposed characters
		// and combining character sequences.
		s = removeDiacritics(s)
		s = transliterateFallbackMap(s)
	}

	// Convert all non-letter and non-digit runes into spaces, so that tokenization is based on words.
	var builder strings.Builder
	builder.Grow(len(s))
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			continue
		}
		builder.WriteRune(' ')
	}
	words := strings.Fields(builder.String())

	if opts.RemoveStopwords {
		filtered := make([]string, 0, len(words))
		for _, word := range words {
			if _, exists := opts.Stopwords[word]; !exists {
				filtered = append(filtered, word)
			}
		}
		words = filtered
	}

	slug := strings.Join(words, separator)

	if opts.Strict {
		slug = strictFilterASCII(slug, separator)
	}

	if opts.MaxLength > 0 {
		if opts.SmartTruncate {
			slug = smartTrimRunes(slug, opts.MaxLength, separator)
		} else {
			slug = truncateRunes(slug, opts.MaxLength)
		}
	}

	if opts.DeterministicAI {
		slug = collapseRepeatedSubstring(slug, separator)
	}

	if opts.NormalizeTag {
		slug = normalizeTagIdentifier(slug, separator)
	}

	defaultSlugCache.Add(cacheKey, slug)
	return slug
}

// Deslugify converts a slug back into readable text by replacing the separator with spaces.
//
// This function does not attempt to restore capitalization or punctuation.
func Deslugify(slug string, separator string) string {
	if slug == "" {
		return ""
	}
	separator = sanitizeSeparator(separator)
	return strings.ReplaceAll(slug, separator, " ")
}

// ClearCache removes all cached slugs.
//
// This function exists for long-running processes that want explicit control over memory use,
// and for test suites that want to avoid cross-test interference.
func ClearCache() {
	defaultSlugCache.Clear()
}

// SetCacheCapacity changes the maximum number of cache entries.
//
// A capacity of zero or less disables caching entirely.
func SetCacheCapacity(capacity int) {
	defaultSlugCache.SetCapacity(capacity)
}

// sanitizeSeparator normalizes the separator string so downstream code can rely on a non-empty value.
func sanitizeSeparator(separator string) string {
	separator = strings.TrimSpace(separator)
	if separator == "" {
		return "-"
	}
	return separator
}

// transliterateFallbackMap applies a small, explicit transliteration map.
//
// This function exists to preserve behavior for a few common characters even when a user is not
// relying solely on Unicode diacritic removal.
func transliterateFallbackMap(s string) string {
	var builder strings.Builder
	builder.Grow(len(s))
	for _, r := range s {
		if replacement, ok := transliterationMap[r]; ok {
			builder.WriteString(replacement)
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

// applyReplacementsDeterministic applies literal replacements in a stable and overlap-safe order.
//
// The replacement order is longest key first, and for equal lengths it is lexical order.
// This makes behavior deterministic and avoids surprising partial replacements when keys overlap.
func applyReplacementsDeterministic(s string, replacements map[string]string) string {
	if len(replacements) == 0 {
		return s
	}

	keys := make([]string, 0, len(replacements))
	for key := range replacements {
		if key == "" {
			// An empty replacement key would match everywhere and is almost certainly a caller bug.
			// Ignoring it avoids an infinite loop or extreme memory growth.
			continue
		}
		keys = append(keys, key)
	}

	sort.Slice(keys, func(i, j int) bool {
		if len(keys[i]) != len(keys[j]) {
			return len(keys[i]) > len(keys[j])
		}
		return keys[i] < keys[j]
	})

	pairs := make([]string, 0, len(keys)*2)
	for _, key := range keys {
		pairs = append(pairs, key, replacements[key])
	}

	replacer := strings.NewReplacer(pairs...)
	return replacer.Replace(s)
}

// strictFilterASCII removes characters that are not ASCII letters, ASCII digits, or part of the separator.
//
// This function avoids regular expressions to reduce per-call allocations and compilation overhead.
func strictFilterASCII(slug string, separator string) string {
	if slug == "" {
		return ""
	}

	allowedSeparatorRunes := make(map[rune]struct{}, len(separator))
	for _, r := range separator {
		allowedSeparatorRunes[r] = struct{}{}
	}

	var builder strings.Builder
	builder.Grow(len(slug))
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			continue
		}
		if _, ok := allowedSeparatorRunes[r]; ok {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// truncateRunes truncates a string to at most max Unicode code points.
//
// This function exists because slicing by byte index can split multi-byte UTF-8 runes.
func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}

	count := 0
	for i := range s {
		if count == max {
			return s[:i]
		}
		count++
	}
	return s
}

// smartTrimRunes truncates a string to at most max runes and then trims back to the last full separator.
//
// This function is intended to preserve whole words when slugs are joined by a separator.
func smartTrimRunes(s string, max int, separator string) string {
	cut := truncateRunes(s, max)
	if len(cut) == len(s) {
		return s
	}
	if separator == "" {
		return cut
	}
	if last := strings.LastIndex(cut, separator); last > 0 {
		return cut[:last]
	}
	return cut
}

// collapseRepeatedSubstring collapses repeated occurrences of separator into a single occurrence.
//
// This function correctly handles multi-character separators.
func collapseRepeatedSubstring(s string, separator string) string {
	if s == "" || separator == "" {
		return s
	}

	// Fast path for common single-character separators.
	if len(separator) == 1 {
		sepByte := separator[0]
		var builder strings.Builder
		builder.Grow(len(s))
		previousWasSeparator := false
		for i := 0; i < len(s); i++ {
			b := s[i]
			if b == sepByte {
				if previousWasSeparator {
					continue
				}
				previousWasSeparator = true
				builder.WriteByte(b)
				continue
			}
			previousWasSeparator = false
			builder.WriteByte(b)
		}
		return builder.String()
	}

	var builder strings.Builder
	builder.Grow(len(s))

	for i := 0; i < len(s); {
		if strings.HasPrefix(s[i:], separator) {
			builder.WriteString(separator)
			i += len(separator)
			for strings.HasPrefix(s[i:], separator) {
				i += len(separator)
			}
			continue
		}
		builder.WriteByte(s[i])
		i++
	}

	return builder.String()
}

// normalizeTagIdentifier trims leading and trailing separators and collapses repeated separators.
func normalizeTagIdentifier(s string, separator string) string {
	if s == "" {
		return ""
	}
	separator = sanitizeSeparator(separator)
	s = strings.Trim(s, separator)
	s = collapseRepeatedSubstring(s, separator)
	return s
}

// buildCacheKey creates a stable cache key that includes all options that affect output.
//
// This function uses a cryptographic hash to keep keys compact while avoiding collisions
// from unsafe integer encoding and map iteration nondeterminism.
func buildCacheKey(input string, opts *Options, separator string) string {
	h := sha256.New()
	h.Write([]byte("in="))
	h.Write([]byte(input))
	h.Write([]byte("|sep="))
	h.Write([]byte(separator))
	h.Write([]byte("|max="))
	h.Write([]byte(strconv.Itoa(opts.MaxLength)))
	h.Write([]byte("|lower="))
	h.Write([]byte(boolToStr(opts.Lowercase)))
	h.Write([]byte("|strict="))
	h.Write([]byte(boolToStr(opts.Strict)))
	h.Write([]byte("|smart="))
	h.Write([]byte(boolToStr(opts.SmartTruncate)))
	h.Write([]byte("|rmstop="))
	h.Write([]byte(boolToStr(opts.RemoveStopwords)))
	h.Write([]byte("|trans="))
	h.Write([]byte(boolToStr(opts.Transliterate)))
	h.Write([]byte("|detai="))
	h.Write([]byte(boolToStr(opts.DeterministicAI)))
	h.Write([]byte("|normtag="))
	h.Write([]byte(boolToStr(opts.NormalizeTag)))

	if len(opts.Replacements) > 0 {
		keys := make([]string, 0, len(opts.Replacements))
		for key := range opts.Replacements {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			h.Write([]byte("|rep:"))
			h.Write([]byte(key))
			h.Write([]byte("=>"))
			h.Write([]byte(opts.Replacements[key]))
		}
	}

	if opts.RemoveStopwords && len(opts.Stopwords) > 0 {
		keys := make([]string, 0, len(opts.Stopwords))
		for key := range opts.Stopwords {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			h.Write([]byte("|sw:"))
			h.Write([]byte(key))
		}
	}

	return hex.EncodeToString(h.Sum(nil))
}

// boolToStr converts a boolean to a stable single-character string.
func boolToStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
