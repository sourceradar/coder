package tools

import (
	"context"
	"fmt"
	"github.com/recrsn/coder/internal/common"
	"github.com/recrsn/coder/internal/llm"
	"github.com/recrsn/coder/internal/schema"
	"github.com/recrsn/coder/internal/ui"
	"strings"
)

// NewAgentTool creates a tool that launches an interactive agent with read-only tools
func NewAgentTool(registry *Registry, client *llm.Client, userInterface *ui.UI, modelName string, permissionManager *common.PermissionManager) *Tool {
	inputSchema := schema.Schema{
		Type: "object",
		Properties: map[string]schema.Property{
			"description": {
				Type:        "string",
				Description: "A short (3-5 word) description of the task",
			},
			"prompt": {
				Type:        "string",
				Description: "The task for the agent to perform",
			},
		},
		Required: []string{"description", "prompt"},
	}

	return &Tool{
		Name: "agent",
		Description: "Launch a new agent that can analyze code by using read-only tools." +
			" Agent cannot use any write tools or receive input from the user." +
			" It can only send a response back to the caller.",
		InputSchema: inputSchema,
		Explain: func(input map[string]any) ExplainResult {
			description, _ := input["description"].(string)
			return ExplainResult{
				Title:   fmt.Sprintf("Agent(%s)", description),
				Context: fmt.Sprintf("Launch an agent to perform task: %s", description),
			}
		},
		Execute: func(input map[string]any) (string, error) {
			prompt, _ := input["prompt"].(string)
			description, _ := input["description"].(string)

			// Create a filtered registry with only read-only tools
			readOnlyTools := []string{
				"read", "ls", "glob", "grep", "tree", "outline",
				"lsp_definition", "lsp_references", "lsp_callhierarchy",
			}

			// Create a filter registry with only read-only tools
			filteredRegistry := NewRegistry()
			for _, name := range readOnlyTools {
				tool, ok := registry.Get(name)
				if ok {
					filteredRegistry.Register(name, tool)
				}
			}

			// Create an output builder
			var outputBuilder strings.Builder
			outputBuilder.WriteString(fmt.Sprintf("## Agent Task: %s\n\n", description))

			// Build agent system prompt
			agentPrompt := `You are a code analysis agent. You can only use read-only tools to analyze code.
You CANNOT use any write tools or tools that modify the filesystem.
You must complete the task assigned to you and return a concise response.

Format your response in markdown. Include relevant code snippets and explanations.
Show your work by explaining how you arrived at your conclusions.`

			// Create an agent with filtered tools
			agent := llm.NewAgent(
				"CodeAnalysisAgent",
				agentPrompt,
				filteredRegistry.ListTools(),
				llm.ModelConfig{
					Model:       modelName,
					Temperature: 0.1, // Lower temperature for more precise analysis
				},
				client,
				// Tool call handler for the agent
				func(ctx context.Context, toolName string, args map[string]any) (string, error) {
					// Check for context cancellation
					select {
					case <-ctx.Done():
						return "", fmt.Errorf("tool execution interrupted")
					default:
						// Continue processing
					}

					// Get the tool from the registry
					tool, ok := filteredRegistry.Get(toolName)
					if !ok {
						errorMsg := fmt.Sprintf("Tool not found: %s", toolName)
						userInterface.PrintError(errorMsg)

						outputBuilder.WriteString(fmt.Sprintf("### Error: %s\n\n", errorMsg))
						return errorMsg, nil
					}

					var detail = tool.Explain(args)
					request := common.PermissionRequest{
						ToolName:  toolName,
						Arguments: args,
						Title:     detail.Title,
						Context:   detail.Context,
					}

					response := permissionManager.RequestPermission(request)
					execute := response.Granted
					alternate := response.AlternateAction

					outputBuilder.WriteString(fmt.Sprintf("### Tool Call: %s\n", toolName))
					outputBuilder.WriteString("```\n")
					for k, v := range args {
						outputBuilder.WriteString(fmt.Sprintf("%s: %v\n", k, v))
					}
					outputBuilder.WriteString("```\n\n")

					// If denied, return alternate instructions
					if !execute {
						errorMsg := "Permission denied by user"
						outputBuilder.WriteString(fmt.Sprintf("Error: %s\n\n", errorMsg))
						return fmt.Sprintf("Tool use denied by user. %s", alternate), nil
					}

					// Execute the tool
					result, err := tool.Run(args)

					// Display the result
					userInterface.PrintToolCall(toolName, args, result, err)

					outputBuilder.WriteString("### Tool Result\n")
					if err != nil {
						errorMsg := fmt.Sprintf("Error executing %s: %s", toolName, err.Error())
						outputBuilder.WriteString(fmt.Sprintf("Error: %s\n\n", errorMsg))
						return errorMsg, nil
					}

					// Truncate very long results
					resultOutput := result
					if len(result) > 2000 {
						resultOutput = result[:1997] + "..."
					}
					outputBuilder.WriteString("```\n")
					outputBuilder.WriteString(resultOutput)
					outputBuilder.WriteString("\n```\n\n")

					return result, nil
				},
				// Message handler for the agent
				func(message string) {
					userInterface.PrintAssistantMessage(message)

					outputBuilder.WriteString("### Agent Message\n")
					outputBuilder.WriteString(message)
					outputBuilder.WriteString("\n\n")
				},
			)

			// Add the user prompt
			agent.AddMessage("user", prompt)

			// Run the agent
			ctx := context.Background()
			finalMessage, err := agent.Run(ctx)
			if err != nil {
				return fmt.Sprintf("Error running agent: %s", err.Error()), nil
			}

			// Add the final summary to the output
			outputBuilder.WriteString("## Final Analysis\n\n")
			outputBuilder.WriteString(finalMessage.Content)

			return outputBuilder.String(), nil
		},
	}
}
