package chat

import (
	"context"
	"fmt"
	"github.com/recrsn/coder/internal/chat/prompts"
	"time"

	"github.com/recrsn/coder/internal/llm"
)

// MessageSummary represents a summarized version of the chat history
type MessageSummary struct {
	Summary      string
	LastUpdateAt time.Time
}

// SummarizeMessages generates a summary of previous messages in the conversation
// focusing on extracting key points and condensing them into a concise summary.
func (s *Session) SummarizeMessages() (string, error) {
	// If there are not enough messages to summarize, return early
	if len(s.agent.Messages) <= 1 { // Skip just the system message
		return "", nil
	}

	// Create a context that can be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Prepare messages for summarization
	// We'll use a special system message to instruct the LLM to create a summary
	summaryPrompt := []llm.Message{
		{
			Role:    "system",
			Content: prompts.RenderSummaryPrompt(),
		},
	}

	for _, msg := range s.agent.Messages {
		if msg.Role == "system" {
			summaryPrompt = append(summaryPrompt, msg)
		}
	}

	// Add a final user message asking for the summary
	summaryPrompt = append(summaryPrompt, llm.Message{
		Role: "user",
		Content: "Please summarize our conversation so far into a concise, " +
			"factual summary",
	})

	cfg := selectModel("summary", s.config)
	// Create chat completion request
	req := llm.ChatCompletionRequest{
		Model:       cfg.Model,
		Messages:    summaryPrompt,
		Temperature: cfg.Temperature,
	}

	// Send request to LLM
	spinner := s.ui.StartSpinner("Generating conversation summary...")
	resp, err := s.client.CreateChatCompletion(ctx, req)
	if err != nil {
		s.ui.StopSpinnerFail(spinner, "Failed to generate summary")
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	// Check if we got a valid response
	if len(resp.Choices) == 0 {
		s.ui.StopSpinnerFail(spinner, "No summary generated")
		return "", fmt.Errorf("no summary generated")
	}

	// Extract the summary
	summary := resp.Choices[0].Message.Content
	s.ui.StopSpinner(spinner, "Summary generated")

	s.agent.ClearContext()
	s.agent.AddMessage("user", fmt.Sprintf(""+
		"This session is being continued from a previous conversation."+
		" The conversation is summarized below:"+
		"\n%s\n"+
		"Please continue the conversation from where we left it off without asking the any further questions."+
		"Continue with the last task that you were asked to work on.",
		summary,
	))

	return summary, nil
}
