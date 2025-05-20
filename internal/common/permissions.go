package common

// PermissionRequest represents a request for permission to perform an action
type PermissionRequest struct {
	// ToolName is the name of the tool requesting permission
	ToolName string

	// Arguments contains the arguments passed to the tool
	Arguments map[string]any

	// Title is a short description of the permission being requested
	Title string

	// Context provides detailed information about the permission request
	Context string
}

// PermissionResponse represents the response to a permission request
type PermissionResponse struct {
	// Granted indicates whether permission was granted
	Granted bool

	// AlternateAction contains alternative instructions if permission was denied
	AlternateAction string
}

// PermissionHandler defines the interface for handling permission requests
type PermissionHandler interface {
	// RequestPermission requests permission for an action
	// Returns whether permission was granted and any alternate action
	RequestPermission(request PermissionRequest) PermissionResponse
}
