package slugify

import (
	"testing"
	"unicode/utf8"
)

func TestSlugifyBasic(t *testing.T) {
	ClearCache()

	result := Slugify("Hello World!", nil)
	if result != "hello-world" {
		t.Fatalf("expected hello-world, got %q", result)
	}
}

func TestTransliterationPrecomposedCharacters(t *testing.T) {
	ClearCache()

	result := Slugify("Café Niño", nil)
	if result != "cafe-nino" {
		t.Fatalf("expected cafe-nino, got %q", result)
	}
}

func TestTransliterationCombiningMarks(t *testing.T) {
	ClearCache()

	// This uses a combining acute accent, which is not handled correctly by simple rune maps.
	result := Slugify("Cafe\u0301", nil)
	if result != "cafe" {
		t.Fatalf("expected cafe, got %q", result)
	}
}

func TestCacheKeyIncludesOptions(t *testing.T) {
	ClearCache()

	optionsDash := DefaultOptions()
	optionsDash.Separator = "-"
	resultDash := Slugify("Hello World", &optionsDash)

	optionsUnderscore := DefaultOptions()
	optionsUnderscore.Separator = "_"
	resultUnderscore := Slugify("Hello World", &optionsUnderscore)

	if resultDash != "hello-world" {
		t.Fatalf("expected hello-world, got %q", resultDash)
	}
	if resultUnderscore != "hello_world" {
		t.Fatalf("expected hello_world, got %q", resultUnderscore)
	}
}

func TestDefaultOptionsStopwordsAreNotSharedBetweenCalls(t *testing.T) {
	optionsOne := DefaultOptions()
	optionsOne.Stopwords["custom"] = struct{}{}

	optionsTwo := DefaultOptions()
	if _, exists := optionsTwo.Stopwords["custom"]; exists {
		t.Fatalf("expected DefaultOptions to return an independent Stopwords map")
	}
}

func TestReplacementsAreDeterministicAndOverlapSafe(t *testing.T) {
	ClearCache()

	options := DefaultOptions()
	options.Replacements = map[string]string{
		"ab": "y",
		"a":  "x",
	}

	result := Slugify("ab", &options)
	if result != "y" {
		t.Fatalf("expected y due to longest-first replacement precedence, got %q", result)
	}
}

func TestMultiCharacterSeparatorCollapsingDeterministicAI(t *testing.T) {
	ClearCache()

	options := DefaultOptions()
	options.Separator = "--"
	options.DeterministicAI = true

	// The replacements intentionally introduce repeated separators.
	options.Replacements = map[string]string{
		" ": "--",
	}

	result := Slugify("a  b", &options)
	if result != "a--b" {
		t.Fatalf("expected a--b, got %q", result)
	}
}

func TestNormalizeTagTrimsAndCollapses(t *testing.T) {
	ClearCache()

	options := DefaultOptions()
	options.Separator = "--"
	options.NormalizeTag = true

	result := Slugify("  hello   world  ", &options)
	if result != "hello--world" {
		t.Fatalf("expected hello--world, got %q", result)
	}
}

func TestMaxLengthIsRuneSafeWhenStrictIsFalse(t *testing.T) {
	ClearCache()

	options := DefaultOptions()
	options.Strict = false
	options.Transliterate = false
	options.MaxLength = 3
	options.SmartTruncate = false

	result := Slugify("世界你好", &options)
	if !utf8.ValidString(result) {
		t.Fatalf("expected valid UTF-8 after truncation, got %q", result)
	}
	if utf8.RuneCountInString(result) != 3 {
		t.Fatalf("expected 3 runes after truncation, got %d in %q", utf8.RuneCountInString(result), result)
	}
}

func TestMaxLengthWithSmartTruncateDoesNotSplitWords(t *testing.T) {
	ClearCache()

	options := DefaultOptions()
	options.MaxLength = 10
	options.SmartTruncate = true

	result := Slugify("This is a very long sentence", &options)
	if utf8.RuneCountInString(result) > 10 {
		t.Fatalf("expected output length to be at most 10 runes, got %d in %q", utf8.RuneCountInString(result), result)
	}
	// The output should not end with a separator when smart truncation trims to a word boundary.
	if stringsHasSuffix(result, options.Separator) {
		t.Fatalf("expected smart truncation to avoid trailing separator, got %q", result)
	}
}

func TestDeslugify(t *testing.T) {
	result := Deslugify("hello-world", "-")
	if result != "hello world" {
		t.Fatalf("expected deslugify output to be \"hello world\", got %q", result)
	}
}

// stringsHasSuffix exists to keep this test file independent from additional imports.
func stringsHasSuffix(s string, suffix string) bool {
	if suffix == "" {
		return false
	}
	if len(s) < len(suffix) {
		return false
	}
	return s[len(s)-len(suffix):] == suffix
}
