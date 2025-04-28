# Coder

Coder is a command-line AI programming assistant that provides help with coding tasks using OpenAI's APIs and local
programming tools.

## Installation

```bash
# Clone the repository
git clone https://github.com/recrsn/coder.git

# Build the application
cd coder
go build

# Run coder
./coder
```

## Usage

1. Set up your OpenAI API key:

```bash
# First-time setup
./coder
# Then use the command
/config provider.api_key=YOUR_API_KEY
```

2. Start chatting with the assistant by typing your programming questions.

3. Use tools by asking the AI to perform tasks like running shell commands, searching files, etc.

## Commands

Coder supports the following commands:

- `/help` - Show help message with available commands
- `/exit` - Exit the application
- `/clear` - Clear the screen
- `/config` - Show or edit configuration
- `/tools` - List available tools
- `/version` - Show version information

## Configuration

Configuration is stored in `~/.coder/config.yaml` and can be edited directly or via the `/config` command:

```yaml
provider:
  api_key: "your-api-key"
  model: "gpt-4o"
  endpoint: "https://api.openai.com/v1/chat/completions"
ui:
  color_enabled: true
  show_spinner: true
```