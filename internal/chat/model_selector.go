package chat

import (
	"github.com/recrsn/coder/internal/config"
	"github.com/recrsn/coder/internal/llm"
)

func selectModel(usage string, cfg config.Config) llm.ModelConfig {
	defaultCfg := llm.ModelConfig{
		Model:       cfg.Provider.Model,
		Temperature: 0.6,
	}

	switch usage {
	case "summary":
		if cfg.Provider.LiteModel != "" {
			return llm.ModelConfig{
				Model:       cfg.Provider.LiteModel,
				Temperature: 0.3,
			}
		}
		return defaultCfg
	case "chat":
		return defaultCfg
	default:
		return defaultCfg
	}
}
