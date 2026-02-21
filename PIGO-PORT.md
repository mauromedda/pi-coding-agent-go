# Plan: Port pi-coding-agent + TUI to Go

## Context

The TypeScript `pi-mono` monorepo contains two key packages: `pi-tui` (a terminal UI framework with differential rendering, component system, overlay support) and `pi-coding-agent` (an AI coding agent harness with session management, tool execution, LLM streaming, and compaction). This plan ports both to Go as a high-performance, secure implementation with the TUI as a reusable public SDK.

**Binary name**: `pi-go`
**Module path**: `github.com/mauromedda/pi-coding-agent-go`
**Go version**: 1.24+
**Extensions**: Deferred (MVP without plugin system)

---

## Review History

### Gemini Architectural Review (2026-02-21)

Rating: **A-**. Key findings addressed in this revision:

| Finding | Severity | Resolution |
|---------|----------|------------|
| Multi-module `go.work` is premature | Critical | Collapsed to single `go.mod` |
| `Render() []string` causes GC pressure | Critical | Changed to `Render(out *RenderBuffer, width int)` with pool |
| No terminal crash recovery | Critical | Added `RestoreOnPanic()` in `main.go` defer |
| Bash tool needs PTY for ANSI output | Critical | Added `creack/pty` dependency |
| `atomic.Bool` render coalescing not idiomatic | High | Changed to buffered channel of size 1 |
| Fuzzy search has no dependency | High | Added `sahilm/fuzzy` |
| Windows VT processing not addressed | High | Added `SetConsoleMode` in `process_windows.go` |
| No IDE integration | Major | Added `internal/ide/` package |
| No checkpoint/rewind system | Major | Added git-based checkpointing |

---

## Architecture: Single-Module Monorepo

**Decision**: Single `go.mod` at root. `pkg/tui/` and `pkg/ai/` are standard importable packages, not separate modules. Extract to independent modules later when external consumers exist.

**Rationale**: Multi-module `go.work` adds CI/CD friction (independent tagging, cross-module replace directives) with no current benefit. YAGNI.

```
pi-coding-agent-go/
├── go.mod                                # Single module: github.com/mauromedda/pi-coding-agent-go
├── go.sum
├── Makefile                              # Build, test, lint, release
├── .goreleaser.yml                       # Cross-platform binary distribution
├── .golangci.yml                         # Linter config
├── .github/
│   └── workflows/
│       ├── ci.yml                        # PR/push: lint, test, build
│       └── release.yml                   # Tag-triggered: build + GitHub Release
├── install.sh                            # Curl-pipe installer (inspired by Rust version)
│
├── pkg/
│   ├── tui/                              # Public TUI SDK (importable)
│   │   ├── component.go                  # Component, InputHandler, Focusable interfaces
│   │   ├── renderbuffer.go               # RenderBuffer: pooled line buffer
│   │   ├── container.go                  # Container struct (ordered children)
│   │   ├── tui.go                        # TUI engine: differential rendering, focus, overlays
│   │   ├── overlay.go                    # Overlay types, positioning, compositing
│   │   ├── terminal/
│   │   │   ├── terminal.go               # Terminal interface
│   │   │   ├── process.go                # ProcessTerminal: raw mode, Kitty protocol
│   │   │   ├── process_unix.go           # Unix: SIGWINCH, tcsetattr
│   │   │   ├── process_windows.go        # Windows: VT input mode + SetConsoleMode
│   │   │   ├── restore.go                # RestoreOnPanic: crash recovery
│   │   │   └── virtual.go               # VirtualTerminal: test mock
│   │   ├── key/
│   │   │   ├── key.go                    # Key type, ParseKey, MatchesKey
│   │   │   ├── kitty.go                  # Kitty keyboard protocol parser (stub Phase 1; impl Phase 5)
│   │   │   └── legacy.go                 # Legacy escape sequence maps
│   │   ├── input/
│   │   │   └── buffer.go                 # StdinBuffer: sequence completion, bracketed paste
│   │   ├── width/
│   │   │   ├── width.go                  # VisibleWidth (grapheme-aware, cached)
│   │   │   ├── ansi.go                   # ANSI code extraction/tracking
│   │   │   ├── slice.go                  # SliceByColumn, ExtractSegments
│   │   │   └── wrap.go                   # WrapTextWithAnsi, TruncateToWidth
│   │   ├── component/
│   │   │   ├── editor.go                 # Multi-line: word-wrap, undo, kill ring, autocomplete
│   │   │   ├── input.go                  # Single-line: horizontal scroll, undo, kill ring
│   │   │   ├── markdown.go               # Terminal markdown renderer (ANSI styled)
│   │   │   ├── selectlist.go             # Filterable scrollable list
│   │   │   ├── box.go                    # Padding container with background
│   │   │   ├── text.go                   # Static text display
│   │   │   ├── truncatedtext.go          # Single-line with ellipsis
│   │   │   ├── loader.go                 # Spinner animation
│   │   │   ├── spacer.go                 # Vertical spacer
│   │   │   ├── history.go               # Ctrl+R searchable prompt history
│   │   │   └── image.go                  # Kitty + iTerm2 inline images
│   │   ├── fuzzy/
│   │   │   └── fuzzy.go                  # Thin wrapper over sahilm/fuzzy
│   │   └── internal/
│   │       ├── ansitrack/
│   │       │   └── tracker.go            # SGR state machine
│   │       ├── killring/
│   │       │   └── killring.go           # Emacs-style ring buffer
│   │       ├── undo/
│   │       │   └── undo.go              # Generic UndoStack[S]
│   │       └── pool/
│   │           └── buffers.go            # sync.Pool for bytes.Buffer, strings.Builder
│   │
│   └── ai/                               # Public AI SDK (importable)
│       ├── types.go                       # Message, Content, Tool, Usage, Model, StopReason
│       ├── stream.go                      # EventStream[T] via channels
│       ├── registry.go                    # Provider registry
│       ├── models.go                      # Built-in model definitions
│       ├── provider/
│       │   ├── provider.go               # ApiProvider interface
│       │   ├── anthropic/
│       │   │   ├── anthropic.go          # Anthropic Messages API streaming
│       │   │   └── convert.go            # Message format conversion
│       │   ├── openai/
│       │   │   ├── openai.go             # OpenAI Chat Completions (+ Ollama, vLLM)
│       │   │   ├── convert.go            # Message format conversion
│       │   │   └── compat.go             # Compatibility flags for local inference
│       │   ├── google/
│       │   │   ├── google.go             # Google Generative AI
│       │   │   └── convert.go            # Message format conversion
│       │   └── vertex/
│       │       ├── vertex.go             # Google Vertex AI
│       │       └── convert.go            # Message format conversion
│       └── internal/
│           ├── sse/
│           │   └── reader.go             # Server-Sent Events parser
│           └── httputil/
│               └── client.go             # Shared HTTP client, retries, proxy
│
├── internal/                              # APPLICATION (not importable)
│   ├── agent/
│   │   ├── agent.go                      # Agent loop: prompt -> stream -> tools -> repeat
│   │   ├── types.go                      # AgentEvent, AgentTool, AgentState
│   │   └── tool.go                       # Tool validation, argument parsing
│   ├── session/
│   │   ├── session.go                    # AgentSession orchestrator (model, retry, compaction)
│   │   ├── persistence.go                # JSONL format: read/write/tree navigation
│   │   ├── compaction.go                 # Context compaction: summarize old, keep recent
│   │   └── branch.go                     # Branch summarization
│   ├── tools/
│   │   ├── registry.go                   # Tool registry: create, activate/deactivate
│   │   ├── read.go                       # Read files (offset/limit, image detect, truncation)
│   │   ├── write.go                      # Write/create files, auto-create dirs
│   │   ├── edit.go                       # Surgical replacement, fuzzy fallback, unified diff
│   │   ├── bash.go                       # Shell exec via PTY: streaming, timeout, process tree kill
│   │   ├── grep.go                       # Ripgrep wrapper: JSON mode, match limiting
│   │   ├── grep_builtin.go              # Fallback grep when rg not installed
│   │   ├── find.go                       # File discovery via ripgrep --files
│   │   ├── find_builtin.go              # Fallback find when rg not installed
│   │   └── ls.go                         # Directory listing, .gitignore respect
│   ├── ide/
│   │   ├── detect.go                     # Auto-detect IDE from env vars + process inspection
│   │   ├── editor.go                     # $EDITOR launch for Ctrl+G (plan/prompt editing)
│   │   ├── diff.go                       # IDE-aware diff output (native vs terminal)
│   │   ├── checkpoint.go                 # Git-based snapshot before tool execution; rewind support
│   │   └── filemention.go               # Parse @file#line-line syntax, resolve, inject context
│   ├── config/
│   │   ├── config.go                     # Settings: global + project, deep merge
│   │   ├── paths.go                      # Standard paths (~/.pi-go/, .pi-go/)
│   │   ├── auth.go                       # Auth: API keys, file locking (flock)
│   │   └── models.go                     # Model registry: built-in + custom, provider resolution
│   ├── prompt/
│   │   ├── system.go                     # System prompt: tools + context files + skills + date/cwd
│   │   └── skills.go                     # Skill loading (SKILL.md parsing)
│   ├── permission/
│   │   ├── permission.go                 # Permission checker: normal/yolo/plan modes
│   │   └── sandbox.go                    # Path validation, allowed directories
│   ├── mode/
│   │   ├── interactive/
│   │   │   ├── interactive.go            # Interactive TUI mode: main loop, mode switching
│   │   │   ├── planmode.go               # Plan mode: read-only, tool restrictions
│   │   │   ├── editmode.go               # Edit mode: full tool access
│   │   │   └── components/
│   │   │       ├── footer.go             # Status bar: model, mode, costs, Shift+Tab hint
│   │   │       ├── assistant_msg.go      # Assistant message (markdown + tool calls)
│   │   │       ├── user_msg.go           # User message display
│   │   │       ├── tool_exec.go          # Tool execution progress
│   │   │       ├── diff_view.go          # Unified diff display
│   │   │       ├── permission_dialog.go  # Permission prompt overlay
│   │   │       ├── model_selector.go     # Model picker overlay
│   │   │       └── session_selector.go   # Session picker overlay
│   │   ├── print/
│   │   │   └── print.go                 # Non-interactive: pipe/script output
│   │   └── rpc/
│   │       ├── rpc.go                    # RPC mode for external integrations
│   │       └── types.go                  # RPC request/response types
│   └── eventbus/
│       └── eventbus.go                   # Typed event bus with subscriber management
│
├── cmd/
│   └── pi-go/
│       ├── main.go                       # CLI entry point + RestoreOnPanic defer
│       └── flags.go                      # --yolo, --model, --plan, --print, --thinking, etc.
│
└── scripts/
    └── generate_models.go                # Fetch/generate built-in model definitions
```

---

## Key Interfaces

### TUI SDK: Component System

```go
// pkg/tui/component.go

type Component interface {
    Render(out *RenderBuffer, width int)  // Write lines into pooled buffer; lines must not exceed width
    Invalidate()                          // Clear cached render state
}

type InputHandler interface {
    HandleInput(data string)
}

type Focusable interface {
    SetFocused(focused bool)
    IsFocused() bool
}

const CursorMarker = "\x1b_pi:c\x07"  // Zero-width; TUI strips and positions cursor
```

### TUI SDK: RenderBuffer

```go
// pkg/tui/renderbuffer.go

// RenderBuffer is a pooled line buffer that components write into.
// The TUI engine allocates from sync.Pool and recycles after each frame.
type RenderBuffer struct {
    Lines []string
}

func (b *RenderBuffer) WriteLine(line string)
func (b *RenderBuffer) WriteLines(lines []string)
func (b *RenderBuffer) Reset()
func (b *RenderBuffer) Len() int
```

**Rationale**: Avoids per-frame `[]string` allocations. Components write into a shared buffer recycled via `sync.Pool`. Simpler than a full `ScreenBuffer`/`Rect` grid, which would over-engineer the layering model.

### TUI SDK: Render Coalescing

```go
// pkg/tui/tui.go — render notification channel

renderCh := make(chan struct{}, 1)

func (t *TUI) RequestRender() {
    select {
    case t.renderCh <- struct{}{}:
    default: // Already pending; coalesced
    }
}
```

**Rationale**: Buffered channel of size 1 is more idiomatic than `atomic.Bool` for "notify if not already pending". The render goroutine blocks on `<-renderCh`; natural coalescing, zero polling.

### TUI SDK: Terminal Crash Recovery

```go
// pkg/tui/terminal/restore.go

// RestoreOnPanic recovers from panics, restores the terminal to cooked mode,
// shows the cursor, prints the panic trace, and exits with code 1.
// Must be deferred in main().
func RestoreOnPanic() {
    if r := recover(); r != nil {
        // 1. Restore terminal to cooked mode
        // 2. Show cursor: \x1b[?25h
        // 3. Disable raw mode via x/term
        // 4. Print panic with stack trace
        // 5. os.Exit(1)
    }
}
```

### AI SDK: Provider Abstraction

```go
// pkg/ai/provider/provider.go

type ApiProvider interface {
    Api() ai.Api
    Stream(ctx context.Context, model *ai.Model, llmCtx *ai.Context, opts *ai.StreamOptions) *ai.AssistantStream
}

// pkg/ai/stream.go — Channel-based event streaming

type EventStream[T any] struct {
    events chan T
    done   chan struct{}
    result atomic.Pointer[ai.AssistantMessage]
}

func (s *EventStream[T]) Events() <-chan T    // Range over events
func (s *EventStream[T]) Result() *AssistantMessage  // Block until done
```

### Agent: Tool Interface

```go
// internal/agent/types.go

type AgentTool struct {
    Name        string
    Label       string
    Description string
    Parameters  json.RawMessage  // JSON Schema
    ReadOnly    bool             // If true, can run concurrently with other read-only tools
    Execute     func(ctx context.Context, id string, params map[string]any,
                     onUpdate func(ToolUpdate)) (ToolResult, error)
}
```

**Addition**: `ReadOnly bool` enables parallel execution of read-only tools (read, grep, find, ls) within a single agent turn, while write/bash tools remain strictly sequential.

### Permission System

```go
// internal/permission/permission.go

type Mode int

const (
    ModeNormal Mode = iota  // Prompt user for dangerous ops
    ModeYolo                // Skip all prompts (--yolo)
    ModePlan                // Read-only: block write/bash
)

type Checker struct {
    mode       Mode
    allowRules []Rule
    denyRules  []Rule
    askFn      func(tool string, args map[string]any) (bool, error)
}
```

### IDE Integration

```go
// internal/ide/detect.go

type IDE int

const (
    IDENone IDE = iota
    IDEVSCode
    IDEJetBrains
    IDEOther
)

// Detect checks environment variables and process tree to identify the running IDE.
// Checks: TERM_PROGRAM, VSCODE_PID, VSCODE_GIT_ASKPASS_MAIN,
//         JETBRAINS_IDE_PORT, TERMINAL_EMULATOR
func Detect() IDE
```

```go
// internal/ide/editor.go

// OpenInEditor writes content to a temp file, launches $EDITOR (or $VISUAL, or vi),
// blocks until the editor exits, reads back the edited content, and returns it.
// The TUI must exit raw mode before calling this and restore it after.
func OpenInEditor(content string) (edited string, err error)
```

```go
// internal/ide/checkpoint.go

// Checkpoint captures the current working tree state before a tool execution.
// In a git repo: uses `git stash create` to produce a commit-like ref.
// Outside git: copies modified files to ~/.pi-go/checkpoints/<session-id>/<seq>/
type Checkpoint struct {
    Ref       string    // Git stash ref or checkpoint directory path
    Timestamp time.Time
    ToolName  string    // Tool that triggered the checkpoint
    ToolArgs  string    // Summary of tool arguments
}

// CheckpointStack manages a stack of checkpoints for the current session.
type CheckpointStack struct {
    stack []Checkpoint
}

func (s *CheckpointStack) Save(toolName, toolArgs string) error
func (s *CheckpointStack) Rewind() error       // Restore most recent checkpoint
func (s *CheckpointStack) RewindTo(n int) error // Restore nth checkpoint
func (s *CheckpointStack) List() []Checkpoint
```

```go
// internal/ide/filemention.go

// FileMention represents a parsed @file#line-line reference from user input.
type FileMention struct {
    Path      string
    StartLine int  // 0 if not specified
    EndLine   int  // 0 if not specified
}

// ParseMentions extracts all @file#line-line references from input text.
// Returns the cleaned text (mentions replaced with inline content) and the parsed mentions.
func ParseMentions(input string, workDir string) (cleaned string, mentions []FileMention, err error)
```

---

## Operating Modes (Claude Code UX)

### Plan Mode (default on startup)
- Read-only operations: `read`, `grep`, `find`, `ls` allowed
- `write`, `edit`, `bash` blocked with message "Switch to Edit mode (Shift+Tab)"
- Footer shows: `[PLAN]` indicator + `Shift+Tab -> Edit`
- Agent receives system prompt addendum restricting to analysis/planning

### Edit Mode (Shift+Tab to toggle)
- Full tool access (subject to permission checks)
- Footer shows: `[EDIT]` indicator + `Shift+Tab -> Plan`
- Permission prompts for dangerous operations unless `--yolo`

### Yolo Mode (`pi-go --yolo`)
- Edit mode with all permission checks bypassed
- Footer shows: `[YOLO]` warning indicator
- Equivalent to Claude Code's `--dangerously-skip-permissions`

---

## Keyboard Shortcuts

| Shortcut | Action | Context |
|----------|--------|---------|
| `Shift+Tab` | Cycle mode: Plan -> Edit -> Plan | Always |
| `Ctrl+G` | Open current prompt/plan in `$EDITOR` | Chat input / Plan view |
| `Ctrl+O` | Toggle verbose output (show tool calls) | Always |
| `Ctrl+R` | Searchable prompt history | Chat input |
| `Ctrl+C` | Cancel current agent operation | During agent run |
| `Ctrl+L` | Clear terminal screen | Always |
| `Esc+Esc` | Rewind to last checkpoint | After tool execution |
| `@file#L-L` | File mention with optional line range | Chat input |

---

## Data Flow

### Agent Loop

```
User Input -> AgentSession.Prompt()
    |
    |-- Parse @file#line mentions (ide/filemention.go)
    |-- Check permission mode (plan: restrict tools)
    |-- Build system prompt (tools + context + skills + mode)
    |-- Build message context (session tree -> leaf)
    |
    +-- [LOOP]
         |-- provider.Stream(model, context, opts)
         |     +-- SSE goroutine reads HTTP response body
         |         +-- events channel -> agent events -> TUI updates
         |
         |-- For each ToolCall in response:
         |     |-- Permission check (mode + rules)
         |     |-- Checkpoint current state (ide/checkpoint.go)
         |     |-- Batch read-only tools for parallel execution (errgroup)
         |     |-- tool.Execute(ctx, id, params, onUpdate)
         |     +-- Append ToolResult to messages
         |
         |-- Check steering channel (non-blocking)
         |     +-- If steering: interrupt, inject message, continue
         |
         |-- Check follow-up queue
         |     +-- If pending: set as next input, continue
         |
         +-- No more tool calls & no follow-up -> emit agent_end
              +-- Check auto-compaction threshold
```

### TUI Rendering Pipeline

```
RequestRender() -> renderCh (buffered chan, size 1, auto-coalesced)
    |
    +-- Render goroutine:
         |-- Acquire RenderBuffer from sync.Pool
         |-- Container.Render(buf, width) -- all children
         |-- compositeOverlays(buf) -- modal dialogs on top
         |-- extractCursorPosition(buf) -- find + strip CursorMarker
         |-- diff buf.Lines vs previousLines -- find changed range
         |-- Build CSI 2026 synchronized output buffer
         |     +-- Only changed lines: \x1b[2K + new content
         |-- terminal.Write(output) -- single atomic write
         +-- Return RenderBuffer to sync.Pool
```

### IDE Integration Flow

```
User types @src/main.go#10-20 in prompt
    |
    +-- filemention.ParseMentions()
         |-- Resolve path relative to workDir
         |-- Read lines 10-20 from file
         +-- Inject content as context block in user message

User presses Ctrl+G
    |
    +-- TUI exits raw mode
         |-- ide.OpenInEditor(currentPrompt)
         |     |-- Write to temp file
         |     |-- exec.Command($EDITOR, tempFile)
         |     |-- Wait for editor to exit
         |     +-- Read back edited content
         +-- TUI restores raw mode, updates prompt

Tool about to execute (write/edit/bash)
    |
    +-- checkpointStack.Save(toolName, toolArgs)
         |-- git stash create (if in git repo)
         +-- Push checkpoint ref to stack

User presses Esc+Esc (or /rewind)
    |
    +-- checkpointStack.Rewind()
         |-- git stash apply <ref>
         +-- Pop checkpoint from stack
```

---

## Concurrency Model

| Goroutine | Purpose | Lifetime |
|-----------|---------|----------|
| Main | CLI parsing, TUI event loop | Process |
| Stdin reader | `os.Stdin.Read()` -> StdinBuffer | While TUI active |
| Render | Coalesced rendering via buffered channel | While TUI active |
| Agent loop | `Agent.Prompt()` -> one per interaction | Until agent_end |
| SSE reader | Per-LLM-call HTTP streaming | Until stream done |
| Tool exec | Read-only: concurrent via errgroup; Write: sequential | Until tool returns |

**Key patterns:**
- `context.Context` propagation for cancellation (Ctrl+C -> cancel agent -> cancel tools -> cancel HTTP)
- Buffered channel (size 1) for render coalescing (multiple invalidations -> one render)
- `sync.RWMutex` on Container.children for concurrent render vs mutation
- Channels for event streaming (SSE -> agent events -> TUI)
- `errgroup.Group` for parallel read-only tool execution within a turn

---

## Performance Strategy

| Area | Technique |
|------|-----------|
| Rendering | Differential: only changed lines sent to terminal |
| Rendering | CSI 2026 synchronized output: flicker-free atomic updates |
| Rendering | `sync.Pool` for `RenderBuffer` and `bytes.Buffer` |
| Rendering | Buffered channel coalescing: no polling, zero-cost when idle |
| Width calc | LRU cache (512 entries) for non-ASCII strings |
| Width calc | Fast path: pure ASCII -> `len(s)`, no segmentation |
| Streaming | `bufio.Scanner` directly on `http.Response.Body` |
| Sessions | JSONL read with `bufio.Scanner`: line-by-line, never full load |
| Bash tool | PTY-based exec via `creack/pty`; large output -> temp file |
| Tools | Read-only tools (read, grep, find, ls) run concurrently via errgroup |
| Binary | goreleaser: `ldflags: -s -w` (strip symbols), CGO_ENABLED=0 |

---

## Security

| Area | Mechanism |
|------|-----------|
| File access | `filepath.Abs` + prefix check against allowed dirs |
| Path traversal | Reject `..` components in LLM-generated paths |
| Bash execution | PTY subprocess with controlled env; permission prompt in normal mode |
| Auth storage | `~/.pi-go/auth.json` with `0600` perms; `flock(2)` for locking |
| API keys | Never logged; env vars preferred; OAuth with mutex-protected refresh |
| Input validation | Tool params validated against JSON Schema before execution |
| Yolo mode | Explicit opt-in flag; warning in footer |
| Checkpoint | Git stash-based; no data leaves local filesystem |
| Terminal crash | `RestoreOnPanic()` deferred in main: always restores cooked mode |

---

## Dependencies

| Dependency | Purpose | Phase |
|------------|---------|-------|
| `github.com/rivo/uniseg` | Grapheme cluster segmentation | 1 |
| `github.com/mattn/go-runewidth` | East Asian width + emoji width | 1 |
| `golang.org/x/term` | Raw mode, terminal size | 1 |
| `golang.org/x/sys` | SIGWINCH, flock, Windows VT | 1 |
| `github.com/sahilm/fuzzy` | Fuzzy string matching/filtering | 1 |
| `github.com/creack/pty` | PTY for bash tool (Unix only) | 3 |
| `github.com/google/uuid` | Session IDs | 4 |
| `github.com/yuin/goldmark` | Markdown parsing for terminal render | 4 |
| `gopkg.in/yaml.v3` | SKILL.md frontmatter parsing | 4 |
| `golang.org/x/oauth2` | OAuth2 token refresh (Vertex AI) | 5 |
| `golang.org/x/oauth2/google` | Google service account auth | 5 |

**No Cobra, no Viper, no Bubble Tea, no YAML/TOML libs.** CLI uses stdlib `flag`; config is JSON via `encoding/json`.

### Runtime Dependencies (optional)

| Tool | Purpose | Fallback |
|------|---------|----------|
| `rg` (ripgrep) | Fast grep + file discovery | `grep_builtin.go` / `find_builtin.go` using stdlib |
| `git` | Checkpoint/rewind, session context | File-copy checkpoints; degraded mode |

---

## Binary Distribution (Inspired by Rust pi_agent)

### goreleaser Configuration
- **Targets**: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
- **Format**: `.tar.gz` (Unix), `.zip` (Windows)
- **Contents**: binary + LICENSE + README.md
- **Checksums**: `SHA256SUMS` file with `LC_ALL=C sort`
- **Build flags**: `CGO_ENABLED=0`, `ldflags: -s -w -X main.version={{.Version}}`

### GitHub Actions Release Workflow
```yaml
on:
  push:
    tags: ['v[0-9]+.*']

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.24' }
      - uses: goreleaser/goreleaser-action@v6
        with: { args: 'release --clean' }
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### Install Script (`install.sh`)
Inspired by the Rust version's comprehensive installer:
- Auto-detect platform + architecture
- Download latest or pinned version from GitHub Releases
- SHA256 checksum verification
- Install to `~/.local/bin` (user) or `/usr/local/bin` (system)
- Offline mode support with `--offline-tarball`
- Usage: `curl -fsSL https://raw.githubusercontent.com/mauromedda/pi-coding-agent-go/main/install.sh | bash`

### CI Workflow (`ci.yml`)
- Matrix: ubuntu-latest, macos-latest, windows-latest
- Steps: `golangci-lint run`, `go test -race -count=1 ./...`, `go build ./cmd/pi-go`

---

## Implementation Phases

### Phase 1: TUI Foundation
**Files**: `pkg/tui/width/`, `pkg/tui/terminal/`, `pkg/tui/key/`, `pkg/tui/input/`, `pkg/tui/component.go`, `pkg/tui/renderbuffer.go`, `pkg/tui/container.go`, `pkg/tui/tui.go`

1. `width/width.go` -- VisibleWidth with grapheme segmentation + LRU cache
2. `width/ansi.go` -- ANSI extraction, SGR tracking
3. `width/slice.go` -- SliceByColumn, ExtractSegments
4. `width/wrap.go` -- WrapTextWithAnsi, TruncateToWidth
5. `terminal/terminal.go` -- Terminal interface
6. `terminal/process.go` -- ProcessTerminal: raw mode, resize
7. `terminal/process_unix.go` -- Unix: SIGWINCH, tcsetattr
8. `terminal/process_windows.go` -- Windows: VT input mode + `SetConsoleMode(ENABLE_VIRTUAL_TERMINAL_PROCESSING)`
9. `terminal/restore.go` -- RestoreOnPanic: crash recovery
10. `terminal/virtual.go` -- VirtualTerminal: test mock for TDD
11. `key/key.go` + `key/legacy.go` -- Key parsing (legacy escape sequences)
12. `key/kitty.go` -- Kitty protocol parser **stub** (interface only; impl in Phase 5)
13. `input/buffer.go` -- StdinBuffer: sequence completion + bracketed paste
14. `component.go` -- Component, InputHandler, Focusable interfaces
15. `renderbuffer.go` -- RenderBuffer with sync.Pool recycling
16. `container.go` -- Container struct
17. `tui.go` -- Differential rendering engine with buffered-channel coalescing
18. Basic components: `text.go`, `box.go`, `spacer.go`, `loader.go`
19. `fuzzy/fuzzy.go` -- Thin wrapper over `sahilm/fuzzy`

### Phase 2: AI Streaming (Anthropic Only)
**Files**: `pkg/ai/`

1. `types.go` -- All message, model, content, usage types
2. `stream.go` -- `EventStream[T]` with channels
3. `registry.go` -- Provider registry
4. `models.go` -- Built-in model definitions (Anthropic models only; others as constants for later)
5. `internal/sse/reader.go` -- SSE parser
6. `internal/httputil/client.go` -- Shared HTTP client with retries, proxy support
7. `provider/provider.go` -- ApiProvider interface
8. `provider/anthropic/anthropic.go` + `convert.go` -- Anthropic Messages API streaming

### Phase 2b: Additional Providers (can run parallel with Phase 3)
**Files**: `pkg/ai/provider/openai/`, `pkg/ai/provider/google/`, `pkg/ai/provider/vertex/`

1. `provider/openai/` -- OpenAI Chat Completions (+ Ollama/vLLM compat)
2. `provider/google/` -- Google Generative AI
3. `provider/vertex/` -- Vertex AI (requires `golang.org/x/oauth2`)

### Phase 3: Agent Core + Tools
**Files**: `internal/agent/`, `internal/tools/`, `internal/ide/`

1. `agent/types.go` -- AgentEvent, AgentTool (with ReadOnly flag), AgentState
2. `agent/agent.go` -- Agent struct: Prompt, Steer, Abort, Subscribe; parallel read-only tool dispatch
3. `agent/tool.go` -- Tool validation + param parsing
4. `tools/registry.go` -- Tool registry with rg detection + fallback registration
5. `tools/read.go` -- Read tool (file read, image detect, truncation)
6. `tools/write.go` -- Write tool (create/overwrite, mkdir -p)
7. `tools/edit.go` -- Edit tool (exact match, fuzzy fallback, diff)
8. `tools/bash.go` -- Bash tool via PTY (`creack/pty`): streaming, timeout, process tree kill
9. `tools/grep.go` -- Grep tool (ripgrep wrapper)
10. `tools/grep_builtin.go` -- Fallback grep using stdlib `regexp` + `filepath.WalkDir`
11. `tools/find.go` -- Find tool (ripgrep --files)
12. `tools/find_builtin.go` -- Fallback find using `filepath.WalkDir` + `.gitignore` parsing
13. `tools/ls.go` -- Ls tool (readdir + .gitignore)
14. `ide/detect.go` -- IDE auto-detection from env vars
15. `ide/checkpoint.go` -- Git stash-based checkpointing before tool execution
16. `ide/filemention.go` -- Parse @file#line-line syntax

### Phase 4: Interactive Agent
**Files**: `internal/session/`, `internal/config/`, `internal/prompt/`, `internal/permission/`, `internal/mode/`, `internal/ide/`, `cmd/pi-go/`

1. `config/paths.go` -- Standard paths
2. `config/config.go` -- Settings loading + deep merge
3. `config/auth.go` -- Auth storage with flock
4. `config/models.go` -- Model registry resolution
5. `permission/permission.go` -- Permission checker (normal/yolo/plan)
6. `permission/sandbox.go` -- Path validation
7. `prompt/system.go` -- System prompt construction
8. `prompt/skills.go` -- SKILL.md parsing
9. `session/persistence.go` -- JSONL I/O
10. `session/session.go` -- AgentSession orchestrator
11. `session/compaction.go` -- Auto-compaction
12. `ide/editor.go` -- Ctrl+G: open prompt/plan in $EDITOR
13. `ide/diff.go` -- IDE-aware diff rendering
14. TUI components: `editor.go`, `input.go`, `selectlist.go`, `markdown.go`, `history.go`
15. App components: `footer.go`, `assistant_msg.go`, `tool_exec.go`, `permission_dialog.go`
16. `mode/interactive/interactive.go` -- Plan/Edit mode switching (Shift+Tab)
17. Wire keyboard shortcuts: Ctrl+G, Ctrl+O, Ctrl+R, Esc+Esc (rewind)
18. Wire @file#line mention parsing into user input pipeline
19. `cmd/pi-go/main.go` + `flags.go` -- CLI entry point with `defer terminal.RestoreOnPanic()`

### Phase 5: Distribution + Polish
**Files**: `.goreleaser.yml`, `.github/workflows/`, `install.sh`, `Makefile`

1. `Makefile` -- build, test, lint, install targets
2. `.golangci.yml` -- Linter configuration
3. `.goreleaser.yml` -- Cross-platform release
4. `.github/workflows/ci.yml` -- PR/push CI
5. `.github/workflows/release.yml` -- Tag-triggered release
6. `install.sh` -- Curl-pipe installer
7. Kitty keyboard protocol full implementation (`key/kitty.go`)
8. Image rendering (Kitty + iTerm2) (`component/image.go`)
9. Overlay system (model/session pickers)
10. Print mode + RPC mode
11. Google + Vertex AI providers (if not done in Phase 2b)
12. Self-update command (`cmd/pi-go/update.go`): check GitHub Releases, download binary

### Phase 6: IDE Extensions (Separate Repositories)
**Files**: Separate repos; not part of this Go monorepo

1. **VS Code Extension** (TypeScript)
   - Spawn `pi-go` as child process
   - Capture stdout/stderr for native UI rendering
   - Side-by-side diff with Accept/Reject buttons
   - Webview panel for conversation history
   - `Option+K` / `Alt+K` for @-mention insertion
   - `Cmd+Esc` / `Ctrl+Esc` for toggle focus
2. **JetBrains Plugin** (Kotlin)
   - Bridge from integrated terminal to IDE diff viewer
   - Auto-share diagnostics (lint/type errors) with agent
   - `Cmd+Option+K` / `Alt+Ctrl+K` for file reference insertion

---

## Verification

### Unit Tests
```bash
go test -race -count=1 ./pkg/tui/...
go test -race -count=1 ./pkg/ai/...
go test -race -count=1 ./internal/...
```

### Integration Tests
- TUI: `VirtualTerminal` mock -- verify differential rendering, overlay compositing, key dispatch
- AI: Record/replay HTTP fixtures (VCR pattern) -- verify SSE parsing, message conversion
- Agent: Mock provider -- verify tool call loop, steering, compaction triggers, read-only parallelism
- Tools: Temp directories -- verify read/write/edit/bash/grep/find/ls
- IDE: Checkpoint/rewind with temp git repos -- verify stash create/apply cycle
- IDE: File mention parsing -- verify @path#line-line resolution

### Manual E2E
```bash
# Build and run
go build -o pi-go ./cmd/pi-go && ./pi-go

# Test mode switching
# Type a prompt -> observe Plan mode restrictions -> Shift+Tab -> Edit mode -> tool execution

# Test yolo mode
./pi-go --yolo

# Test Ctrl+G editor integration
# Press Ctrl+G in prompt -> opens $EDITOR -> edit prompt -> save -> returns to TUI

# Test @file mentions
# Type: explain @src/main.go#10-20

# Test checkpoint/rewind
# Let agent edit a file -> Esc+Esc -> verify file reverted

# Test with different providers
PI_API_KEY_ANTHROPIC=sk-... ./pi-go --model claude-sonnet-4-20250514
PI_API_KEY_OPENAI=sk-... ./pi-go --model gpt-4o
./pi-go --model ollama:llama3 --base-url http://localhost:11434/v1

# Test crash recovery
# Kill pi-go with SIGKILL -> verify terminal is usable (RestoreOnPanic)

# Test release build
goreleaser release --snapshot --clean
```

### Benchmark
```bash
go test -bench=. -benchmem ./pkg/tui/width/...
go test -bench=. -benchmem ./pkg/tui/...
```

---

## Reference Files (from TypeScript)

| Go Target | TS Reference | Purpose |
|-----------|-------------|---------|
| `pkg/tui/tui.go` | `pi-mono/packages/tui/src/tui.ts` | Differential rendering, overlay compositing |
| `pkg/tui/width/width.go` | `pi-mono/packages/tui/src/utils.ts` | Grapheme width, ANSI handling |
| `pkg/tui/key/key.go` | `pi-mono/packages/tui/src/keys.ts` | Keyboard input parsing |
| `pkg/tui/input/buffer.go` | `pi-mono/packages/tui/src/stdin-buffer.ts` | Sequence buffering |
| `pkg/tui/component/editor.go` | `pi-mono/packages/tui/src/components/editor.ts` | Multi-line editor |
| `pkg/ai/stream.go` | `pi-mono/packages/ai/src/utils/event-stream.ts` | Channel-based streaming |
| `internal/agent/agent.go` | `pi-mono/packages/agent/src/agent-loop.ts` | Agent loop logic |
| `internal/session/session.go` | `pi-mono/packages/coding-agent/src/core/agent-session.ts` | Session orchestrator |
| `internal/tools/*.go` | `pi-mono/packages/coding-agent/src/core/tools/*.ts` | Tool implementations |
| `internal/session/persistence.go` | `pi-mono/packages/coding-agent/src/core/session-manager.ts` | JSONL persistence |
| `internal/session/compaction.go` | `pi-mono/packages/coding-agent/src/core/compaction/compaction.ts` | Context compaction |

---

## Resolved Decisions

### 1. TypeScript Reference Code: ACCESSIBLE

The `pi-mono` TypeScript codebase is available for reading during the port. Use it as reference for:
- Edge-case handling and boundary conditions
- Exact behavior of the agent loop, tool execution, and streaming
- TUI rendering logic, ANSI handling, and keyboard parsing
- Session persistence format (as inspiration, not compatibility target)

**Approach**: Read TS files listed in the Reference Files table above, understand the logic, then implement idiomatic Go equivalents. Do not transliterate TypeScript line-by-line.

### 2. Session Persistence: CLEAN-SLATE JSONL

Design a new JSONL schema optimized for Go. No backwards compatibility with the TS version required.

#### JSONL Session Format

Each session is stored as `~/.pi-go/sessions/<session-id>.jsonl`. One JSON object per line. The file is append-only during a session; compaction rewrites it.

```jsonl
{"v":1,"type":"session_start","id":"uuid","ts":"RFC3339","model":"claude-sonnet-4-20250514","cwd":"/path"}
{"v":1,"type":"user","ts":"RFC3339","content":"explain this code","mentions":[{"path":"src/main.go","start":10,"end":20}]}
{"v":1,"type":"assistant","ts":"RFC3339","content":"This code does...","model":"claude-sonnet-4-20250514","usage":{"input":1234,"output":567},"stop_reason":"end_turn"}
{"v":1,"type":"tool_call","ts":"RFC3339","id":"tc_001","name":"read","args":{"path":"src/main.go","offset":0,"limit":100}}
{"v":1,"type":"tool_result","ts":"RFC3339","id":"tc_001","content":"package main...","duration_ms":12}
{"v":1,"type":"checkpoint","ts":"RFC3339","ref":"stash@{0}","tool":"edit","tool_args":"src/main.go"}
{"v":1,"type":"compaction","ts":"RFC3339","summary":"User asked about main.go...","kept_from":"RFC3339","messages_compacted":42}
{"v":1,"type":"branch","ts":"RFC3339","parent_id":"tc_001","summary":"Explored alternative approach..."}
{"v":1,"type":"session_end","ts":"RFC3339","total_cost":0.0234,"total_tokens":{"input":12345,"output":6789}}
```

#### Schema Types

```go
// internal/session/persistence.go

type RecordType string

const (
    RecordSessionStart RecordType = "session_start"
    RecordUser         RecordType = "user"
    RecordAssistant    RecordType = "assistant"
    RecordToolCall     RecordType = "tool_call"
    RecordToolResult   RecordType = "tool_result"
    RecordCheckpoint   RecordType = "checkpoint"
    RecordCompaction   RecordType = "compaction"
    RecordBranch       RecordType = "branch"
    RecordSessionEnd   RecordType = "session_end"
)

// Record is the envelope for all JSONL entries.
// The Version field enables forward-compatible schema evolution.
type Record struct {
    Version int        `json:"v"`
    Type    RecordType `json:"type"`
    // Remaining fields vary by type; decoded via json.RawMessage
}
```

**Design principles**:
- `"v":1` field on every record enables schema evolution without migration
- Append-only during active session; compaction rewrites the file
- `bufio.Scanner` for reading: line-by-line, never full load
- `os.File` with `O_APPEND|O_WRONLY` for writing: crash-safe appends
- Session listing: scan only `session_start` records (first line of each file)

### 3. Skill Format: SUPERSET of Claude Code

Read Claude Code's `~/.claude/skills/` YAML frontmatter format natively, but extend it with pi-go-specific fields. Existing Claude Code skills work with pi-go without modification.

#### Claude Code Compatible Fields (read as-is)

```yaml
---
name: skill-name
description: What this skill does. Triggers on "keyword1", "keyword2".
allowed-tools: Read, Write, Edit, Bash
---

# Skill content (markdown)
Instructions for the agent when this skill is activated...
```

#### pi-go Extension Fields

```yaml
---
name: skill-name
description: What this skill does. Triggers on "keyword1", "keyword2".
allowed-tools: Read, Write, Edit, Bash

# pi-go extensions (ignored by Claude Code)
pi-go:
  version: 1                          # Extension schema version
  priority: 10                        # Skill activation priority (lower = higher priority)
  mode: edit                          # Required mode: plan, edit, or any (default: any)
  file-patterns: ["*.go", "*.mod"]    # Auto-activate when these file patterns are touched
  pre-hook: "go vet ./..."            # Shell command to run before skill activation
  post-hook: "golangci-lint run"      # Shell command to run after skill completes
  provides-tools:                     # Custom tools this skill registers
    - name: go-test
      description: Run Go tests
      command: "go test -race -count=1 {{.Package}}"
---
```

#### Skill Resolution Order

1. Project-local: `.pi-go/skills/` in current directory (highest priority)
2. User-global: `~/.pi-go/skills/`
3. Claude Code compat: `~/.claude/skills/` (read-only; never written by pi-go)

#### Implementation

```go
// internal/prompt/skills.go

type Skill struct {
    Name         string   `yaml:"name"`
    Description  string   `yaml:"description"`
    AllowedTools []string `yaml:"allowed-tools"`
    Content      string   // Markdown body after frontmatter

    // pi-go extensions (nil if not present)
    PiGo *PiGoSkillExt `yaml:"pi-go,omitempty"`
}

type PiGoSkillExt struct {
    Version      int               `yaml:"version"`
    Priority     int               `yaml:"priority"`
    Mode         string            `yaml:"mode"`
    FilePatterns []string          `yaml:"file-patterns"`
    PreHook      string            `yaml:"pre-hook"`
    PostHook     string            `yaml:"post-hook"`
    ProvidesTools []SkillToolDef   `yaml:"provides-tools"`
}

type SkillToolDef struct {
    Name        string `yaml:"name"`
    Description string `yaml:"description"`
    Command     string `yaml:"command"`  // Go template with {{.Package}}, {{.File}}, etc.
}

// LoadSkills loads skills from all resolution paths, merging by name.
// Project-local overrides user-global overrides Claude Code compat.
func LoadSkills(projectDir, homeDir string) ([]Skill, error)
```

**Key decisions**:
- YAML parsing for frontmatter uses `gopkg.in/yaml.v3` (add to dependencies, Phase 4)
- Claude Code skills with no `pi-go:` block work unchanged
- `file-patterns` enables proactive skill activation (like CLAUDE.md's "MUST invoke /golang BEFORE Write on .go files")
- `provides-tools` enables skills to register custom agent tools without a plugin system

---

## All Questions Resolved

No open questions remain. Ready to begin Phase 1 implementation.
