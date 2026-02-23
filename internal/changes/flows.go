package changes

import "fmt"

// FlowRegistry defines the stage sequence for each (ChangeType, ChangeSize) pair.
// This is the heart of the adaptive pipeline: instead of a fixed 7-stage sequence,
// the flow adapts to what the change actually needs.
//
// All flows end with StageVerify. The stages before it vary:
//   - Small changes: minimal documentation (3 stages)
//   - Medium changes: moderate documentation (4 stages)
//   - Large changes: comprehensive documentation (5-6 stages)
var FlowRegistry = map[ChangeType]map[ChangeSize][]ChangeStage{
	TypeFix: {
		SizeSmall:  {StageDescribe, StageTasks, StageVerify},
		SizeMedium: {StageDescribe, StageSpec, StageTasks, StageVerify},
		SizeLarge:  {StageDescribe, StageSpec, StageDesign, StageTasks, StageVerify},
	},
	TypeFeature: {
		SizeSmall:  {StageDescribe, StageTasks, StageVerify},
		SizeMedium: {StagePropose, StageSpec, StageTasks, StageVerify},
		SizeLarge:  {StagePropose, StageSpec, StageClarify, StageDesign, StageTasks, StageVerify},
	},
	TypeRefactor: {
		SizeSmall:  {StageScope, StageTasks, StageVerify},
		SizeMedium: {StageScope, StageDesign, StageTasks, StageVerify},
		SizeLarge:  {StageScope, StageSpec, StageDesign, StageTasks, StageVerify},
	},
	TypeEnhancement: {
		SizeSmall:  {StageDescribe, StageTasks, StageVerify},
		SizeMedium: {StagePropose, StageSpec, StageTasks, StageVerify},
		SizeLarge:  {StagePropose, StageSpec, StageClarify, StageDesign, StageTasks, StageVerify},
	},
}

// StageFlow returns the ordered list of stages for the given type and size.
// Returns an error if the combination is not recognized.
func StageFlow(t ChangeType, s ChangeSize) ([]ChangeStage, error) {
	if err := ValidateType(t); err != nil {
		return nil, err
	}
	if err := ValidateSize(s); err != nil {
		return nil, err
	}

	sizes, ok := FlowRegistry[t]
	if !ok {
		return nil, fmt.Errorf("no flow defined for type %q", t)
	}
	flow, ok := sizes[s]
	if !ok {
		return nil, fmt.Errorf("no flow defined for %s/%s", t, s)
	}

	// Return a copy to prevent mutation of the registry.
	result := make([]ChangeStage, len(flow))
	copy(result, flow)
	return result, nil
}

// stageFilenames maps change stages to their artifact filenames.
var stageFilenames = map[ChangeStage]string{
	StageDescribe: "describe.md",
	StageScope:    "scope.md",
	StagePropose:  "propose.md",
	StageSpec:     "spec.md",
	StageClarify:  "clarify.md",
	StageDesign:   "design.md",
	StageTasks:    "tasks.md",
	StageVerify:   "verify.md",
}

// StageFilename returns the artifact filename for a given stage.
// Returns empty string for unknown stages.
func StageFilename(stage ChangeStage) string {
	return stageFilenames[stage]
}
