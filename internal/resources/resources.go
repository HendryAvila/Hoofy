// Package resources implements MCP resource handlers for the SDD pipeline.
//
// Resources provide read-only data that the host can consume for context.
// They use URI-based addressing (sdd://...) following MCP conventions.
package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/HendryAvila/sdd-hoffy/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
)

// Handler manages SDD resource endpoints.
type Handler struct {
	store config.Store
}

// NewHandler creates a resource Handler with its dependencies.
func NewHandler(store config.Store) *Handler {
	return &Handler{store: store}
}

// StatusResource returns the MCP resource definition for project status.
func (h *Handler) StatusResource() mcp.Resource {
	return mcp.NewResource(
		"sdd://project/status",
		"SDD Project Status",
		mcp.WithResourceDescription("Current SDD pipeline status, stage, and clarity score"),
		mcp.WithMIMEType("application/json"),
	)
}

// HandleStatus returns the current project status as JSON.
func (h *Handler) HandleStatus(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	projectRoot, err := findResourceRoot()
	if err != nil {
		return nil, fmt.Errorf("finding project root: %w", err)
	}

	cfg, err := h.store.Load(projectRoot)
	if err != nil {
		return errorResource(req.Params.URI, err.Error()), nil
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling status: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

// errorResource returns a resource with an error message.
func errorResource(uri, message string) []mcp.ResourceContents {
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      uri,
			MIMEType: "text/plain",
			Text:     fmt.Sprintf("Error: %s", message),
		},
	}
}

// findResourceRoot is a simplified version of project root detection
// for resource handlers.
func findResourceRoot() (string, error) {
	// Resources reuse the same logic as tools.
	// In a more complex setup, this could be injected.
	return findRoot()
}
