package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultTimeout = 30 * time.Second
)

// Client provides a simple client for interacting with OpenAI API
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     APILogger
}

// NewClient creates a new OpenAI API client
func NewClient(baseURL, apiKey string, logger APILogger) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		logger: logger,
	}
}

// Message represents a message in a conversation
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// FunctionCall represents a function call by the model
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// FunctionDefinition defines a function that can be called by the model
type FunctionDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

// Tool represents a tool available to the model
type Tool struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

// ChatCompletionRequest is the request structure for chat completions
type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// ToolCall represents a tool call by the model
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// ChatCompletionChoice represents a completion choice
type ChatCompletionChoice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// ChatCompletionResponse is the response structure for chat completions
type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Choices []ChatCompletionChoice `json:"choices"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// CreateChatCompletion creates a chat completion with context for cancellation
func (c *Client) CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	// Create the request with the provided context
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		// Check if the error was caused by context cancellation
		if ctx.Err() != nil {
			if c.logger != nil {
				c.logger.LogInteraction(req, nil, fmt.Errorf("request cancelled: %w", ctx.Err()))
			}
			return nil, fmt.Errorf("request cancelled: %w", ctx.Err())
		}
		err := fmt.Errorf("request error: %w", err)
		if c.logger != nil {
			c.logger.LogInteraction(req, nil, err)
		}
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		err := fmt.Errorf("reading response: %w", err)
		if c.logger != nil {
			c.logger.LogInteraction(req, nil, err)
		}
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
			if c.logger != nil {
				c.logger.LogInteraction(req, nil, fmt.Errorf("API error: %s", errResp.Error.Message))
			}
			return nil, fmt.Errorf("API error: %s", errResp.Error.Message)
		}
		if c.logger != nil {
			c.logger.LogInteraction(req, nil, fmt.Errorf("API error: %s", string(body)))
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var respData ChatCompletionResponse
	if err := json.Unmarshal(body, &respData); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	// Log the interaction
	if c.logger != nil {
		c.logger.LogInteraction(req, respData, nil)
	}

	return &respData, nil
}
