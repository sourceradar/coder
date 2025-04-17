package config

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		Provider: ProviderConfig{
			Endpoint: "https://api.openai.com/v1/chat/completions",
			APIKey:   "",
			Model:    "gpt-4o",
		},
		UI: UIConfig{
			ColorEnabled: true,
			ShowSpinner:  true,
		},
		Prompt: PromptConfig{
			TemplateFile: "default.go.tmpl",
		},
	}
}
