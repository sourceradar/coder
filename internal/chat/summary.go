package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/recrsn/coder/internal/llm"
)

// MessageSummary represents a summarized version of the chat history
type MessageSummary struct {
	Summary      string
	LastUpdateAt time.Time
}

// AddSummaryToContext adds the conversation summary to the context for future messages
func (s *Session) AddSummaryToContext() error {
	// Generate a summary of the conversation
	summary, err := s.SummarizeMessages()
	if err != nil {
		return err
	}

	// If no summary (not enough messages), just return
	if summary == "" {
		return nil
	}

	// Check if we already have a summary message in our context
	// It would be the second message if it exists (after the system prompt)
	hasSummaryMessage := len(s.messages) > 1 && s.messages[1].Role == "system" &&
		strings.Contains(s.messages[1].Content, "Here is a summary of the conversation so far:")

	// Create or update the summary message
	summaryMessage := llm.Message{
		Role:    "system",
		Content: "Here is a summary of the conversation so far:\n\n" + summary + "\n\nPlease use this as context for your responses.",
	}

	if hasSummaryMessage {
		// Replace the existing summary message
		s.messages[1] = summaryMessage
	} else {
		// Insert the summary as the second message, right after the system prompt
		if len(s.messages) > 1 {
			// Make room for the summary message
			newMessages := make([]llm.Message, 0, len(s.messages)+1)
			newMessages = append(newMessages, s.messages[0])     // System prompt
			newMessages = append(newMessages, summaryMessage)    // Summary
			newMessages = append(newMessages, s.messages[1:]...) // Rest of messages
			s.messages = newMessages
		}
	}

	// Store the summary in the session
	s.conversationSummary = summary

	return nil
}

// SummarizeMessages generates a summary of previous messages in the conversation
// focusing on extracting key points and condensing them into a concise summary.
func (s *Session) SummarizeMessages() (string, error) {
	// If there are not enough messages to summarize, return early
	if len(s.messages) <= 1 { // Skip just the system message
		return "", nil
	}

	// Create a context that can be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Prepare messages for summarization
	// We'll use a special system message to instruct the LLM to create a summary
	summaryPrompt := []llm.Message{
		{
			Role: "system",
			Content: "Your task is to create a concise summary of the conversation provided below. " +
				"Focus only on the key points, technical details, and important decisions made. " +
				"The summary should be factual and neutral, highlighting the main topics discussed, " +
				"code concepts explained, and solutions proposed. " +
				"Keep the summary under 300 words. Ignore any casual conversation or pleasantries.",
		},
	}

	// Add a selection of the most recent messages
	// We'll limit this to avoid hitting token limits
	// Take at most the last 10 message exchanges (up to 20 messages)
	startIdx := max(1, len(s.messages)-20) // Skip the first system message
	for _, msg := range s.messages[startIdx:] {
		// Skip tool messages as they often contain verbose output
		if msg.Role != "tool" {
			summaryPrompt = append(summaryPrompt, msg)
		}
	}

	// Add a final user message asking for the summary
	summaryPrompt = append(summaryPrompt, llm.Message{
		Role:    "user",
		Content: "Please summarize our conversation so far into a concise, factual summary focusing on key technical points.",
	})

	// Create chat completion request
	req := llm.ChatCompletionRequest{
		Model:       s.config.Provider.Model,
		Messages:    summaryPrompt,
		Temperature: 0.3, // Use lower temperature for more factual summary
	}

	// Send request to LLM
	spinner := s.ui.StartSpinner("Generating conversation summary...")
	resp, err := s.client.CreateChatCompletionWithContext(ctx, req)
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

	return summary, nil
}
