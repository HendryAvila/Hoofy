package changes

import (
	"testing"
)

func TestStageFlow(t *testing.T) {
	tests := []struct {
		name     string
		cType    ChangeType
		cSize    ChangeSize
		wantFlow []ChangeStage
		wantErr  bool
	}{
		// --- Fix flows ---
		{
			name:     "fix/small",
			cType:    TypeFix,
			cSize:    SizeSmall,
			wantFlow: []ChangeStage{StageDescribe, StageTasks, StageVerify},
		},
		{
			name:     "fix/medium",
			cType:    TypeFix,
			cSize:    SizeMedium,
			wantFlow: []ChangeStage{StageDescribe, StageSpec, StageTasks, StageVerify},
		},
		{
			name:     "fix/large",
			cType:    TypeFix,
			cSize:    SizeLarge,
			wantFlow: []ChangeStage{StageDescribe, StageSpec, StageDesign, StageTasks, StageVerify},
		},
		// --- Feature flows ---
		{
			name:     "feature/small",
			cType:    TypeFeature,
			cSize:    SizeSmall,
			wantFlow: []ChangeStage{StageDescribe, StageTasks, StageVerify},
		},
		{
			name:     "feature/medium",
			cType:    TypeFeature,
			cSize:    SizeMedium,
			wantFlow: []ChangeStage{StagePropose, StageSpec, StageTasks, StageVerify},
		},
		{
			name:     "feature/large",
			cType:    TypeFeature,
			cSize:    SizeLarge,
			wantFlow: []ChangeStage{StagePropose, StageSpec, StageClarify, StageDesign, StageTasks, StageVerify},
		},
		// --- Refactor flows ---
		{
			name:     "refactor/small",
			cType:    TypeRefactor,
			cSize:    SizeSmall,
			wantFlow: []ChangeStage{StageScope, StageTasks, StageVerify},
		},
		{
			name:     "refactor/medium",
			cType:    TypeRefactor,
			cSize:    SizeMedium,
			wantFlow: []ChangeStage{StageScope, StageDesign, StageTasks, StageVerify},
		},
		{
			name:     "refactor/large",
			cType:    TypeRefactor,
			cSize:    SizeLarge,
			wantFlow: []ChangeStage{StageScope, StageSpec, StageDesign, StageTasks, StageVerify},
		},
		// --- Enhancement flows ---
		{
			name:     "enhancement/small",
			cType:    TypeEnhancement,
			cSize:    SizeSmall,
			wantFlow: []ChangeStage{StageDescribe, StageTasks, StageVerify},
		},
		{
			name:     "enhancement/medium",
			cType:    TypeEnhancement,
			cSize:    SizeMedium,
			wantFlow: []ChangeStage{StagePropose, StageSpec, StageTasks, StageVerify},
		},
		{
			name:     "enhancement/large",
			cType:    TypeEnhancement,
			cSize:    SizeLarge,
			wantFlow: []ChangeStage{StagePropose, StageSpec, StageClarify, StageDesign, StageTasks, StageVerify},
		},
		// --- Error cases ---
		{
			name:    "invalid type",
			cType:   ChangeType("hotfix"),
			cSize:   SizeSmall,
			wantErr: true,
		},
		{
			name:    "invalid size",
			cType:   TypeFix,
			cSize:   ChangeSize("xl"),
			wantErr: true,
		},
		{
			name:    "empty type",
			cType:   ChangeType(""),
			cSize:   SizeSmall,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StageFlow(tt.cType, tt.cSize)
			if (err != nil) != tt.wantErr {
				t.Fatalf("StageFlow(%q, %q) error = %v, wantErr = %v", tt.cType, tt.cSize, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.wantFlow) {
				t.Fatalf("StageFlow(%q, %q) returned %d stages, want %d: got %v", tt.cType, tt.cSize, len(got), len(tt.wantFlow), got)
			}
			for i, stage := range got {
				if stage != tt.wantFlow[i] {
					t.Errorf("StageFlow(%q, %q)[%d] = %q, want %q", tt.cType, tt.cSize, i, stage, tt.wantFlow[i])
				}
			}
		})
	}
}

func TestStageFlowReturnsCopy(t *testing.T) {
	flow1, err := StageFlow(TypeFix, SizeSmall)
	if err != nil {
		t.Fatal(err)
	}
	flow2, err := StageFlow(TypeFix, SizeSmall)
	if err != nil {
		t.Fatal(err)
	}

	// Mutate flow1 and verify flow2 is unaffected.
	flow1[0] = StageVerify
	if flow2[0] == StageVerify {
		t.Error("StageFlow returned a reference to the registry, not a copy")
	}
}

func TestStageFlowAllEndWithVerify(t *testing.T) {
	for ct, sizes := range FlowRegistry {
		for cs, flow := range sizes {
			last := flow[len(flow)-1]
			if last != StageVerify {
				t.Errorf("Flow %s/%s ends with %q, want %q", ct, cs, last, StageVerify)
			}
		}
	}
}

func TestStageFilename(t *testing.T) {
	tests := []struct {
		stage ChangeStage
		want  string
	}{
		{StageDescribe, "describe.md"},
		{StageScope, "scope.md"},
		{StagePropose, "propose.md"},
		{StageSpec, "spec.md"},
		{StageClarify, "clarify.md"},
		{StageDesign, "design.md"},
		{StageTasks, "tasks.md"},
		{StageVerify, "verify.md"},
		{ChangeStage("unknown"), ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.stage), func(t *testing.T) {
			got := StageFilename(tt.stage)
			if got != tt.want {
				t.Errorf("StageFilename(%q) = %q, want %q", tt.stage, got, tt.want)
			}
		})
	}
}
