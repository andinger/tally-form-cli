package cli

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"simple ASCII", "Hello World", 25, "hello-world"},
		{"German umlauts", "Wie zufrieden sind Sie?", 25, "wie-zufrieden-sind-sie"},
		{"all umlauts + sharp s", "Ärger über Größe", 25, "aerger-ueber-groesse"},
		{"truncates at word boundary", "Wie zufrieden sind Sie mit unserem Service heute?", 25, "wie-zufrieden-sind-sie"},
		{"truncates exactly at boundary", "alpha-beta-gamma", 11, "alpha-beta"},
		{"single long word past limit", "supercalifragilisticexpialidocious", 25, "supercalifragilisticexpia"},
		{"removes punctuation", "Email-Adresse??!", 25, "email-adresse"},
		{"collapses runs of separators", "foo   ---   bar", 25, "foo-bar"},
		{"trims leading and trailing dashes", "---foo bar---", 25, "foo-bar"},
		{"strips emojis and unicode", "🎉 Feedback ✅", 25, "feedback"},
		{"empty input", "", 25, ""},
		{"only punctuation", "?!?.,", 25, ""},
		{"max zero returns empty", "anything", 0, ""},
		{"keeps numbers", "Frage 1 von 5", 25, "frage-1-von-5"},
		{"underscore treated as separator", "snake_case_input", 25, "snake-case-input"},
		{"upper-case umlaut Ü is transliterated", "Über uns", 25, "ueber-uns"},
		{"non-latin gets stripped (Japanese)", "こんにちは Hello", 25, "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slugify(tt.in, tt.max)
			if got != tt.want {
				t.Errorf("slugify(%q, %d) = %q, want %q", tt.in, tt.max, got, tt.want)
			}
		})
	}
}
