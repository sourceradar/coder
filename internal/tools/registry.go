package tools

import "github.com/recrsn/coder/internal/llm"

// Registry holds all available tools
type Registry struct {
	tools map[string]*Tool
}

// NewRegistry creates a new registry with all tools
func NewRegistry() *Registry {
	registry := &Registry{
		tools: make(map[string]*Tool),
	}

	return registry
}

// Register adds a tool to the registry
func (r *Registry) Register(name string, tool *Tool) {
	r.tools[name] = tool
}

// Get retrieves a tool from the registry
func (r *Registry) Get(name string) (*Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// ListTools returns a list of all available tool names
func (r *Registry) ListTools() []llm.Tool {
	var tools []llm.Tool
	for name, tool := range r.tools {
		tools = append(tools, llm.Tool{
			Type: "function",
			Function: llm.FunctionDefinition{
				Name:        name,
				Parameters:  tool.InputSchema,
				Description: tool.Description,
			},
		})
	}
	return tools
}

func (r *Registry) GetAll() []*Tool {
	var tools []*Tool
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}
