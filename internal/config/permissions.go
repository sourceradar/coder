package config

// PermissionConfig holds configuration for permission handling
type PermissionConfig struct {
	// AutoApprove automatically approves certain types of tools without asking
	AutoApprove map[string]bool `mapstructure:"auto_approve"`
}

// DefaultPermissionConfig returns the default permission configuration
func DefaultPermissionConfig() PermissionConfig {
	return PermissionConfig{
		AutoApprove: map[string]bool{
			// Safe tools that don't need confirmation
			"read":          true,
			"ls":            true,
			"glob":          true,
			"grep":          true,
			"tree":          true,
			"outline":       true,
			"definition":    true,
			"references":    true,
			"agent":         true,
			"callHierarchy": true,
		},
	}
}
