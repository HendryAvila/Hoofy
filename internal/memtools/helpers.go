// Package memtools provides MCP tool handlers for the persistent memory system.
//
// Each tool handler follows the same pattern as internal/tools:
// - A struct with dependencies (memory.Store) injected via constructor
// - Definition() returns the mcp.Tool schema
// - Handle() processes the request and returns a result
//
// Tools are storage tools: they receive AI-generated content and persist it.
package memtools

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// intArg extracts an integer argument from a tool request, returning
// defaultVal if the key is missing or not a number (JSON numbers are float64).
func intArg(req mcp.CallToolRequest, key string, defaultVal int) int {
	v, ok := req.GetArguments()[key].(float64)
	if !ok {
		return defaultVal
	}
	return int(v)
}

// boolArg extracts a boolean argument from a tool request.
func boolArg(req mcp.CallToolRequest, key string, defaultVal bool) bool {
	v, ok := req.GetArguments()[key].(bool)
	if !ok {
		return defaultVal
	}
	return v
}
