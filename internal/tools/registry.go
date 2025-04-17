package tools

// Registry holds all available tools
type Registry struct {
	tools map[string]interface{}
}

// NewRegistry creates a new registry with all tools
func NewRegistry() *Registry {
	registry := &Registry{
		tools: make(map[string]interface{}),
	}

	return registry
}

// Register adds a tool to the registry
func (r *Registry) Register(name string, tool interface{}) {
	r.tools[name] = tool
}

// Get retrieves a tool from the registry
func (r *Registry) Get(name string) (interface{}, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// ListTools returns a list of all available tool names
func (r *Registry) ListTools() []string {
	var names []string
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}
