# 🦙 LlamaSidekick

A menu-driven CLI tool that connects to your local Ollama installation to help with coding tasks.

## Features

- **Plan Mode**: Create development plans and break down tasks
- **Edit Mode**: Get help editing code with suggestions and diffs
- **Agent Mode**: Autonomous multi-step task execution
- **CMD Mode**: Generate commands (copies to clipboard, never executes)
- **Model Configuration**: Auto-discover available Ollama models and assign different models to different modes

## Prerequisites

- [Ollama](https://ollama.ai) installed and running
- Go 1.21 or later

## Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/llamasidekick.git
cd llamasidekick

# Install dependencies
go mod download

# Build
go build -o llamasidekick

# Run
./llamasidekick
```

## Configuration

On first run, LlamaSidekick creates a config file at:
- **Linux**: `~/.config/llamasidekick/config.yaml`
- **Windows**: `%APPDATA%\llamasidekick\config.yaml`

Default configuration:
```yaml
ollama:
  host: http://localhost:11434
  model: codellama:7b
  temperature: 0.7
  debug: false  # Set to true to see detailed request/response logs
models:
  plan: codellama:7b
  edit: codellama:7b
  agent: codellama:7b
  cmd: codellama:7b
ui:
  theme: default
```

### Debug Mode

Enable debug mode to see exactly what's being sent to Ollama and what responses are received:

```yaml
ollama:
  debug: true
```

When enabled, you'll see:
- Model name being used
- Temperature setting
- System prompt (instructions to the model)
- User prompt (your input)
- Full response from Ollama

This is useful for troubleshooting model behavior or understanding how prompts are structured.

You can assign different models to different modes for optimal performance. For example, use a larger model for agent mode and a faster model for CMD mode.

Edit this file to customize your settings, or use the **Configure Models** menu option in the CLI.

## Usage

Simply run:
```bash
llamasidekick
```

Navigate the menu with arrow keys or `j`/`k`, select a mode with Enter, and type `q` to quit.

### Mode Details

#### Plan Mode
Ask questions about planning features, breaking down tasks, or structuring projects. The assistant will help you create clear, actionable development plans.

#### Edit Mode
Get help with code modifications, refactoring, and improvements. Share code snippets and ask for suggestions.

#### Agent Mode
For complex, multi-step tasks that require autonomous problem-solving and execution planning.

#### CMD Mode
Ask how to perform tasks via command line. Commands are automatically copied to your clipboard - just paste and run! **Never executes commands automatically.**

#### Configure Models
Select this option to:
- Auto-discover all available Ollama models on your system
- Assign different models to each mode (Plan, Edit, Agent, CMD)
- Optimize performance by using faster models for simple tasks and more capable models for complex tasks

For example, you might use:
- `codellama:7b` for quick CMD suggestions
- `deepseek-coder:33b` for complex Agent tasks
- `llama3:70b` for detailed Plan mode reasoning

## Session Management

LlamaSidekick saves session data in your project's `.llamasidekick/` folder, including:
- Conversation history
- Active files
- Current mode

This folder is gitignored by default.

## Development

```bash
# Run without building
go run .

# Run tests
go test ./...

# Format code
go fmt ./...
```

## License

MIT
