package changes

import (
	"testing"
)

func TestValidateType(t *testing.T) {
	tests := []struct {
		name    string
		input   ChangeType
		wantErr bool
	}{
		{"feature is valid", TypeFeature, false},
		{"fix is valid", TypeFix, false},
		{"refactor is valid", TypeRefactor, false},
		{"enhancement is valid", TypeEnhancement, false},
		{"empty is invalid", ChangeType(""), true},
		{"unknown is invalid", ChangeType("hotfix"), true},
		{"case sensitive", ChangeType("Feature"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateType(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateSize(t *testing.T) {
	tests := []struct {
		name    string
		input   ChangeSize
		wantErr bool
	}{
		{"small is valid", SizeSmall, false},
		{"medium is valid", SizeMedium, false},
		{"large is valid", SizeLarge, false},
		{"empty is invalid", ChangeSize(""), true},
		{"unknown is invalid", ChangeSize("xl"), true},
		{"case sensitive", ChangeSize("Small"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSize(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSize(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple description",
			input: "Fix FTS5 empty query crash",
			want:  "fix-fts5-empty-query-crash",
		},
		{
			name:  "already slugified",
			input: "fix-something",
			want:  "fix-something",
		},
		{
			name:  "special characters removed",
			input: "Add OAuth2.0 (Google) integration!",
			want:  "add-oauth20-google-integration",
		},
		{
			name:  "consecutive spaces collapsed",
			input: "fix   multiple   spaces",
			want:  "fix-multiple-spaces",
		},
		{
			name:  "underscores become hyphens",
			input: "refactor_auth_module",
			want:  "refactor-auth-module",
		},
		{
			name:  "mixed separators",
			input: "fix - the _ bug -- now",
			want:  "fix-the-bug-now",
		},
		{
			name:  "leading and trailing spaces",
			input: "  fix something  ",
			want:  "fix-something",
		},
		{
			name:  "empty string",
			input: "",
			want:  "unnamed-change",
		},
		{
			name:  "only spaces",
			input: "   ",
			want:  "unnamed-change",
		},
		{
			name:  "only special characters",
			input: "!@#$%^&*()",
			want:  "unnamed-change",
		},
		{
			name:  "unicode characters stripped to ascii",
			input: "Añadir soporte español",
			want:  "aadir-soporte-espaol",
		},
		{
			name:  "numbers only",
			input: "12345",
			want:  "12345",
		},
		{
			name:  "long description truncated",
			input: "this is a very long description that exceeds the maximum slug length of fifty characters significantly",
			want:  "this-is-a-very-long-description-that-exceeds-the",
		},
		{
			name:  "exactly 50 chars",
			input: "12345678901234567890123456789012345678901234567890",
			want:  "12345678901234567890123456789012345678901234567890",
		},
		{
			name:  "51 chars truncated at word boundary",
			input: "abcdefghijklmnopqrstuvwxyz abcdefghijklmnopqrstuvwx",
			want:  "abcdefghijklmnopqrstuvwxyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Slugify(tt.input)
			if got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
			// Invariant: slug is never longer than maxSlugLen.
			if len(got) > maxSlugLen {
				t.Errorf("Slugify(%q) length = %d, exceeds max %d", tt.input, len(got), maxSlugLen)
			}
			// Invariant: slug never starts or ends with hyphen.
			if got != "unnamed-change" && (got[0] == '-' || got[len(got)-1] == '-') {
				t.Errorf("Slugify(%q) = %q, starts or ends with hyphen", tt.input, got)
			}
		})
	}
}
