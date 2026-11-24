package teams

import "testing"

func TestSlug(t *testing.T) {
	for in, want := range map[string]string{
		"FK Austria Wien":     "fk-austria-wien",
		"Bayern München":      "bayern-muenchen",
		"  Atlético Madrid":   "atletico-madrid",
		"Paris Saint-Germain": "paris-saint-germain",
		"???":                 "",
	} {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}
