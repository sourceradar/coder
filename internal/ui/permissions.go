package ui

import (
	"fmt"
	"github.com/recrsn/coder/internal/common"
)

// UIPermissionHandler implements the PermissionHandler interface for the terminal UI
type UIPermissionHandler struct {
	ui *UI
}

// NewUIPermissionHandler creates a new UI permission handler
func NewUIPermissionHandler(ui *UI) *UIPermissionHandler {
	return &UIPermissionHandler{
		ui: ui,
	}
}

// RequestPermission displays a permission request to the user and returns their response
func (h *UIPermissionHandler) RequestPermission(request common.PermissionRequest) common.PermissionResponse {
	// Format the permission request with title and context
	var explanation string

	// Title and context
	explanation += fmt.Sprintf("Tool: %s\n\n", request.ToolName)
	explanation += fmt.Sprintf("%s\n\n", request.Title)
	explanation += request.Context + "\n\n"

	// Arguments
	explanation += "Arguments:\n"
	for k, v := range request.Arguments {
		explanation += fmt.Sprintf("  %s: %v\n", k, v)
	}

	// Ask for user confirmation
	granted, alternate := h.ui.AskPermission(explanation)

	return common.PermissionResponse{
		Granted:         granted,
		AlternateAction: alternate,
	}
}
