package tools

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/HendryAvila/Hoofy/internal/config"
	"github.com/HendryAvila/Hoofy/internal/templates"
	"github.com/mark3labs/mcp-go/mcp"
)

// --- BusinessRulesTool ---

func TestBusinessRulesTool_Handle_Success(t *testing.T) {
	tmpDir, cleanup := setupTestProjectAtStage(t, config.ModeGuided, config.StageBusinessRules)
	defer cleanup()

	// Write requirements (required by business rules).
	reqPath := config.StagePath(tmpDir, config.StageSpecify)
	if err := writeStageFile(reqPath, "# Requirements\n\n- FR-001: Users can sign up\n- FR-002: Users can log in"); err != nil {
		t.Fatalf("write requirements: %v", err)
	}

	store := config.NewFileStore()
	renderer, _ := templates.NewRenderer()
	tool := NewBusinessRulesTool(store, renderer)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"definitions": "- **Customer**: A person who has completed at least one purchase\n- **Order**: A confirmed request for products",
		"facts":       "- A Customer has exactly one Account\n- An Account can have zero or more Orders",
		"constraints": "- When an Order total exceeds $500, Then manager approval is required",
	}

	result, err := tool.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	if isErrorResult(result) {
		t.Fatalf("expected success, got error: %s", getResultText(result))
	}

	text := getResultText(result)
	if !strings.Contains(text, "Business Rules Documented") {
		t.Error("result should contain 'Business Rules Documented'")
	}
	if !strings.Contains(text, "Customer") {
		t.Error("result should contain definition content")
	}
	if !strings.Contains(text, "Account") {
		t.Error("result should contain facts content")
	}
	if !strings.Contains(text, "manager approval") {
		t.Error("result should contain constraints content")
	}
}

func TestBusinessRulesTool_Handle_MissingRequiredFields(t *testing.T) {
	tmpDir, cleanup := setupTestProjectAtStage(t, config.ModeGuided, config.StageBusinessRules)
	defer cleanup()

	reqPath := config.StagePath(tmpDir, config.StageSpecify)
	if err := writeStageFile(reqPath, "# Requirements\n\nSome content."); err != nil {
		t.Fatalf("write requirements: %v", err)
	}

	store := config.NewFileStore()
	renderer, _ := templates.NewRenderer()
	tool := NewBusinessRulesTool(store, renderer)

	tests := []struct {
		name   string
		args   map[string]interface{}
		errMsg string
	}{
		{
			name:   "missing definitions",
			args:   map[string]interface{}{"facts": "some facts", "constraints": "some constraints"},
			errMsg: "definitions",
		},
		{
			name:   "missing facts",
			args:   map[string]interface{}{"definitions": "some defs", "constraints": "some constraints"},
			errMsg: "facts",
		},
		{
			name:   "missing constraints",
			args:   map[string]interface{}{"definitions": "some defs", "facts": "some facts"},
			errMsg: "constraints",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := mcp.CallToolRequest{}
			req.Params.Arguments = tt.args

			result, err := tool.Handle(context.Background(), req)
			if err != nil {
				t.Fatalf("Handle failed: %v", err)
			}
			if !isErrorResult(result) {
				t.Error("should return error when required field is missing")
			}
			text := getResultText(result)
			if !strings.Contains(text, tt.errMsg) {
				t.Errorf("error should mention '%s': %s", tt.errMsg, text)
			}
		})
	}
}

func TestBusinessRulesTool_Handle_WrongStage(t *testing.T) {
	_, cleanup := setupTestProjectAtStage(t, config.ModeGuided, config.StageClarify)
	defer cleanup()

	store := config.NewFileStore()
	renderer, _ := templates.NewRenderer()
	tool := NewBusinessRulesTool(store, renderer)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"definitions": "some defs",
		"facts":       "some facts",
		"constraints": "some constraints",
	}

	result, err := tool.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	if !isErrorResult(result) {
		t.Error("should return error when at wrong stage")
	}
	text := getResultText(result)
	if !strings.Contains(text, "wrong pipeline stage") {
		t.Errorf("error should mention wrong stage: %s", text)
	}
}

func TestBusinessRulesTool_Handle_EmptyRequirements(t *testing.T) {
	_, cleanup := setupTestProjectAtStage(t, config.ModeGuided, config.StageBusinessRules)
	defer cleanup()

	store := config.NewFileStore()
	renderer, _ := templates.NewRenderer()
	tool := NewBusinessRulesTool(store, renderer)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"definitions": "some defs",
		"facts":       "some facts",
		"constraints": "some constraints",
	}

	result, err := tool.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	if !isErrorResult(result) {
		t.Error("should return error when requirements are empty")
	}
}

func TestBusinessRulesTool_Handle_AdvancesPipeline(t *testing.T) {
	tmpDir, cleanup := setupTestProjectAtStage(t, config.ModeGuided, config.StageBusinessRules)
	defer cleanup()

	reqPath := config.StagePath(tmpDir, config.StageSpecify)
	if err := writeStageFile(reqPath, "# Requirements\n\nSome content."); err != nil {
		t.Fatalf("write requirements: %v", err)
	}

	store := config.NewFileStore()
	renderer, _ := templates.NewRenderer()
	tool := NewBusinessRulesTool(store, renderer)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"definitions": "- **User**: A registered person",
		"facts":       "- A User has one Profile",
		"constraints": "- When User is inactive for 90 days, Then account is deactivated",
	}

	_, err := tool.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	cfg, _ := store.Load(tmpDir)
	if cfg.CurrentStage != config.StageClarify {
		t.Errorf("stage should be clarify after business-rules, got: %s", cfg.CurrentStage)
	}
}

func TestBusinessRulesTool_Handle_OptionalFieldsDefault(t *testing.T) {
	tmpDir, cleanup := setupTestProjectAtStage(t, config.ModeGuided, config.StageBusinessRules)
	defer cleanup()

	reqPath := config.StagePath(tmpDir, config.StageSpecify)
	if err := writeStageFile(reqPath, "# Requirements\n\nSome content."); err != nil {
		t.Fatalf("write requirements: %v", err)
	}

	store := config.NewFileStore()
	renderer, _ := templates.NewRenderer()
	tool := NewBusinessRulesTool(store, renderer)

	// Only required fields — derivations and glossary should be omitted.
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"definitions": "- **User**: A registered person",
		"facts":       "- A User has one Profile",
		"constraints": "- When User logs in, Then last_login is updated",
	}

	result, err := tool.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	if isErrorResult(result) {
		t.Fatalf("expected success, got error: %s", getResultText(result))
	}

	text := getResultText(result)
	// Verify that the output does NOT include derivations or glossary sections
	// when they were not provided (conditional rendering in template).
	if !strings.Contains(text, "User") {
		t.Error("result should contain definition content")
	}
}

func TestBusinessRulesTool_Handle_WithOptionalFields(t *testing.T) {
	tmpDir, cleanup := setupTestProjectAtStage(t, config.ModeGuided, config.StageBusinessRules)
	defer cleanup()

	reqPath := config.StagePath(tmpDir, config.StageSpecify)
	if err := writeStageFile(reqPath, "# Requirements\n\nSome content."); err != nil {
		t.Fatalf("write requirements: %v", err)
	}

	store := config.NewFileStore()
	renderer, _ := templates.NewRenderer()
	tool := NewBusinessRulesTool(store, renderer)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"definitions": "- **Customer**: A person who has completed a purchase",
		"facts":       "- A Customer has exactly one Account",
		"constraints": "- When Order exceeds $500, Then approval required",
		"derivations": "- A Customer is \"premium\" when total spend exceeds $10,000",
		"glossary":    "- **SKU**: Stock Keeping Unit\n- **AOV**: Average Order Value",
	}

	result, err := tool.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	if isErrorResult(result) {
		t.Fatalf("expected success, got error: %s", getResultText(result))
	}

	text := getResultText(result)
	if !strings.Contains(text, "premium") {
		t.Error("result should contain derivations content")
	}
	if !strings.Contains(text, "SKU") {
		t.Error("result should contain glossary content")
	}
}

func TestBusinessRulesTool_Handle_FileWritten(t *testing.T) {
	tmpDir, cleanup := setupTestProjectAtStage(t, config.ModeGuided, config.StageBusinessRules)
	defer cleanup()

	reqPath := config.StagePath(tmpDir, config.StageSpecify)
	if err := writeStageFile(reqPath, "# Requirements\n\nSome content."); err != nil {
		t.Fatalf("write requirements: %v", err)
	}

	store := config.NewFileStore()
	renderer, _ := templates.NewRenderer()
	tool := NewBusinessRulesTool(store, renderer)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"definitions": "- **Widget**: A thing",
		"facts":       "- A Widget has a name",
		"constraints": "- When Widget is deleted, Then related items are orphaned",
	}

	_, err := tool.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	// Verify the file was written.
	rulesPath := config.StagePath(tmpDir, config.StageBusinessRules)
	data, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatalf("business-rules.md should exist: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Widget") {
		t.Error("file content should contain definition")
	}
	if !strings.Contains(content, "Business Rules") {
		t.Error("file content should contain 'Business Rules' heading")
	}
}

func TestBusinessRulesTool_Handle_NotifiesBridge(t *testing.T) {
	tmpDir, cleanup := setupTestProjectAtStage(t, config.ModeGuided, config.StageBusinessRules)
	defer cleanup()

	reqPath := config.StagePath(tmpDir, config.StageSpecify)
	if err := writeStageFile(reqPath, "# Requirements\n\nSome content."); err != nil {
		t.Fatalf("write requirements: %v", err)
	}

	store := config.NewFileStore()
	renderer, _ := templates.NewRenderer()
	tool := NewBusinessRulesTool(store, renderer)
	spy := &spyObserver{}
	tool.SetBridge(spy)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"definitions": "- **User**: A person",
		"facts":       "- A User has roles",
		"constraints": "- When User has no roles, Then access is denied",
	}

	result, err := tool.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	if isErrorResult(result) {
		t.Fatalf("unexpected error: %s", getResultText(result))
	}

	if len(spy.calls) != 1 {
		t.Fatalf("expected 1 bridge call, got %d", len(spy.calls))
	}
	if spy.calls[0].stage != config.StageBusinessRules {
		t.Errorf("stage = %q, want business-rules", spy.calls[0].stage)
	}
	if spy.calls[0].projectName != "test-project" {
		t.Errorf("projectName = %q, want test-project", spy.calls[0].projectName)
	}
}

func TestBusinessRulesTool_Handle_NilBridge_NoError(t *testing.T) {
	tmpDir, cleanup := setupTestProjectAtStage(t, config.ModeGuided, config.StageBusinessRules)
	defer cleanup()

	reqPath := config.StagePath(tmpDir, config.StageSpecify)
	if err := writeStageFile(reqPath, "# Requirements\n\nSome content."); err != nil {
		t.Fatalf("write requirements: %v", err)
	}

	store := config.NewFileStore()
	renderer, _ := templates.NewRenderer()
	tool := NewBusinessRulesTool(store, renderer)
	// Do NOT set bridge — it should be nil and still work.

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"definitions": "- **User**: A person",
		"facts":       "- A User has roles",
		"constraints": "- When User is suspended, Then access is denied",
	}

	result, err := tool.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle failed with nil bridge: %v", err)
	}
	if isErrorResult(result) {
		t.Fatalf("unexpected error with nil bridge: %s", getResultText(result))
	}
}

func TestBusinessRulesTool_SetBridge(t *testing.T) {
	store := config.NewFileStore()
	renderer, _ := templates.NewRenderer()
	tool := NewBusinessRulesTool(store, renderer)
	spy := &spyObserver{}

	// SetBridge should not panic and should be callable.
	tool.SetBridge(spy)
	tool.SetBridge(nil) // nil should also be safe
}

func TestBusinessRulesTool_Handle_ExpertMode(t *testing.T) {
	tmpDir, cleanup := setupTestProjectAtStage(t, config.ModeExpert, config.StageBusinessRules)
	defer cleanup()

	reqPath := config.StagePath(tmpDir, config.StageSpecify)
	if err := writeStageFile(reqPath, "# Requirements\n\nSome content."); err != nil {
		t.Fatalf("write requirements: %v", err)
	}

	store := config.NewFileStore()
	renderer, _ := templates.NewRenderer()
	tool := NewBusinessRulesTool(store, renderer)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"definitions": "- **Entity**: A domain object",
		"facts":       "- An Entity has an ID",
		"constraints": "- When Entity is created, Then ID is immutable",
	}

	result, err := tool.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	if isErrorResult(result) {
		t.Fatalf("expected success in expert mode, got error: %s", getResultText(result))
	}

	text := getResultText(result)
	if !strings.Contains(text, "Business Rules Documented") {
		t.Error("result should contain 'Business Rules Documented' in expert mode")
	}

	// Verify pipeline advanced.
	cfg, _ := store.Load(tmpDir)
	if cfg.CurrentStage != config.StageClarify {
		t.Errorf("stage should be clarify after business-rules in expert mode, got: %s", cfg.CurrentStage)
	}
}
