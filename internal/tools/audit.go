// Package tools — see helpers.go for package doc.
//
// audit.go implements the sdd_audit scanner tool.
// It reads spec artifacts from the docs directory and scans project
// source files to produce a structured markdown report. The AI then
// compares specs against actual code — the tool is purely a data
// collector, never an interpreter.
//
// Design: read-only scanner (like reverse_engineer.go).
// No config.Store dependency — works without hoofy.json.
package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/HendryAvila/Hoofy/internal/config"
	"github.com/HendryAvila/Hoofy/internal/memory"
	"github.com/mark3labs/mcp-go/mcp"
)

// --- Requirement ID extraction ---

// requirementIDPattern matches FR-NNN, NFR-NNN, and FR-NNNN patterns.
var requirementIDPattern = regexp.MustCompile(`\b((?:FR|NFR)-\d{3,4})\b`)

// extractRequirementIDs parses FR/NFR IDs from markdown content.
// Returns a deduplicated, sorted list of IDs found.
func extractRequirementIDs(content string) []string {
	matches := requirementIDPattern.FindAllString(content, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]bool, len(matches))
	var unique []string
	for _, m := range matches {
		if !seen[m] {
			seen[m] = true
			unique = append(unique, m)
		}
	}
	sort.Strings(unique)
	return unique
}

// requirementWithDescription holds an ID and its surrounding context.
type requirementWithDescription struct {
	ID          string
	Description string // the full line where the ID appears
}

// extractRequirementsWithDescriptions parses FR/NFR IDs and their
// descriptions from requirements.md content. Each requirement line
// typically looks like: "- **FR-001**: Users can create an account"
func extractRequirementsWithDescriptions(content string) []requirementWithDescription {
	lines := strings.Split(content, "\n")
	var results []requirementWithDescription
	seen := make(map[string]bool)

	for _, line := range lines {
		ids := requirementIDPattern.FindAllString(line, -1)
		for _, id := range ids {
			if seen[id] {
				continue
			}
			seen[id] = true
			desc := strings.TrimSpace(line)
			// Strip leading list markers.
			desc = strings.TrimLeft(desc, "- *")
			desc = strings.TrimSpace(desc)
			results = append(results, requirementWithDescription{
				ID:          id,
				Description: desc,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].ID < results[j].ID
	})
	return results
}

// --- Source file scanning ---

// sourceFileExtensions are file extensions considered "source code".
var sourceFileExtensions = map[string]bool{
	".go": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true,
	".py": true, ".rs": true, ".rb": true, ".java": true, ".kt": true,
	".cs": true, ".cpp": true, ".c": true, ".h": true, ".hpp": true,
	".swift": true, ".dart": true, ".ex": true, ".exs": true,
	".php": true, ".scala": true, ".vue": true, ".svelte": true,
}

// auditSourceFile holds metadata about a scanned source file.
type auditSourceFile struct {
	Path  string
	Size  int64
	Lines int
}

// scanSourceFiles walks the project tree and collects source file metadata.
// Respects ignoreDirs (shared with reverse_engineer.go) and skips the docs dir.
func scanSourceFiles(root, docsDir, scanPath string) []auditSourceFile {
	scanRoot := root
	if scanPath != "" {
		scanRoot = filepath.Join(root, scanPath)
	}

	// Resolve the docs directory name relative to root for skipping.
	docsBase := filepath.Base(docsDir)

	var files []auditSourceFile

	_ = filepath.WalkDir(scanRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // graceful degradation
		}

		if d.IsDir() {
			name := d.Name()
			// Skip common noise directories + the docs directory itself.
			if ignoreDirs[name] || name == docsBase {
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(d.Name())
		if !sourceFileExtensions[ext] {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		rel, _ := filepath.Rel(root, path)

		lines := 0
		if info.Size() <= maxFileSize {
			if data, err := os.ReadFile(path); err == nil {
				lines = strings.Count(string(data), "\n")
				if len(data) > 0 && data[len(data)-1] != '\n' {
					lines++ // last line without newline
				}
			}
		}

		files = append(files, auditSourceFile{
			Path:  rel,
			Size:  info.Size(),
			Lines: lines,
		})
		return nil
	})

	return files
}

// --- Spec artifact reading ---

// auditArtifact holds a read spec artifact's content and metadata.
type auditArtifact struct {
	Stage    config.Stage
	Filename string
	Content  string
	Size     int64
	Exists   bool
}

// auditStages are the spec stages the audit tool reads.
var auditStages = []config.Stage{
	config.StagePrinciples,
	config.StageCharter,
	config.StageSpecify,
	config.StageBusinessRules,
	config.StageDesign,
	config.StageTasks,
}

// readAuditArtifacts reads all spec artifacts from the docs directory.
func readAuditArtifacts(docsDir string) []auditArtifact {
	var artifacts []auditArtifact

	for _, stage := range auditStages {
		filename := config.StageFilename(stage)
		if filename == "" {
			continue
		}

		path := filepath.Join(docsDir, filename)
		a := auditArtifact{
			Stage:    stage,
			Filename: filename,
		}

		info, err := os.Stat(path)
		if err != nil {
			artifacts = append(artifacts, a)
			continue
		}

		a.Size = info.Size()
		data, err := os.ReadFile(path)
		if err != nil {
			artifacts = append(artifacts, a)
			continue
		}

		a.Exists = true
		a.Content = string(data)
		artifacts = append(artifacts, a)
	}

	return artifacts
}

// --- Report builder ---

// buildAuditReport assembles the structured markdown report from artifacts
// and source files. The AI receives this and performs the actual comparison.
func buildAuditReport(
	root string,
	docsDir string,
	artifacts []auditArtifact,
	sourceFiles []auditSourceFile,
	detailLevel string,
	scanDuration time.Duration,
) string {
	var report strings.Builder

	// --- Header + AI instructions ---
	report.WriteString("# Spec Audit Report\n\n")
	report.WriteString("> **AI Instructions**: Compare the spec IDs and descriptions below ")
	report.WriteString("against the source files listed. Identify:\n")
	report.WriteString("> 1. Requirements implemented in code but not in specs (undocumented features)\n")
	report.WriteString("> 2. Requirements in specs but not evidenced in code (unimplemented or dead specs)\n")
	report.WriteString("> 3. Inconsistencies between spec descriptions and implementation\n")
	report.WriteString("> 4. Tasks marked incomplete that appear implemented\n\n")

	// --- Metadata ---
	report.WriteString("## Scan Metadata\n\n")
	fmt.Fprintf(&report, "- **Project root**: `%s`\n", root)
	fmt.Fprintf(&report, "- **Docs directory**: `%s`\n", docsDir)
	fmt.Fprintf(&report, "- **Source files found**: %d\n", len(sourceFiles))

	totalLines := 0
	for _, f := range sourceFiles {
		totalLines += f.Lines
	}
	fmt.Fprintf(&report, "- **Total source lines**: %d\n", totalLines)

	existingCount := 0
	for _, a := range artifacts {
		if a.Exists {
			existingCount++
		}
	}
	fmt.Fprintf(&report, "- **Spec artifacts found**: %d/%d\n", existingCount, len(artifacts))
	fmt.Fprintf(&report, "- **Scan duration**: %s\n", scanDuration.Round(time.Millisecond))
	fmt.Fprintf(&report, "- **Detail level**: %s\n\n", detailLevel)

	// --- Spec Artifacts ---
	report.WriteString("## Spec Artifacts\n\n")

	for _, a := range artifacts {
		if !a.Exists {
			fmt.Fprintf(&report, "### %s\n\n_Not found_ — `%s` does not exist.\n\n",
				config.Stages[a.Stage].Name, a.Filename)
			continue
		}

		fmt.Fprintf(&report, "### %s (`%s`, %d bytes)\n\n", config.Stages[a.Stage].Name, a.Filename, a.Size)

		switch detailLevel {
		case "summary":
			// Just report existence and size.
			fmt.Fprintf(&report, "✅ Exists (%d bytes)\n\n", a.Size)
		case "full":
			// Full content.
			report.WriteString("```markdown\n")
			report.WriteString(a.Content)
			report.WriteString("\n```\n\n")
		default: // "standard"
			// Truncated content.
			content := truncateContent(a.Content, 3000)
			report.WriteString("```markdown\n")
			report.WriteString(content)
			report.WriteString("\n```\n\n")
		}
	}

	// --- Requirement IDs ---
	report.WriteString("## Requirement IDs\n\n")

	// Collect all requirement IDs from all artifacts.
	var allReqs []requirementWithDescription
	for _, a := range artifacts {
		if !a.Exists {
			continue
		}
		if a.Stage == config.StageSpecify {
			// From requirements.md, extract with descriptions.
			allReqs = append(allReqs, extractRequirementsWithDescriptions(a.Content)...)
		}
	}

	// Also collect IDs from other artifacts (cross-references).
	var crossRefIDs []string
	for _, a := range artifacts {
		if !a.Exists || a.Stage == config.StageSpecify {
			continue
		}
		ids := extractRequirementIDs(a.Content)
		crossRefIDs = append(crossRefIDs, ids...)
	}

	if len(allReqs) > 0 {
		report.WriteString("### From Requirements\n\n")
		report.WriteString("| ID | Description |\n")
		report.WriteString("|---|---|\n")
		for _, r := range allReqs {
			// Escape pipes in description.
			desc := strings.ReplaceAll(r.Description, "|", "\\|")
			fmt.Fprintf(&report, "| %s | %s |\n", r.ID, desc)
		}
		report.WriteString("\n")
	} else {
		report.WriteString("_No requirement IDs found in specs._\n\n")
	}

	if len(crossRefIDs) > 0 {
		// Deduplicate.
		seen := make(map[string]bool)
		var unique []string
		for _, id := range crossRefIDs {
			if !seen[id] {
				seen[id] = true
				unique = append(unique, id)
			}
		}
		sort.Strings(unique)

		report.WriteString("### Cross-Referenced in Other Artifacts\n\n")
		report.WriteString("IDs found in design, tasks, business rules, etc.: ")
		report.WriteString(strings.Join(unique, ", "))
		report.WriteString("\n\n")
	}

	// --- Source Files ---
	report.WriteString("## Source Files\n\n")

	if len(sourceFiles) == 0 {
		report.WriteString("_No source files found._\n\n")
	} else {
		switch detailLevel {
		case "summary":
			// Group by top-level directory.
			dirCounts := make(map[string]int)
			for _, f := range sourceFiles {
				parts := strings.SplitN(f.Path, string(filepath.Separator), 2)
				dir := parts[0]
				if len(parts) == 1 {
					dir = "." // root-level files
				}
				dirCounts[dir]++
			}
			dirs := make([]string, 0, len(dirCounts))
			for d := range dirCounts {
				dirs = append(dirs, d)
			}
			sort.Strings(dirs)

			report.WriteString("| Directory | File Count |\n")
			report.WriteString("|---|---|\n")
			for _, d := range dirs {
				fmt.Fprintf(&report, "| `%s` | %d |\n", d, dirCounts[d])
			}
			report.WriteString("\n")

		default: // "standard" and "full"
			report.WriteString("| File | Lines | Size |\n")
			report.WriteString("|---|---|---|\n")
			for _, f := range sourceFiles {
				fmt.Fprintf(&report, "| `%s` | %d | %d B |\n", f.Path, f.Lines, f.Size)
			}
			report.WriteString("\n")
		}
	}

	return report.String()
}

// --- MCP Tool handler ---

// AuditTool handles the sdd_audit MCP tool.
// It reads spec artifacts and scans source files to produce a structured
// report for the AI to analyze. Read-only — never writes files.
type AuditTool struct{}

// NewAuditTool creates an AuditTool.
// No dependencies — pure filesystem scanner.
func NewAuditTool() *AuditTool {
	return &AuditTool{}
}

// Definition returns the MCP tool definition for registration.
func (t *AuditTool) Definition() mcp.Tool {
	return mcp.NewTool("sdd_audit",
		mcp.WithDescription(
			"Compare project specs against actual source code. "+
				"Reads spec artifacts (requirements, business rules, design, tasks) "+
				"and scans source files to produce a structured audit report. "+
				"The AI then analyzes this report to find discrepancies between "+
				"specs and implementation. READ-ONLY — never writes files. "+
				"Works without hoofy.json for ad-hoc audits.",
		),
		mcp.WithString("scan_path",
			mcp.Description("Subdirectory to scan for source files instead of project root. "+
				"Useful for monorepos where you want to audit a specific package."),
		),
		mcp.WithString("detail_level",
			mcp.Description("Verbosity: 'summary' (artifact existence + file counts), "+
				"'standard' (default — truncated content + file list), "+
				"'full' (complete artifact content + file list)."),
			mcp.Enum(memory.DetailLevelValues()...),
		),
	)
}

// Handle processes the sdd_audit tool call.
func (t *AuditTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	detailLevel := memory.ParseDetailLevel(req.GetString("detail_level", ""))
	scanPath := req.GetString("scan_path", "")

	// Resolve project root.
	root, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("finding project root: %w", err)
	}

	// Validate scan_path if provided.
	if scanPath != "" {
		candidate := filepath.Join(root, scanPath)
		info, err := os.Stat(candidate)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("scan_path '%s' not found: %v", scanPath, err)), nil
		}
		if !info.IsDir() {
			return mcp.NewToolResultError(fmt.Sprintf("scan_path '%s' is not a directory", scanPath)), nil
		}
	}

	start := time.Now()

	// Resolve docs directory — works even without hoofy.json.
	docsDir := config.DocsPath(root)

	// Read all spec artifacts.
	artifacts := readAuditArtifacts(docsDir)

	// Scan source files.
	sourceFiles := scanSourceFiles(root, docsDir, scanPath)

	duration := time.Since(start)

	// Build report.
	result := buildAuditReport(root, docsDir, artifacts, sourceFiles, detailLevel, duration)

	// Append token footer.
	tokens := memory.EstimateTokens(result)
	result += memory.TokenFooter(tokens)

	return mcp.NewToolResultText(result), nil
}
