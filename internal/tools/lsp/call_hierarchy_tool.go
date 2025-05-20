package lsp

import (
	"fmt"
	"github.com/recrsn/coder/internal/lsp"
	"path/filepath"
	"strings"

	"github.com/recrsn/coder/internal/schema"
	"github.com/recrsn/coder/internal/tools"
	lsp2 "go.bug.st/lsp"
)

// NewCallHierarchyTool creates a tool for exploring call hierarchies using LSP
func NewCallHierarchyTool(manager *lsp.Manager) *tools.Tool {
	return &tools.Tool{
		Name:        "callhierarchy",
		Description: "Explore function call hierarchies using the Language Server Protocol",
		InputSchema: schema.Schema{
			Type: "object",
			Properties: map[string]schema.Property{
				"file_path": {
					Type:        "string",
					Description: "The path to the file containing the function or method",
				},
				"line": {
					Type:        "integer",
					Description: "The line number of the function or method (0-based)",
				},
				"character": {
					Type:        "integer",
					Description: "The character offset of the function or method (0-based)",
				},
				"direction": {
					Type:        "string",
					Description: "The direction of call hierarchy to explore (incoming or outgoing)",
				},
			},
			Required: []string{"file_path", "line", "character", "direction"},
		},
		Explain: func(input map[string]any) tools.ExplainResult {
			filePath, _ := input["file_path"].(string)
			line, _ := input["line"].(float64)
			character, _ := input["character"].(float64)
			direction, _ := input["direction"].(string)

			title := fmt.Sprintf("CallHierarchy(%s, %s)", filePath, direction)
			content := fmt.Sprintf("Will explore %s calls for the function or method at %s:%d:%d",
				direction, filePath, int(line), int(character))

			return tools.ExplainResult{
				Title:   title,
				Context: content,
			}
		},
		Execute: func(input map[string]any) (string, error) {
			// Extract parameters
			filePath, _ := input["file_path"].(string)
			line, _ := input["line"].(float64)
			character, _ := input["character"].(float64)
			direction, _ := input["direction"].(string)

			// Convert to absolute path if needed
			absPath, err := filepath.Abs(filePath)
			if err != nil {
				return "", fmt.Errorf("failed to resolve absolute path: %w", err)
			}

			// First prepare call hierarchy
			items, err := manager.PrepareCallHierarchy(absPath, int(line), int(character))
			if err != nil {
				return "", err
			}

			if len(items) == 0 {
				return "No call hierarchy information available for this position", nil
			}

			// Use the first item (most relevant)
			item := items[0]

			var result strings.Builder

			// Show the selected item
			result.WriteString(fmt.Sprintf("Function: %s\n", item.Name))
			if item.Detail != "" {
				result.WriteString(fmt.Sprintf("Details: %s\n", item.Detail))
			}
			result.WriteString(fmt.Sprintf("Location: %s:%d:%d\n\n",
				item.URI.AsPath().String(),
				item.Range.Start.Line+1,
				item.Range.Start.Character+1))

			// Get calls based on direction
			switch direction {
			case "incoming":
				return handleIncomingCalls(manager, item, &result)
			case "outgoing":
				return handleOutgoingCalls(manager, item, &result)
			default:
				return "", fmt.Errorf("invalid direction: %s (must be 'incoming' or 'outgoing')", direction)
			}
		},
	}
}

// handleIncomingCalls processes and formats incoming calls
func handleIncomingCalls(manager *lsp.Manager, item lsp2.CallHierarchyItem, result *strings.Builder) (string, error) {
	incomingCalls, err := manager.GetIncomingCalls(item)
	if err != nil {
		return "", err
	}

	if len(incomingCalls) == 0 {
		result.WriteString("No incoming calls found\n")
		return result.String(), nil
	}

	result.WriteString(fmt.Sprintf("Found %d incoming calls:\n\n", len(incomingCalls)))

	// Group calls by caller
	for _, call := range incomingCalls {
		caller := call.From
		result.WriteString(fmt.Sprintf("From: %s\n", caller.Name))
		if caller.Detail != "" {
			result.WriteString(fmt.Sprintf("Details: %s\n", caller.Detail))
		}
		result.WriteString(fmt.Sprintf("Location: %s:%d:%d\n",
			caller.URI.AsPath().String(),
			caller.Range.Start.Line+1,
			caller.Range.Start.Character+1))

		// Show call sites
		if len(call.FromRanges) > 0 {
			result.WriteString("Call sites:\n")
			for _, callRange := range call.FromRanges {
				result.WriteString(fmt.Sprintf("  - Line %d, Col %d\n",
					callRange.Start.Line+1,
					callRange.Start.Character+1))
			}
		}
		result.WriteString("\n")
	}

	return result.String(), nil
}

// handleOutgoingCalls processes and formats outgoing calls
func handleOutgoingCalls(manager *lsp.Manager, item lsp2.CallHierarchyItem, result *strings.Builder) (string, error) {
	outgoingCalls, err := manager.GetOutgoingCalls(item)
	if err != nil {
		return "", err
	}

	if len(outgoingCalls) == 0 {
		result.WriteString("No outgoing calls found\n")
		return result.String(), nil
	}

	result.WriteString(fmt.Sprintf("Found %d outgoing calls:\n\n", len(outgoingCalls)))

	// Group calls by callee
	for _, call := range outgoingCalls {
		callee := call.Ro
		result.WriteString(fmt.Sprintf("To: %s\n", callee.Name))
		if callee.Detail != "" {
			result.WriteString(fmt.Sprintf("Details: %s\n", callee.Detail))
		}
		result.WriteString(fmt.Sprintf("Location: %s:%d:%d\n",
			callee.URI.AsPath().String(),
			callee.Range.Start.Line+1,
			callee.Range.Start.Character+1))

		// Show call sites
		if len(call.FromRanges) > 0 {
			result.WriteString("Call sites:\n")
			for _, callRange := range call.FromRanges {
				result.WriteString(fmt.Sprintf("  - Line %d, Col %d\n",
					callRange.Start.Line+1,
					callRange.Start.Character+1))
			}
		}
		result.WriteString("\n")
	}

	return result.String(), nil
}
