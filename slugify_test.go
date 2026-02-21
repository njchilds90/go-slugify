package slugify

import "testing"

func TestSlugifyBasic(t *testing.T) {
	result := Slugify("Hello World!", nil)
	if result != "hello-world" {
		t.Errorf("expected hello-world, got %s", result)
	}
}

func TestTransliteration(t *testing.T) {
	result := Slugify("Café Niño", nil)
	if result != "cafe-nino" {
		t.Errorf("expected cafe-nino, got %s", result)
	}
}

func TestMaxLength(t *testing.T) {
	opts := DefaultOptions()
	opts.MaxLength = 10
	result := Slugify("This is a very long sentence", &opts)
	if len(result) > 10 {
		t.Errorf("max length exceeded")
	}
}

func TestDeslugify(t *testing.T) {
	result := Deslugify("hello-world", "-")
	if result != "hello world" {
		t.Errorf("deslugify failed")
	}
}
