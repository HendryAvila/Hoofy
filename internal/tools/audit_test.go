package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HendryAvila/Hoofy/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
)

// --- extractRequirementIDs tests ---

func TestExtractRequirementIDs_Basic(t *testing.T) {
	content := "- **FR-001**: Users can register\n- **FR-002**: Users can log in\n"
	ids := extractRequirementIDs(content)

	if len(ids) != 2 {
		t.Fatalf("got %d IDs, want 2", len(ids))
	}
	if ids[0] != "FR-001" || ids[1] != "FR-002" {
		t.Errorf("got %v, want [FR-001 FR-002]", ids)
	}
}

func TestExtractRequirementIDs_NFR(t *testing.T) {
	content := "NFR-001: Must be fast\nNFR-002: Must be secure\n"
	ids := extractRequirementIDs(content)

	if len(ids) != 2 {
		t.Fatalf("got %d IDs, want 2", len(ids))
	}
	if ids[0] != "NFR-001" || ids[1] != "NFR-002" {
		t.Errorf("got %v, want [NFR-001 NFR-002]", ids)
	}
}

func TestExtractRequirementIDs_Mixed(t *testing.T) {
	content := "FR-001 implements NFR-003, see also FR-010 and NFR-001.\n"
	ids := extractRequirementIDs(content)

	want := []string{"FR-001", "FR-010", "NFR-001", "NFR-003"}
	if len(ids) != len(want) {
		t.Fatalf("got %d IDs, want %d: %v", len(ids), len(want), ids)
	}
	for i, id := range ids {
		if id != want[i] {
			t.Errorf("ids[%d] = %q, want %q", i, id, want[i])
		}
	}
}

func TestExtractRequirementIDs_Deduplication(t *testing.T) {
	content := "FR-001 references FR-001 and FR-001 again\n"
	ids := extractRequirementIDs(content)

	if len(ids) != 1 {
		t.Fatalf("got %d IDs, want 1 (dedup)", len(ids))
	}
	if ids[0] != "FR-001" {
		t.Errorf("got %q, want FR-001", ids[0])
	}
}

func TestExtractRequirementIDs_FourDigit(t *testing.T) {
	content := "FR-1001 covers the extended scope\n"
	ids := extractRequirementIDs(content)

	if len(ids) != 1 || ids[0] != "FR-1001" {
		t.Errorf("got %v, want [FR-1001]", ids)
	}
}

func TestExtractRequirementIDs_NoMatch(t *testing.T) {
	content := "No requirement IDs here. Just plain text.\n"
	ids := extractRequirementIDs(content)

	if ids != nil {
		t.Errorf("got %v, want nil", ids)
	}
}

func TestExtractRequirementIDs_Sorted(t *testing.T) {
	content := "FR-003 then FR-001 then NFR-002\n"
	ids := extractRequirementIDs(content)

	want := []string{"FR-001", "FR-003", "NFR-002"}
	if len(ids) != len(want) {
		t.Fatalf("got %d IDs, want %d", len(ids), len(want))
	}
	for i, id := range ids {
		if id != want[i] {
			t.Errorf("ids[%d] = %q, want %q", i, id, want[i])
		}
	}
}

// --- extractRequirementsWithDescriptions tests ---

func TestExtractRequirementsWithDescriptions_Standard(t *testing.T) {
	content := "# Requirements\n\n- **FR-001**: Users can create an account\n- **FR-002**: Users can log in with email\n"
	reqs := extractRequirementsWithDescriptions(content)

	if len(reqs) != 2 {
		t.Fatalf("got %d reqs, want 2", len(reqs))
	}
	if reqs[0].ID != "FR-001" {
		t.Errorf("reqs[0].ID = %q, want FR-001", reqs[0].ID)
	}
	if !strings.Contains(reqs[0].Description, "FR-001") {
		t.Errorf("reqs[0].Description should contain the ID: %q", reqs[0].Description)
	}
	if !strings.Contains(reqs[0].Description, "Users can create an account") {
		t.Errorf("reqs[0].Description should contain the description text: %q", reqs[0].Description)
	}
}

func TestExtractRequirementsWithDescriptions_Dedup(t *testing.T) {
	content := "- **FR-001**: First mention\n- **FR-001**: Duplicate mention\n"
	reqs := extractRequirementsWithDescriptions(content)

	if len(reqs) != 1 {
		t.Fatalf("got %d reqs, want 1 (dedup)", len(reqs))
	}
}

func TestExtractRequirementsWithDescriptions_Sorted(t *testing.T) {
	content := "- **FR-003**: Third\n- **FR-001**: First\n- **NFR-001**: Non-func\n"
	reqs := extractRequirementsWithDescriptions(content)

	if len(reqs) != 3 {
		t.Fatalf("got %d reqs, want 3", len(reqs))
	}
	if reqs[0].ID != "FR-001" {
		t.Errorf("first should be FR-001, got %q", reqs[0].ID)
	}
	if reqs[1].ID != "FR-003" {
		t.Errorf("second should be FR-003, got %q", reqs[1].ID)
	}
	if reqs[2].ID != "NFR-001" {
		t.Errorf("third should be NFR-001, got %q", reqs[2].ID)
	}
}

func TestExtractRequirementsWithDescriptions_Empty(t *testing.T) {
	reqs := extractRequirementsWithDescriptions("No requirements here.\n")
	if len(reqs) != 0 {
		t.Errorf("got %d reqs, want 0", len(reqs))
	}
}

// --- scanSourceFiles tests ---

func TestScanSourceFiles_GoProject(t *testing.T) {
	root := setupGoProject(t)
	docsDir := filepath.Join(root, "docs")

	files := scanSourceFiles(root, docsDir, "")

	if len(files) == 0 {
		t.Fatal("should find source files in Go project")
	}

	// Should find .go files.
	foundGo := false
	for _, f := range files {
		if strings.HasSuffix(f.Path, ".go") {
			foundGo = true
			if f.Lines == 0 {
				t.Errorf("file %s should have >0 lines", f.Path)
			}
			if f.Size == 0 {
				t.Errorf("file %s should have >0 size", f.Path)
			}
		}
	}
	if !foundGo {
		t.Error("should find .go files")
	}
}

func TestScanSourceFiles_SkipsIgnoreDirs(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "src/app.go", "package main\n")
	writeTestFile(t, root, "node_modules/lib/index.js", "module.exports = {};\n")
	writeTestFile(t, root, ".git/hooks/pre-commit", "#!/bin/sh\n")
	writeTestFile(t, root, "vendor/dep/dep.go", "package dep\n")

	docsDir := filepath.Join(root, "docs")
	files := scanSourceFiles(root, docsDir, "")

	for _, f := range files {
		if strings.Contains(f.Path, "node_modules") {
			t.Error("should skip node_modules")
		}
		if strings.Contains(f.Path, ".git") {
			t.Error("should skip .git")
		}
		if strings.Contains(f.Path, "vendor") {
			t.Error("should skip vendor")
		}
	}

	if len(files) != 1 {
		t.Errorf("should find exactly 1 file (src/app.go), got %d", len(files))
	}
}

func TestScanSourceFiles_SkipsDocsDir(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "src/app.go", "package main\n")
	writeTestFile(t, root, "docs/design.md", "# Design\n")
	writeTestFile(t, root, "docs/something.go", "package docs\n")

	docsDir := filepath.Join(root, "docs")
	files := scanSourceFiles(root, docsDir, "")

	for _, f := range files {
		if strings.Contains(f.Path, "docs") {
			t.Errorf("should skip docs dir, found: %s", f.Path)
		}
	}
}

func TestScanSourceFiles_WithScanPath(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "pkg/a/main.go", "package a\n")
	writeTestFile(t, root, "pkg/b/main.go", "package b\n")
	writeTestFile(t, root, "other/c.go", "package other\n")

	docsDir := filepath.Join(root, "docs")
	files := scanSourceFiles(root, docsDir, "pkg")

	// Should only find files under pkg/
	for _, f := range files {
		if !strings.HasPrefix(f.Path, "pkg") {
			t.Errorf("with scan_path=pkg, should only find pkg/ files, got: %s", f.Path)
		}
	}
	if len(files) != 2 {
		t.Errorf("should find 2 files under pkg/, got %d", len(files))
	}
}

func TestScanSourceFiles_NonSourceExtensions(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "README.md", "# README\n")
	writeTestFile(t, root, "data.json", "{}\n")
	writeTestFile(t, root, "style.css", "body{}\n")

	docsDir := filepath.Join(root, "docs")
	files := scanSourceFiles(root, docsDir, "")

	if len(files) != 0 {
		t.Errorf("should find 0 source files (only non-source exts), got %d", len(files))
	}
}

func TestScanSourceFiles_Empty(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")
	files := scanSourceFiles(root, docsDir, "")

	if len(files) != 0 {
		t.Errorf("should find 0 files in empty dir, got %d", len(files))
	}
}

func TestScanSourceFiles_LineCount(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "main.go", "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n")

	docsDir := filepath.Join(root, "docs")
	files := scanSourceFiles(root, docsDir, "")

	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	if files[0].Lines != 7 {
		t.Errorf("lines = %d, want 7", files[0].Lines)
	}
}

func TestScanSourceFiles_LineCountNoTrailingNewline(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "main.go", "package main\nfunc main() {}") // no trailing newline

	docsDir := filepath.Join(root, "docs")
	files := scanSourceFiles(root, docsDir, "")

	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	if files[0].Lines != 2 {
		t.Errorf("lines = %d, want 2 (no trailing newline)", files[0].Lines)
	}
}

// --- readAuditArtifacts tests ---

func TestReadAuditArtifacts_AllExist(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")

	// Create all expected artifacts.
	for _, stage := range auditStages {
		filename := config.StageFilename(stage)
		if filename == "" {
			continue
		}
		content := "# " + string(stage) + "\n\nContent here.\n"
		writeTestFile(t, root, filepath.Join("docs", filename), content)
	}

	artifacts := readAuditArtifacts(docsDir)

	for _, a := range artifacts {
		if !a.Exists {
			t.Errorf("artifact %s should exist", a.Stage)
		}
		if a.Content == "" {
			t.Errorf("artifact %s should have content", a.Stage)
		}
		if a.Size == 0 {
			t.Errorf("artifact %s should have size > 0", a.Stage)
		}
	}
}

func TestReadAuditArtifacts_NoneExist(t *testing.T) {
	docsDir := filepath.Join(t.TempDir(), "docs")
	// Don't create the directory — all artifacts should be missing.

	artifacts := readAuditArtifacts(docsDir)

	for _, a := range artifacts {
		if a.Exists {
			t.Errorf("artifact %s should not exist", a.Stage)
		}
		if a.Content != "" {
			t.Errorf("artifact %s should have empty content", a.Stage)
		}
	}
}

func TestReadAuditArtifacts_Partial(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")

	// Only create requirements and design.
	writeTestFile(t, root, "docs/requirements.md", "# Requirements\n- FR-001: Test\n")
	writeTestFile(t, root, "docs/design.md", "# Design\n## Architecture\n")

	artifacts := readAuditArtifacts(docsDir)

	existCount := 0
	for _, a := range artifacts {
		if a.Exists {
			existCount++
		}
	}

	if existCount != 2 {
		t.Errorf("existCount = %d, want 2", existCount)
	}
}

func TestReadAuditArtifacts_StageCount(t *testing.T) {
	docsDir := filepath.Join(t.TempDir(), "docs")
	artifacts := readAuditArtifacts(docsDir)

	if len(artifacts) != len(auditStages) {
		t.Errorf("got %d artifacts, want %d (one per audited stage)", len(artifacts), len(auditStages))
	}
}

// --- buildAuditReport tests ---

func TestBuildAuditReport_Standard(t *testing.T) {
	root := "/tmp/test-project"
	docsDir := "/tmp/test-project/docs"

	artifacts := []auditArtifact{
		{Stage: config.StageSpecify, Filename: "requirements.md", Content: "# Req\n- **FR-001**: Create account\n- **FR-002**: Login\n", Size: 50, Exists: true},
		{Stage: config.StageDesign, Filename: "design.md", Content: "# Design\nCovers FR-001, FR-002\n", Size: 30, Exists: true},
		{Stage: config.StagePrinciples, Filename: "principles.md", Exists: false},
	}

	sourceFiles := []auditSourceFile{
		{Path: "cmd/main.go", Size: 100, Lines: 10},
		{Path: "internal/handler.go", Size: 500, Lines: 50},
	}

	report := buildAuditReport(root, docsDir, artifacts, sourceFiles, "standard", 50*time.Millisecond)

	// Header.
	if !strings.Contains(report, "# Spec Audit Report") {
		t.Error("should have report title")
	}
	if !strings.Contains(report, "AI Instructions") {
		t.Error("should have AI instructions block")
	}

	// Metadata.
	if !strings.Contains(report, root) {
		t.Error("should include project root")
	}
	if !strings.Contains(report, "Source files found") {
		t.Error("should report source file count")
	}
	if !strings.Contains(report, "2/3") {
		t.Error("should report artifact count (2 existing / 3 total)")
	}

	// Artifacts.
	if !strings.Contains(report, "requirements.md") {
		t.Error("should include requirements.md")
	}
	if !strings.Contains(report, "Not found") {
		t.Error("should mark missing artifacts")
	}

	// Requirement IDs table.
	if !strings.Contains(report, "FR-001") {
		t.Error("should extract FR-001 from requirements")
	}
	if !strings.Contains(report, "FR-002") {
		t.Error("should extract FR-002 from requirements")
	}

	// Cross-references.
	if !strings.Contains(report, "Cross-Referenced") {
		t.Error("should have cross-reference section for IDs found in design")
	}

	// Source files.
	if !strings.Contains(report, "cmd/main.go") {
		t.Error("should list source files")
	}
	if !strings.Contains(report, "internal/handler.go") {
		t.Error("should list all source files")
	}
}

func TestBuildAuditReport_Summary(t *testing.T) {
	artifacts := []auditArtifact{
		{Stage: config.StageSpecify, Filename: "requirements.md", Content: "# Req\n- FR-001: Test\n", Size: 30, Exists: true},
	}
	sourceFiles := []auditSourceFile{
		{Path: "cmd/main.go", Size: 100, Lines: 10},
		{Path: "cmd/util.go", Size: 200, Lines: 20},
	}

	report := buildAuditReport("/tmp", "/tmp/docs", artifacts, sourceFiles, "summary", time.Millisecond)

	// Summary artifacts: should show existence but NOT content.
	if !strings.Contains(report, "✅ Exists") {
		t.Error("summary should indicate artifact exists")
	}
	if strings.Contains(report, "```markdown") {
		t.Error("summary should NOT include code fences with content")
	}

	// Summary source files: grouped by directory.
	if !strings.Contains(report, "Directory") {
		t.Error("summary should group source files by directory")
	}
	if !strings.Contains(report, "File Count") {
		t.Error("summary should show file counts per directory")
	}
}

func TestBuildAuditReport_Full(t *testing.T) {
	content := "# Requirements\n\n- **FR-001**: Full content test\n"
	artifacts := []auditArtifact{
		{Stage: config.StageSpecify, Filename: "requirements.md", Content: content, Size: int64(len(content)), Exists: true},
	}

	report := buildAuditReport("/tmp", "/tmp/docs", artifacts, nil, "full", time.Millisecond)

	// Full: should include complete content in code fences.
	if !strings.Contains(report, "```markdown") {
		t.Error("full should include code fences")
	}
	if !strings.Contains(report, "Full content test") {
		t.Error("full should include complete artifact content")
	}
}

func TestBuildAuditReport_NoArtifacts(t *testing.T) {
	var artifacts []auditArtifact
	var sourceFiles []auditSourceFile

	report := buildAuditReport("/tmp", "/tmp/docs", artifacts, sourceFiles, "standard", time.Millisecond)

	if !strings.Contains(report, "No requirement IDs found") {
		t.Error("should indicate no requirement IDs")
	}
	if !strings.Contains(report, "No source files found") {
		t.Error("should indicate no source files")
	}
}

func TestBuildAuditReport_NoCrossReferences(t *testing.T) {
	// Only requirements, no other artifacts with cross-references.
	artifacts := []auditArtifact{
		{Stage: config.StageSpecify, Filename: "requirements.md", Content: "- FR-001: Test\n", Size: 15, Exists: true},
	}

	report := buildAuditReport("/tmp", "/tmp/docs", artifacts, nil, "standard", time.Millisecond)

	if strings.Contains(report, "Cross-Referenced") {
		t.Error("should NOT have cross-reference section when no other artifacts reference IDs")
	}
}

func TestBuildAuditReport_PipeEscaping(t *testing.T) {
	artifacts := []auditArtifact{
		{Stage: config.StageSpecify, Filename: "requirements.md", Content: "- **FR-001**: Input | Output spec\n", Size: 35, Exists: true},
	}

	report := buildAuditReport("/tmp", "/tmp/docs", artifacts, nil, "standard", time.Millisecond)

	// Pipe in description should be escaped for markdown table.
	if !strings.Contains(report, `\|`) {
		t.Error("should escape pipe characters in requirement descriptions")
	}
}

// --- AuditTool handler tests ---

func setupAuditProject(t *testing.T) (string, func()) {
	t.Helper()
	root := t.TempDir()

	// Create a project with source files and docs.
	writeTestFile(t, root, "main.go", "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n")
	writeTestFile(t, root, "internal/handler.go", "package handler\n\n// Handle processes requests.\nfunc Handle() {}\n")

	// Create docs with spec artifacts.
	writeTestFile(t, root, "docs/requirements.md", "# Requirements\n\n- **FR-001**: Users can register\n- **FR-002**: Users can log in\n- **NFR-001**: Latency < 200ms\n")
	writeTestFile(t, root, "docs/design.md", "# Design\n\n## Components\n\nHandlerModule covers FR-001, FR-002.\n")

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("setup: getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("setup: chdir: %v", err)
	}
	cleanup := func() { _ = os.Chdir(origDir) }

	return root, cleanup
}

func TestAuditTool_Handle_Success(t *testing.T) {
	_, cleanup := setupAuditProject(t)
	defer cleanup()

	tool := NewAuditTool()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}

	result, err := tool.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	if isErrorResult(result) {
		t.Fatalf("expected success, got error: %s", getResultText(result))
	}

	text := getResultText(result)

	// Report structure.
	if !strings.Contains(text, "# Spec Audit Report") {
		t.Error("should have report title")
	}
	if !strings.Contains(text, "AI Instructions") {
		t.Error("should have AI instructions")
	}

	// Should find source files.
	if !strings.Contains(text, "main.go") {
		t.Error("should list main.go")
	}

	// Should find requirement IDs.
	if !strings.Contains(text, "FR-001") {
		t.Error("should extract FR-001")
	}
	if !strings.Contains(text, "FR-002") {
		t.Error("should extract FR-002")
	}
	if !strings.Contains(text, "NFR-001") {
		t.Error("should extract NFR-001")
	}

	// Should have token footer.
	if !strings.Contains(text, "tokens") {
		t.Error("should have token footer")
	}
}

func TestAuditTool_Handle_DetailLevels(t *testing.T) {
	_, cleanup := setupAuditProject(t)
	defer cleanup()

	tool := NewAuditTool()

	// Standard.
	reqStd := mcp.CallToolRequest{}
	reqStd.Params.Arguments = map[string]interface{}{"detail_level": "standard"}
	resultStd, _ := tool.Handle(context.Background(), reqStd)
	textStd := getResultText(resultStd)

	// Summary.
	reqSum := mcp.CallToolRequest{}
	reqSum.Params.Arguments = map[string]interface{}{"detail_level": "summary"}
	resultSum, _ := tool.Handle(context.Background(), reqSum)
	textSum := getResultText(resultSum)

	// Summary should be shorter than standard.
	if len(textSum) >= len(textStd) {
		t.Error("summary should be shorter than standard")
	}
}

func TestAuditTool_Handle_ScanPath(t *testing.T) {
	root, cleanup := setupAuditProject(t)
	defer cleanup()

	// Add files in a subdirectory.
	writeTestFile(t, root, "services/api/api.go", "package api\nfunc Serve() {}\n")
	writeTestFile(t, root, "services/api/handler.go", "package api\nfunc Handle() {}\n")

	tool := NewAuditTool()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"scan_path": "services/api",
	}

	result, err := tool.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	if isErrorResult(result) {
		t.Fatalf("expected success, got error: %s", getResultText(result))
	}

	text := getResultText(result)
	// Should find files in the scanned subdirectory.
	if !strings.Contains(text, "api.go") {
		t.Error("should find api.go in scan_path")
	}
}

func TestAuditTool_Handle_ScanPath_Invalid(t *testing.T) {
	_, cleanup := setupAuditProject(t)
	defer cleanup()

	tool := NewAuditTool()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"scan_path": "nonexistent/path",
	}

	result, err := tool.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	if !isErrorResult(result) {
		t.Error("should return error for nonexistent scan_path")
	}
	text := getResultText(result)
	if !strings.Contains(text, "not found") {
		t.Errorf("error should mention 'not found': %s", text)
	}
}

func TestAuditTool_Handle_ScanPath_NotDir(t *testing.T) {
	root, cleanup := setupAuditProject(t)
	defer cleanup()

	writeTestFile(t, root, "file.txt", "just a file")

	tool := NewAuditTool()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"scan_path": "file.txt",
	}

	result, err := tool.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	if !isErrorResult(result) {
		t.Error("should return error when scan_path is a file")
	}
	text := getResultText(result)
	if !strings.Contains(text, "not a directory") {
		t.Errorf("error should mention 'not a directory': %s", text)
	}
}

func TestAuditTool_Handle_NoDocsDir(t *testing.T) {
	// Project with no docs directory at all.
	root := t.TempDir()
	writeTestFile(t, root, "main.go", "package main\nfunc main() {}\n")

	origDir, _ := os.Getwd()
	_ = os.Chdir(root)
	defer func() { _ = os.Chdir(origDir) }()

	tool := NewAuditTool()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}

	result, err := tool.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	// Should succeed gracefully — just report no artifacts found.
	if isErrorResult(result) {
		t.Fatalf("expected success even without docs, got error: %s", getResultText(result))
	}

	text := getResultText(result)
	if !strings.Contains(text, "Not found") {
		t.Error("should report artifacts as not found")
	}
	// Should still find source files.
	if !strings.Contains(text, "main.go") {
		t.Error("should still scan and find source files")
	}
}

func TestAuditTool_Handle_EmptyProject(t *testing.T) {
	root := t.TempDir()

	origDir, _ := os.Getwd()
	_ = os.Chdir(root)
	defer func() { _ = os.Chdir(origDir) }()

	tool := NewAuditTool()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}

	result, err := tool.Handle(context.Background(), req)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	if isErrorResult(result) {
		t.Fatalf("expected success, got error: %s", getResultText(result))
	}

	text := getResultText(result)
	if !strings.Contains(text, "# Spec Audit Report") {
		t.Error("should still produce a report")
	}
	if !strings.Contains(text, "No source files found") {
		t.Error("should indicate no source files")
	}
}

// --- Definition test ---

func TestAuditTool_Definition(t *testing.T) {
	tool := NewAuditTool()
	def := tool.Definition()

	if def.Name != "sdd_audit" {
		t.Errorf("tool name = %q, want %q", def.Name, "sdd_audit")
	}
	if def.Description == "" {
		t.Error("definition should have a description")
	}
}
