package llm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// APILogger logs API requests and responses to a file
type APILogger interface {
	LogInteraction(req interface{}, resp interface{}, err error)
}

type FileLogger struct {
	logFilePath string
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp string      `json:"timestamp"`
	Request   interface{} `json:"request,omitempty"`
	Response  interface{} `json:"response,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// NewAPILogger creates a new APILogger instance
func NewAPILogger(configDir string) APILogger {
	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("Warning: couldn't create config directory: %v\n", err)
	}

	return &FileLogger{
		logFilePath: filepath.Join(configDir, "api_logs.jsonl"),
	}
}

// LogInteraction logs an API request/response pair
func (l *FileLogger) LogInteraction(req interface{}, resp interface{}, err error) {
	// Create log entry
	logEntry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Request:   req,
	}

	if err != nil {
		logEntry.Error = err.Error()
	} else if resp != nil {
		logEntry.Response = resp
	}

	// Marshal to JSON
	logJSON, jsonErr := json.Marshal(logEntry)
	if jsonErr != nil {
		fmt.Printf("Warning: couldn't marshal log entry to JSON: %v\n", jsonErr)
		return
	}

	// Append to log file
	file, fileErr := os.OpenFile(l.logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if fileErr != nil {
		fmt.Printf("Warning: couldn't open log file: %v\n", fileErr)
		return
	}
	defer file.Close()

	if _, writeErr := file.Write(append(logJSON, '\n')); writeErr != nil {
		fmt.Printf("Warning: couldn't write to log file: %v\n", writeErr)
	}
}
