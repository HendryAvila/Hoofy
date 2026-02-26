package memory

import "testing"

func TestParseDetailLevel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"summary", DetailSummary},
		{"standard", DetailStandard},
		{"full", DetailFull},
		{"", DetailStandard},
		{"invalid", DetailStandard},
		{"SUMMARY", DetailStandard}, // case-sensitive — only lowercase accepted
		{"Summary", DetailStandard},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseDetailLevel(tt.input)
			if got != tt.want {
				t.Errorf("ParseDetailLevel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDetailLevelValues(t *testing.T) {
	vals := DetailLevelValues()
	if len(vals) != 3 {
		t.Fatalf("expected 3 values, got %d", len(vals))
	}

	expected := map[string]bool{
		DetailSummary:  true,
		DetailStandard: true,
		DetailFull:     true,
	}

	for _, v := range vals {
		if !expected[v] {
			t.Errorf("unexpected value: %q", v)
		}
	}
}

func TestSummaryFooterIsNotEmpty(t *testing.T) {
	if SummaryFooter == "" {
		t.Error("SummaryFooter should not be empty")
	}
}
