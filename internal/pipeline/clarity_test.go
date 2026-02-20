package pipeline

import (
	"testing"
)

// --- DefaultDimensions ---

func TestDefaultDimensions_Returns8Dimensions(t *testing.T) {
	dims := DefaultDimensions()
	if len(dims) != 8 {
		t.Fatalf("DefaultDimensions() returned %d dimensions, want 8", len(dims))
	}
}

func TestDefaultDimensions_AllHaveNonZeroWeight(t *testing.T) {
	for _, d := range DefaultDimensions() {
		if d.Weight <= 0 {
			t.Errorf("dimension %s has weight %d, want > 0", d.Name, d.Weight)
		}
	}
}

func TestDefaultDimensions_AllUncoveredByDefault(t *testing.T) {
	for _, d := range DefaultDimensions() {
		if d.Covered {
			t.Errorf("dimension %s should be uncovered by default", d.Name)
		}
		if d.Score != 0 {
			t.Errorf("dimension %s has score %d, want 0", d.Name, d.Score)
		}
	}
}

func TestDefaultDimensions_ExpectedNames(t *testing.T) {
	expectedNames := map[string]bool{
		"target_users":      true,
		"core_functionality": true,
		"data_model":        true,
		"integrations":      true,
		"edge_cases":        true,
		"security":          true,
		"scale_performance": true,
		"scope_boundaries":  true,
	}

	for _, d := range DefaultDimensions() {
		if !expectedNames[d.Name] {
			t.Errorf("unexpected dimension name: %s", d.Name)
		}
		delete(expectedNames, d.Name)
	}

	for name := range expectedNames {
		t.Errorf("missing expected dimension: %s", name)
	}
}

// --- CalculateScore ---

func TestCalculateScore_EmptySlice(t *testing.T) {
	got := CalculateScore(nil)
	if got != 0 {
		t.Errorf("CalculateScore(nil) = %d, want 0", got)
	}
}

func TestCalculateScore_AllZeroScores(t *testing.T) {
	dims := DefaultDimensions() // All scores are 0.
	got := CalculateScore(dims)
	if got != 0 {
		t.Errorf("CalculateScore(all zero) = %d, want 0", got)
	}
}

func TestCalculateScore_AllPerfect(t *testing.T) {
	dims := DefaultDimensions()
	for i := range dims {
		dims[i].Score = 100
	}
	got := CalculateScore(dims)
	if got != 100 {
		t.Errorf("CalculateScore(all 100) = %d, want 100", got)
	}
}

func TestCalculateScore_WeightedCorrectly(t *testing.T) {
	// Two dimensions: one with weight 10 scoring 100, one with weight 10 scoring 0.
	// Expected: (100*10 + 0*10) / (10+10) = 50.
	dims := []ClarityDimension{
		{Name: "a", Weight: 10, Score: 100},
		{Name: "b", Weight: 10, Score: 0},
	}
	got := CalculateScore(dims)
	if got != 50 {
		t.Errorf("CalculateScore(50/50) = %d, want 50", got)
	}
}

func TestCalculateScore_HighWeightDominates(t *testing.T) {
	// Weight 9 scores 100, weight 1 scores 0.
	// Expected: (100*9 + 0*1) / (9+1) = 90.
	dims := []ClarityDimension{
		{Name: "heavy", Weight: 9, Score: 100},
		{Name: "light", Weight: 1, Score: 0},
	}
	got := CalculateScore(dims)
	if got != 90 {
		t.Errorf("CalculateScore(weighted) = %d, want 90", got)
	}
}

func TestCalculateScore_ZeroTotalWeight(t *testing.T) {
	dims := []ClarityDimension{
		{Name: "zero", Weight: 0, Score: 100},
	}
	got := CalculateScore(dims)
	if got != 0 {
		t.Errorf("CalculateScore(zero weight) = %d, want 0", got)
	}
}

func TestCalculateScore_IntegerDivision(t *testing.T) {
	// Verify integer division behavior: (80*3 + 50*7) / (3+7) = (240+350)/10 = 59.
	dims := []ClarityDimension{
		{Name: "a", Weight: 3, Score: 80},
		{Name: "b", Weight: 7, Score: 50},
	}
	got := CalculateScore(dims)
	if got != 59 {
		t.Errorf("CalculateScore = %d, want 59", got)
	}
}

// --- UncoveredDimensions ---

func TestUncoveredDimensions_AllUncovered(t *testing.T) {
	dims := DefaultDimensions()
	uncovered := UncoveredDimensions(dims)
	if len(uncovered) != len(dims) {
		t.Errorf("UncoveredDimensions = %d, want %d (all)", len(uncovered), len(dims))
	}
}

func TestUncoveredDimensions_SomeCovered(t *testing.T) {
	dims := DefaultDimensions()
	dims[0].Covered = true
	dims[2].Covered = true
	dims[4].Covered = true

	uncovered := UncoveredDimensions(dims)
	want := len(dims) - 3
	if len(uncovered) != want {
		t.Errorf("UncoveredDimensions = %d, want %d", len(uncovered), want)
	}

	// Verify the covered ones are NOT in the result.
	for _, d := range uncovered {
		if d.Covered {
			t.Errorf("covered dimension %s found in uncovered list", d.Name)
		}
	}
}

func TestUncoveredDimensions_AllCovered(t *testing.T) {
	dims := DefaultDimensions()
	for i := range dims {
		dims[i].Covered = true
	}

	uncovered := UncoveredDimensions(dims)
	if len(uncovered) != 0 {
		t.Errorf("UncoveredDimensions = %d, want 0 (all covered)", len(uncovered))
	}
}

func TestUncoveredDimensions_EmptySlice(t *testing.T) {
	uncovered := UncoveredDimensions(nil)
	if len(uncovered) != 0 {
		t.Errorf("UncoveredDimensions(nil) = %d, want 0", len(uncovered))
	}
}
