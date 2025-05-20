package common

import (
	"github.com/recrsn/coder/internal/config"
	"testing"
)

// mockPermissionHandler implements PermissionHandler for testing
type mockPermissionHandler struct {
	response PermissionResponse
}

func (m *mockPermissionHandler) RequestPermission(request PermissionRequest) PermissionResponse {
	return m.response
}

func TestPermissionManager_RequestPermission(t *testing.T) {
	// Test cases
	tests := []struct {
		name           string
		config         config.PermissionConfig
		handler        PermissionHandler
		request        PermissionRequest
		expectedResult bool
	}{
		{
			name: "Auto-approved tool",
			config: config.PermissionConfig{
				AutoApprove: map[string]bool{"read": true},
			},
			handler: nil, // Not needed for auto-approved tools
			request: PermissionRequest{
				ToolName:  "read",
				Arguments: map[string]any{"file_path": "/tmp/test.txt"},
				Title:     "Read file",
				Context:   "Reading file /tmp/test.txt",
			},
			expectedResult: true,
		},
		{
			name: "Non-auto-approved tool with approval",
			config: config.PermissionConfig{
				AutoApprove: map[string]bool{"read": true},
			},
			handler: &mockPermissionHandler{
				response: PermissionResponse{
					Granted:         true,
					AlternateAction: "",
				},
			},
			request: PermissionRequest{
				ToolName:  "write",
				Arguments: map[string]any{"file_path": "/tmp/test.txt", "content": "test"},
				Title:     "Write file",
				Context:   "Writing to file /tmp/test.txt",
			},
			expectedResult: true,
		},
		{
			name: "Non-auto-approved tool with denial",
			config: config.PermissionConfig{
				AutoApprove: map[string]bool{"read": true},
			},
			handler: &mockPermissionHandler{
				response: PermissionResponse{
					Granted:         false,
					AlternateAction: "Do something else",
				},
			},
			request: PermissionRequest{
				ToolName:  "write",
				Arguments: map[string]any{"file_path": "/tmp/test.txt", "content": "test"},
				Title:     "Write file",
				Context:   "Writing to file /tmp/test.txt",
			},
			expectedResult: false,
		},
		{
			name: "Default policy with no handler",
			config: config.PermissionConfig{
				AutoApprove: map[string]bool{"read": true},
			},
			handler: nil,
			request: PermissionRequest{
				ToolName:  "write",
				Arguments: map[string]any{"file_path": "/tmp/test.txt", "content": "test"},
				Title:     "Write file",
				Context:   "Writing to file /tmp/test.txt",
			},
			expectedResult: false, // Default policy is to deny
		},
	}

	// Run test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			manager := NewPermissionManager(tc.config, tc.handler)
			result := manager.RequestPermission(tc.request)

			if result.Granted != tc.expectedResult {
				t.Errorf("Expected permission granted to be %v, got %v",
					tc.expectedResult, result.Granted)
			}
		})
	}
}
