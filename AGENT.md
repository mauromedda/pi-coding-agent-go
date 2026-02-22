# pi-coding-agent-go

A Go-based implementation of the pi coding agent, providing an interactive terminal UI for AI-assisted development with file operations, permission controls, and built-in tools.

## Quick Start

```bash
# Build
make build

# Run (interactive mode)
make run

# Run with a prompt (non-interactive)
./.pi -p "What is the current directory structure?"

# Print mode (stream to stdout)
./.pi --print "Explain the project structure"
```

## Architecture

```
.pi/
├── cmd/.pi/          # CLI entry point with flag parsing
├── internal/           # Core implementation
│   ├── agent/          # Agent tool wrapper
│   ├── config/         # Configuration loading and management
│   ├── diff/           # Diff generation and display
│   ├── eventbus/       # Event publishing/subscription
│   ├── ide/            # IDE-specific integrations
│   ├── mcp/            # MCP (Model Context Protocol) support
│   ├── memory/         # Persistent memory for context
│   ├── mode/           # Interactive, print, RPC modes
│   ├── permission/     # Sandbox and permission checking
│   ├── prompt/         # System prompt building
│   ├── sandbox/        # Path sandboxing for security
│   ├── session/        # Session state management
│   ├── statusline/     # Status line display
│   └── tools/          # Built-in tools (read, write, edit, grep, bash, etc.)
├── pkg/
│   ├── ai/             # AI provider abstractions
│   └── tui/            # Terminal UI components
└── scripts/            # Utility scripts
```

### Core Components

- **Provider Abstraction** (`pkg/ai`): Unified interface for AI providers (Anthropic, OpenAI, Google, Vertex)
- **Tool Registry** (`internal/tools`): Built-in tools for file operations, system access, and web search
- **Permission System** (`internal/permission`): Path sandboxing and tool execution controls (Yolo, AcceptEdits, Normal modes)
- **Memory System** (`internal/memory`): Persistent memory that survives across sessions
- **TUI** (`pkg/tui`): Terminal UI with interactive mode, status line, and crash recovery

## Features

### Built-in Tools

| Tool | Description | Read-Only |
|------|-------------|-----------|
| `read` | Read file contents with offset/limit | Yes |
| `write` | Write content to a file | No |
| `edit` | Replace text in a file (single or all occurrences) | No |
| `grep` | Search files with regex | Yes |
| `bash` | Execute shell commands | No |
| `ls` | List directory contents | Yes |
| `web_search` | Search the web (requires search provider) | Yes |
| `web_fetch` | Fetch URL content | Yes |

### Permission Modes

| Mode | Description |
|------|-------------|
| `yolo` | No permission checks; all tools allowed |
| `accept_edits` | Read-only tools allowed; edits require acceptance |
| `normal` | Full permission checks with allow/deny/ask rules |

### Configuration

Configuration is loaded from multiple sources (in priority order):

1. CLI flags (e.g., `--model`, `--base-url`)
2. Environment variables
3. `~/.pi/agent/settings.json`
4. Project-local `.pi/settings.json`
5. Default values

### Memory System

Memory entries persist across sessions and provide context to the agent:

- **LEARN:** Persistent corrections (e.g., "Always use gofmt")
- **PROJECT:** Project-specific context
- **SESSION:** Temporary session notes

Memory is automatically loaded from `~/.pi/agent/memory/` and `.pi/memory/`.

## Usage

### Interactive Mode

```bash
./.pi
```

Starts an interactive terminal session. Available commands within the session:

| Command | Description |
|---------|-------------|
| `exit` | Exit the application |
| `clear` | Clear the screen |
| `memory <note>` | Add a memory note |
| `list memory` | List memory entries |

### Print Mode

```bash
./.pi --print "Describe the codebase structure"
```

Non-interactive mode that streams output to stdout.

### Inline Prompt

```bash
./.pi -p "What is in this file?" --file main.go
```

Quick one-shot prompts without entering interactive mode.

## AI Providers

| Provider | Environment Variable | Notes |
|----------|---------------------|-------|
| Anthropic | `ANTHROPIC_API_KEY` | Default model: `claude-3-5-sonnet-20241022` |
| OpenAI | `OPENAI_API_KEY` | Compatible with OpenAI and compatible APIs |
| Google | `GOOGLE_API_KEY` | Google Gemini models |
| Vertex | `GOOGLE_CLOUD_CREDENTIALS` | Google Cloud Vertex AI |

Custom base URLs can be specified with `--base-url` for self-hosted providers.

## Tool Execution

### Path Validation

All file operations are validated against a path sandbox to prevent unauthorized access:

- Paths must be within allowed directories
- Binary files are detected and handled appropriately
- Large files are truncated in output (default: 100KB max output, 10MB max read)

### Permission Checking

Tools are checked against the current permission mode:

1. **Yolo mode:** No checks; all tools execute immediately
2. **Accept Edits mode:** Read-only tools allowed; edits prompt for confirmation
3. **Normal mode:** Full allow/deny/ask rules apply via glob patterns

## Status Line

The status line displays current state (model, mode, memory count) and can be customized:

```json
{
  "status_line": {
    "command": "echo '.pi: $MODEL ($MODE)'",
    "padding": 1
  }
}
```

## Development

```bash
# Run all quality checks
make check

# Run tests with race detector
make test

# Lint
make lint

# Format
make fmt

# Build
make build

# Install
make install

# Snapshot release
make snapshot
```

## Configuration Files

### `~/.pi/agent/settings.json`

Global settings:

```json
{
  "model": "claude-3-5-sonnet-20241022",
  "base_url": "",
  "default_mode": "normal",
  "status_line": {
    "command": "",
    "padding": 0
  }
}
```

### `.pi/settings.json`

Project-local settings (overrides global):

```json
{
  "model": "claude-3-5-sonnet-20241022",
  "permission": {
    "allow": ["read", "bash"],
    "deny": ["web_search"],
    "ask": ["write"]
  }
}
```

## Security

- **Path sandboxing:** All file paths validated against allowed directories
- **Tool permissions:** Granular control over which tools can execute
- **Output limits:** Large outputs are truncated to prevent token exhaustion
- **Binary detection:** Binary files are detected and handled safely

## Examples

### Add a LEARN Memory Entry

In interactive mode:
```
memory Always use gofmt for Go formatting
```

### List Memory Entries

```
list memory
```

### Run with Custom Model

```bash
./.pi --model gemini-2.0-flash-exp
```

### Skip Permission Checks (Dangerous)

```bash
./.pi --dangerously-skip-permissions
```

## License

MIT
