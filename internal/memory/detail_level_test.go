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

func TestNavigationHint(t *testing.T) {
	tests := []struct {
		name    string
		showing int
		total   int
		hint    string
		want    string
	}{
		{"all results fit", 10, 10, "hint", ""},
		{"showing more than total", 15, 10, "hint", ""},
		{"total is zero", 0, 0, "hint", ""},
		{"total is negative", 5, -1, "hint", ""},
		{"capped with hint", 10, 47, "Use mem_get_observation #ID for full content.", "\n📊 Showing 10 of 47. Use mem_get_observation #ID for full content."},
		{"capped without hint", 5, 20, "", "\n📊 Showing 5 of 20."},
		{"showing zero of many", 0, 100, "Try different filters.", "\n📊 Showing 0 of 100. Try different filters."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NavigationHint(tt.showing, tt.total, tt.hint)
			if got != tt.want {
				t.Errorf("NavigationHint(%d, %d, %q) =\n  %q\nwant:\n  %q",
					tt.showing, tt.total, tt.hint, got, tt.want)
			}
		})
	}
}
