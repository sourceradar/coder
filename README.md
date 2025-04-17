# Coder

Coder is a command-line AI programming assistant that provides help with coding tasks using OpenAI's APIs and local programming tools.

## Features

- Interactive chat interface with an AI assistant
- Tool integration for shell commands, file operations, and more
- Customizable system prompts with templating
- Configuration management
- Terminal UI with colors and spinners

### Available Tools

- **Shell**: Execute shell commands 
- **LS**: List files and directories
- **Glob**: Find files matching glob patterns
- **Sed**: Replace text in files
- **Grep**: Search for patterns in files

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
- `/prompt` - Manage prompt templates
- `/version` - Show version information

### Prompt Management

You can customize the system prompt used by the AI:

- `/prompt` - Show current prompt
- `/prompt edit` - Edit the current prompt
- `/prompt list` - List available prompts
- `/prompt new NAME` - Create a new prompt template
- `/prompt use NAME` - Switch to a different prompt
- `/prompt reset` - Reset to the default prompt

## Configuration

Configuration is stored in `~/.coder/config.yaml` and can be edited directly or via the `/config` command:

```yaml
provider:
  api_key: "your-api-key"
  model: "gpt-3.5-turbo"
  temperature: 0.7
  max_tokens: 1000
ui:
  color_enabled: true
  show_spinner: true
prompt:
  template_file: "default.go.tmpl"
```

## Legacy Tool Usage

You can also use the individual tools directly (legacy mode):

```
coder <tool> [--param=value ...]
```

For example:

```bash
# Execute shell commands
coder shell --command="ls -la"

# List files in a directory
coder ls --path="." --recursive=false

# Find files by glob pattern
coder glob --pattern="**/*.go" --root="."

# Replace text in files
coder sed --file="config.json" --pattern="localhost" --replacement="127.0.0.1" --useRegex=true

# Search for patterns in files
coder grep --pattern="func" --paths="[\"main.go\", \"tools/\"]" --recursive=true
```