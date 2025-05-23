package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	md "github.com/MichaelMure/go-term-markdown"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pterm/pterm"
	"github.com/recrsn/coder/internal/config"
)

var (
	// Styles
	appStyle = lipgloss.NewStyle().
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#7D56F4")).
			Bold(true).
			Padding(0, 1)

	userMsgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#32CD32")).
			Bold(true)

	assistantMsgStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#5D5DFF"))

	spinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#5D5DFF"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#32CD32")).
			Bold(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			MarginTop(1)
)

// BubbleTeaUI is a TUI implementation using Bubble Tea
type BubbleTeaUI struct {
	config      config.UIConfig
	exitHandler func()
	program     *tea.Program
	model       *model
}

// Message is a chat message
type Message struct {
	Content    string
	IsUser     bool
	IsMarkdown bool
}

// Tool call result
type ToolCall struct {
	Name    string
	Args    map[string]any
	Result  string
	HasErr  bool
	ErrText string
}

// model represents the application state
type model struct {
	messages      []Message
	toolCalls     []ToolCall
	viewport      viewport.Model
	textarea      textarea.Model
	spinner       spinner.Model
	help          help.Model
	err           error
	inPrompt      bool // Whether we're showing a permission/confirmation prompt
	promptText    string
	viewportReady bool
	spinnerActive bool
	activeSpinner string
	width         int
	height        int
	userInput     chan string      // Channel to receive user input
	spinnerDone   chan string      // Channel to signal spinner completion
	renderCh      chan struct{}    // Channel to trigger re-render
	promptCh      chan promptEvent // Channel for prompt events
}

type promptEvent struct {
	confirmed bool
	text      string
}

// NewBubbleTeaUI creates a new Bubble Tea UI instance
func NewBubbleTeaUI(cfg config.UIConfig, exitHandler func()) (*BubbleTeaUI, error) {
	m := newModel()
	p := tea.NewProgram(m, tea.WithAltScreen())

	go func() {
		if err := p.Start(); err != nil {
			fmt.Println("Error running program:", err)
			os.Exit(1)
		}
	}()

	bui := &BubbleTeaUI{
		config:      cfg,
		exitHandler: exitHandler,
		program:     p,
		model:       m,
	}

	// Send initial render trigger
	go func() {
		time.Sleep(100 * time.Millisecond) // Give program time to start
		m.renderCh <- struct{}{}
	}()

	return bui, nil
}

func newModel() *model {
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Focus()
	ta.CharLimit = 0
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	s := spinner.New()
	s.Style = spinnerStyle
	s.Spinner = spinner.Dot

	vp := viewport.New(0, 0)
	// Set simple key mappings
	vp.KeyMap = viewport.KeyMap{}

	h := help.New()
	h.ShowAll = false

	return &model{
		messages:    make([]Message, 0),
		toolCalls:   make([]ToolCall, 0),
		textarea:    ta,
		viewport:    vp,
		spinner:     s,
		help:        h,
		userInput:   make(chan string),
		spinnerDone: make(chan string),
		renderCh:    make(chan struct{}),
		promptCh:    make(chan promptEvent),
	}
}

// ShowHeader displays the application header
func (ui *BubbleTeaUI) ShowHeader() {
	ui.model.messages = append(ui.model.messages, Message{
		Content:    "Coder - Your Programming Sidekick",
		IsUser:     false,
		IsMarkdown: false,
	})

	// Show info about API logging
	configDir, _ := getConfigDir()
	if configDir != "" {
		ui.model.messages = append(ui.model.messages, Message{
			Content:    "API requests and responses are being logged to the logs directory: " + configDir + "/logs/",
			IsUser:     false,
			IsMarkdown: false,
		})
	}
	ui.triggerRender()
}

// StartSpinner starts a spinner with the given text
func (ui *BubbleTeaUI) StartSpinner(text string) *pterm.SpinnerPrinter {
	if !ui.config.ShowSpinner {
		return nil
	}

	ui.model.spinnerActive = true
	ui.model.activeSpinner = text
	ui.triggerRender()

	// Return nil because we don't use the pterm spinner
	return nil
}

// StopSpinner stops a spinner with success
func (ui *BubbleTeaUI) StopSpinner(spinner *pterm.SpinnerPrinter, text string) {
	ui.model.spinnerActive = false
	ui.model.activeSpinner = ""
	ui.model.spinnerDone <- text
	ui.triggerRender()
}

// StopSpinnerFail stops a spinner with failure
func (ui *BubbleTeaUI) StopSpinnerFail(spinner *pterm.SpinnerPrinter, text string) {
	ui.model.spinnerActive = false
	ui.model.activeSpinner = ""
	ui.model.err = fmt.Errorf(text)
	ui.model.spinnerDone <- text
	ui.triggerRender()
}

// PrintUserMessage prints a user message
func (ui *BubbleTeaUI) PrintUserMessage(message string) {
	ui.model.messages = append(ui.model.messages, Message{
		Content: message,
		IsUser:  true,
	})
	ui.triggerRender()
}

// PrintAssistantMessage prints an assistant message with markdown formatting
func (ui *BubbleTeaUI) PrintAssistantMessage(message string) {
	ui.model.messages = append(ui.model.messages, Message{
		Content:    message,
		IsUser:     false,
		IsMarkdown: true,
	})
	ui.triggerRender()
}

// PrintCodeBlock prints a code block with a highlighted box
func (ui *BubbleTeaUI) PrintCodeBlock(code, language string) {
	content := fmt.Sprintf("```%s\n%s\n```", language, code)
	ui.model.messages = append(ui.model.messages, Message{
		Content:    content,
		IsUser:     false,
		IsMarkdown: true,
	})
	ui.triggerRender()
}

// PrintToolCall prints information about a tool call
func (ui *BubbleTeaUI) PrintToolCall(toolName string, args map[string]any, result string, err error) {
	toolCall := ToolCall{
		Name:   toolName,
		Args:   args,
		Result: result,
	}

	if err != nil {
		toolCall.HasErr = true
		toolCall.ErrText = err.Error()
	}

	ui.model.toolCalls = append(ui.model.toolCalls, toolCall)
	ui.triggerRender()
}

// PrintHelp prints the help message
func (ui *BubbleTeaUI) PrintHelp() {
	helpText := `
Commands:
/help     - Show this help message
/exit     - Exit the application
/clear    - Clear the screen
/config   - Show or edit configuration
/tools    - List available tools
/prompt   - Edit the prompt template
/version  - Show version information
Ctrl+C    - Interrupt current operation
Ctrl+D    - Exit the application
`

	ui.model.messages = append(ui.model.messages, Message{
		Content:    helpText,
		IsUser:     false,
		IsMarkdown: true,
	})
	ui.triggerRender()
}

// PrintError prints an error message
func (ui *BubbleTeaUI) PrintError(message string) {
	ui.model.err = fmt.Errorf(message)
	ui.triggerRender()
}

// PrintSuccess prints a success message
func (ui *BubbleTeaUI) PrintSuccess(message string) {
	ui.model.messages = append(ui.model.messages, Message{
		Content:    "[SUCCESS] " + message,
		IsUser:     false,
		IsMarkdown: false,
	})
	ui.triggerRender()
}

// PrintInfo prints an informational message
func (ui *BubbleTeaUI) PrintInfo(message string) {
	ui.model.messages = append(ui.model.messages, Message{
		Content:    "[INFO] " + message,
		IsUser:     false,
		IsMarkdown: false,
	})
	ui.triggerRender()
}

// AskInput asks for user input with a prompt
func (ui *BubbleTeaUI) AskInput(prompt string) string {
	// Set prompt in the model
	ui.model.textarea.Placeholder = prompt
	ui.triggerRender()

	// Wait for input from channel
	text := <-ui.model.userInput

	if text == "/exit" && ui.exitHandler != nil {
		ui.exitHandler()
		return "/exit"
	}

	return text
}

// AskMultiLineInput asks for multi-line user input with a prompt
func (ui *BubbleTeaUI) AskMultiLineInput(prompt string) string {
	// Set prompt in the model
	ui.model.textarea.Placeholder = prompt
	ui.triggerRender()

	// Wait for input from channel
	return <-ui.model.userInput
}

// ClearScreen clears the terminal screen
func (ui *BubbleTeaUI) ClearScreen() {
	ui.model.messages = make([]Message, 0)
	ui.model.toolCalls = make([]ToolCall, 0)
	ui.triggerRender()
}

// AskToolCallConfirmation asks for confirmation for a tool call
func (ui *BubbleTeaUI) AskToolCallConfirmation(explanation string) (bool, string) {
	ui.model.inPrompt = true
	ui.model.promptText = "Confirm tool call: " + explanation
	ui.triggerRender()

	// Wait for response from prompt channel
	resp := <-ui.model.promptCh
	ui.model.inPrompt = false
	ui.triggerRender()

	return resp.confirmed, resp.text
}

// AskPermission asks the user for permission with a title and context
func (ui *BubbleTeaUI) AskPermission(explanation string) (bool, string) {
	ui.model.inPrompt = true
	ui.model.promptText = "Permission Request: " + explanation
	ui.triggerRender()

	// Wait for response from prompt channel
	resp := <-ui.model.promptCh
	ui.model.inPrompt = false
	ui.triggerRender()

	return resp.confirmed, resp.text
}

// Helper method to trigger a re-render
func (ui *BubbleTeaUI) triggerRender() {
	// Non-blocking send to trigger render
	select {
	case ui.model.renderCh <- struct{}{}:
		// Message sent
	default:
		// Channel full, ignore
	}
}

// Implement tea.Model interface for our model

func (m *model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.spinner.Tick,
		m.listenForRenderTriggers,
	)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		spCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+d":
			return m, tea.Quit
		case "enter":
			if m.inPrompt {
				m.promptCh <- promptEvent{confirmed: true, text: ""}
				return m, nil
			}
			if !m.textarea.Focused() {
				m.textarea.Focus()
				return m, nil
			}
			// Handle input submission
			input := strings.TrimSpace(m.textarea.Value())
			if input != "" {
				m.textarea.Reset()
				// Send input to channel non-blocking
				select {
				case m.userInput <- input:
					// Sent successfully
				default:
					// Channel full, ignore
				}
			}
		case "esc":
			if m.inPrompt {
				m.promptCh <- promptEvent{confirmed: false, text: ""}
				return m, nil
			}
			if m.textarea.Focused() {
				m.textarea.Blur()
			} else {
				m.textarea.Focus()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update viewport and text area based on window size
		headerHeight := 1
		footerHeight := 6 // Help + text area + padding
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - headerHeight - footerHeight

		m.textarea.SetWidth(msg.Width - 4)

		// Mark viewport as ready
		if !m.viewportReady {
			m.viewportReady = true
			m.updateViewportContent()
		}

	case spinner.TickMsg:
		// Update spinner
		if m.spinnerActive {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, m.spinner.Tick

	// Custom event for re-rendering
	case renderEvent:
		m.updateViewportContent()
		return m, m.listenForRenderTriggers

	case spinnerEvent:
		m.updateViewportContent()
		return m, m.listenForRenderTriggers
	}

	// Update components
	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	m.spinner, spCmd = m.spinner.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd, spCmd, m.listenForRenderTriggers)
}

// Custom message type for render events
type renderEvent struct{}

// Listen for render triggers in a goroutine
func (m *model) listenForRenderTriggers() tea.Msg {
	select {
	case <-m.renderCh:
		return renderEvent{}
	case spinnerText := <-m.spinnerDone:
		return spinnerEvent{text: spinnerText}
	}
}

// Custom message type for spinner events
type spinnerEvent struct {
	text string
}

// Update the content in the viewport
func (m *model) updateViewportContent() {
	var content strings.Builder

	// Build all messages
	for _, msg := range m.messages {
		if msg.IsUser {
			content.WriteString(userMsgStyle.Render("You: " + msg.Content))
		} else if msg.IsMarkdown {
			content.WriteString(assistantMsgStyle.Render("$ "))
			content.WriteString(string(md.Render(msg.Content, 80, 0)))
		} else {
			content.WriteString(assistantMsgStyle.Render(msg.Content))
		}
		content.WriteString("\n\n")
	}

	// Add tool calls
	for i, tc := range m.toolCalls {
		content.WriteString(boxStyle.Render(fmt.Sprintf("Tool: %s\n", tc.Name)))
		content.WriteString("Arguments:\n")
		for k, v := range tc.Args {
			content.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
		}

		if tc.HasErr {
			content.WriteString("\nError: " + errorStyle.Render(tc.ErrText) + "\n")
		} else if tc.Result != "" {
			content.WriteString("\nResult:\n" + tc.Result + "\n")
		}

		if i < len(m.toolCalls)-1 {
			content.WriteString("\n")
		}
	}

	// Add spinner if active
	if m.spinnerActive && m.activeSpinner != "" {
		content.WriteString("\n" + spinnerStyle.Render(m.spinner.View()) + " " + m.activeSpinner + "\n")
	}

	// Add error if present
	if m.err != nil {
		content.WriteString("\n" + errorStyle.Render("Error: "+m.err.Error()) + "\n")
		m.err = nil
	}

	// Add prompt if in prompt mode
	if m.inPrompt {
		content.WriteString("\n" + boxStyle.Render(m.promptText) + "\n")
		content.WriteString("[Enter] Confirm   [Esc] Reject\n")
	}

	m.viewport.SetContent(content.String())

	// Scroll to bottom when new content is added
	m.viewport.GotoBottom()
}

func (m *model) View() string {
	// If not yet ready, show loading message
	if !m.viewportReady {
		return "Loading..."
	}

	// Layout:
	// - Header
	// - Viewport with messages
	// - Text input area
	// - Help (optional)

	// Start with title
	view := titleStyle.Render("Coder - Your Programming Sidekick")
	view += "\n"

	// Add viewport with messages
	view += m.viewport.View()
	view += "\n"

	// Add the text input area
	view += fmt.Sprintf("\n%s\n", m.textarea.View())

	// Render prompt overlay if in prompt mode
	if m.inPrompt {
		// Already included in viewport
	}

	return appStyle.Render(view)
}
