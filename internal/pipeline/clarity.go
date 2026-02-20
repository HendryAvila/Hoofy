// Package pipeline - clarity gate scoring.
//
// The Clarity Gate is the CORE value proposition of SDD-Hoffy.
// It analyzes requirements for ambiguity and produces a score (0-100).
// The pipeline cannot advance past the "clarify" stage until the score
// meets the threshold for the active mode.
package pipeline

// ClarityDimension represents one axis of clarity evaluation.
// Each dimension contributes to the overall clarity score.
type ClarityDimension struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Weight      int    `json:"weight"`   // relative importance (1-10)
	Covered     bool   `json:"covered"`  // whether this dimension has been addressed
	Score       int    `json:"score"`    // 0-100 for this dimension
}

// DefaultDimensions returns the standard clarity dimensions.
// These represent the key areas where ambiguity causes AI hallucinations.
func DefaultDimensions() []ClarityDimension {
	return []ClarityDimension{
		{
			Name:        "target_users",
			Description: "Who are the target users? Are personas clearly defined?",
			Weight:      8,
		},
		{
			Name:        "core_functionality",
			Description: "What does the system DO? Are the main features unambiguous?",
			Weight:      10,
		},
		{
			Name:        "data_model",
			Description: "What data does the system manage? Are entities and relationships clear?",
			Weight:      7,
		},
		{
			Name:        "integrations",
			Description: "What external systems does it interact with? Are APIs/protocols defined?",
			Weight:      6,
		},
		{
			Name:        "edge_cases",
			Description: "Are error scenarios and edge cases addressed?",
			Weight:      8,
		},
		{
			Name:        "security",
			Description: "Are authentication, authorization, and data protection requirements clear?",
			Weight:      7,
		},
		{
			Name:        "scale_performance",
			Description: "Are performance expectations and scale requirements defined?",
			Weight:      5,
		},
		{
			Name:        "scope_boundaries",
			Description: "Is it clear what the system does NOT do? Are boundaries explicit?",
			Weight:      9,
		},
	}
}

// ClarityReport contains the full analysis of a requirements set.
type ClarityReport struct {
	Dimensions   []ClarityDimension `json:"dimensions"`
	OverallScore int                `json:"overall_score"`
	Questions    []ClarityQuestion  `json:"questions,omitempty"`
	GatePassed   bool               `json:"gate_passed"`
}

// ClarityQuestion is a question generated to resolve an ambiguity.
type ClarityQuestion struct {
	Dimension string `json:"dimension"` // which dimension this addresses
	Question  string `json:"question"`
	Priority  string `json:"priority"` // high | medium | low
}

// ClarityAnswer records a user's answer to a clarity question.
type ClarityAnswer struct {
	Dimension string `json:"dimension"`
	Question  string `json:"question"`
	Answer    string `json:"answer"`
}

// CalculateScore computes the weighted overall score from dimensions.
func CalculateScore(dimensions []ClarityDimension) int {
	totalWeight := 0
	weightedSum := 0

	for _, d := range dimensions {
		totalWeight += d.Weight
		weightedSum += d.Score * d.Weight
	}

	if totalWeight == 0 {
		return 0
	}

	return weightedSum / totalWeight
}

// UncoveredDimensions returns dimensions that haven't been addressed yet.
func UncoveredDimensions(dimensions []ClarityDimension) []ClarityDimension {
	var uncovered []ClarityDimension
	for _, d := range dimensions {
		if !d.Covered {
			uncovered = append(uncovered, d)
		}
	}
	return uncovered
}
