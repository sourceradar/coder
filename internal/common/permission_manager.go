package common

import (
	"github.com/recrsn/coder/internal/config"
)

// PermissionManager handles permission requests based on configuration
type PermissionManager struct {
	config        config.PermissionConfig
	handler       PermissionHandler
	defaultPolicy bool // Default policy if no specific rule exists
}

// NewPermissionManager creates a new permission manager
func NewPermissionManager(config config.PermissionConfig, handler PermissionHandler) *PermissionManager {
	return &PermissionManager{
		config:        config,
		handler:       handler,
		defaultPolicy: false, // Default to requiring permission
	}
}

// RequestPermission handles a permission request
func (m *PermissionManager) RequestPermission(request PermissionRequest) PermissionResponse {
	// Check if this tool is auto-approved
	if autoApprove, ok := m.config.AutoApprove[request.ToolName]; ok && autoApprove {
		return PermissionResponse{
			Granted:         true,
			AlternateAction: "",
		}
	}

	// If not auto-approved and we have a UI handler, ask the user
	if m.handler != nil {
		return m.handler.RequestPermission(request)
	}

	// If no UI handler, use the default policy
	return PermissionResponse{
		Granted:         m.defaultPolicy,
		AlternateAction: "Permission denied by default policy",
	}
}
