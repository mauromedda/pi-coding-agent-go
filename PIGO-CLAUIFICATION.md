# PIGO-CLAUIFICATION: Claude Code Feature Gap Analysis

## Purpose

Extension to `PIGO-PORT.md`. Identifies features from [Claude Code](https://github.com/anthropics/claude-code) that pi-go should adopt to be a **lean, efficient, compatible** alternative. Every addition is evaluated against our principles: simplicity, performance, quality. No bloat.

**Philosophy**: Claude Code is the reference implementation; pi-go is the lean rewrite. We adopt the *protocols* and *interfaces* (MCP, hooks, memory hierarchy, permission rules), not the overhead (Electron, plugins marketplace, telemetry, agent teams). Where Claude Code adds 500 lines, we aim for 150.

---

## Gap Severity Legend

| Severity | Meaning |
|----------|---------|
| **P0** | Blocking: pi-go is fundamentally limited without this |
| **P1** | Critical: power users will hit this daily |
| **P2** | Important: needed for ecosystem compatibility |
| **P3** | Nice-to-have: adds polish, can defer |

---

## P0: Blocking Gaps

### 1. MCP (Model Context Protocol) Support

**What Claude Code does**: Full MCP client supporting HTTP, SSE, and stdio transports. Tools from MCP servers appear as native tools. OAuth 2.0 for remote servers. Dynamic tool discovery via `list_changed`. Can also *serve* as an MCP server (`claude mcp serve`).

**What PIGO-PORT.md has**: Nothing.

**Why P0**: MCP is the open standard for AI tool integration. Without it, pi-go cannot use GitHub Copilot tools, database connectors, Slack, Notion, or any third-party MCP server. It's the extension system we deferred; but Claude Code proved it's not optional.

**Lean implementation**:

```
internal/mcp/
├── client.go          # MCP client: capability negotiation, tool/resource listing
├── transport.go       # Transport interface
├── stdio.go           # Stdio transport: spawn process, JSON-RPC over stdin/stdout
├── http.go            # Streamable HTTP transport (replaces SSE)
├── server.go          # MCP server mode: expose pi-go tools to external consumers
├── config.go          # .mcp.json parsing, scope resolution (local/project/user)
└── oauth.go           # OAuth 2.0 for remote MCP servers (browser flow)
```

```go
// internal/mcp/client.go

type Client struct {
    transport Transport
    tools     []ai.Tool       // Discovered tools; refreshed on list_changed
    resources []Resource       // Discoverable resources
}

type Transport interface {
    Start(ctx context.Context) error
    Send(ctx context.Context, msg json.RawMessage) error
    Recv(ctx context.Context) (json.RawMessage, error)
    Close() error
}

func (c *Client) ListTools(ctx context.Context) ([]ai.Tool, error)
func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (ToolResult, error)
func (c *Client) ListResources(ctx context.Context) ([]Resource, error)
func (c *Client) ReadResource(ctx context.Context, uri string) ([]byte, error)
```

**Scope**: Three config scopes matching Claude Code:
- **Local**: `.claude/settings.local.json` `mcpServers` (gitignored, personal)
- **Project**: `.mcp.json` at project root (committable, shared)
- **User**: `~/.pi-go/settings.json` `mcpServers` (cross-project)

**Deferred**: Plugin marketplace, managed MCP, `MAX_MCP_OUTPUT_TOKENS` config. Add when demand exists.

**Phase**: 3b (parallel with Agent Core)

---

### 2. CLAUDE.md / Memory Hierarchy

**What Claude Code does**: Multi-level memory system loaded into system prompt. Supports imports (`@path/to/file`), path-specific rules (YAML frontmatter with glob `paths`), modular rules directory (`.claude/rules/`), auto-memory (LLM saves patterns/insights automatically).

**What PIGO-PORT.md has**: `internal/config/config.go` for settings; `internal/prompt/skills.go` for skills. No memory hierarchy; no CLAUDE.md-equivalent; no rules directory.

**Why P0**: CLAUDE.md is how users teach the agent about their project. Without it, pi-go requires re-explaining project conventions every session. Claude Code skills reference CLAUDE.md directly; ecosystem compatibility requires it.

**Lean implementation**:

```
internal/memory/
├── memory.go          # Memory loader: resolve, merge, inject into system prompt
├── rules.go           # Rules directory walker (.pi-go/rules/, .claude/rules/)
└── auto.go            # Auto-memory: LLM-triggered save of project patterns
```

**Memory resolution order** (highest to lowest priority):

| Level | Location | Shared? |
|-------|----------|---------|
| Project rules | `.pi-go/rules/*.md` | Team (git) |
| Project memory | `./PIGOMD.md` or `./.pi-go/PIGOMD.md` | Team (git) |
| Claude Code compat | `./CLAUDE.md` or `./.claude/CLAUDE.md` | Team (git) |
| Claude Code rules | `.claude/rules/*.md` | Team (git) |
| User memory | `~/.pi-go/PIGOMD.md` | Personal |
| User Claude compat | `~/.claude/CLAUDE.md` | Personal |
| Auto memory | `~/.pi-go/projects/<hash>/memory/` | Personal |
| Project local | `./PIGOMD.local.md` (gitignored) | Personal |

**Key design decisions**:
- Read `CLAUDE.md` natively (compatibility); prefer `PIGOMD.md` for pi-go-specific content
- `@path/to/file` import syntax (max depth 5; cycle detection via visited set)
- Path-specific rules via YAML frontmatter: `paths: ["src/**/*.go", "internal/**"]`
- Auto-memory capped at 200 lines in system prompt; topic files loaded on demand
- `/init` command bootstraps `PIGOMD.md` from project analysis
- `/memory` command opens memory files in `$EDITOR`

```go
// internal/memory/memory.go

type MemoryEntry struct {
    Source   string // File path
    Content  string // Resolved content (imports expanded)
    Priority int    // Lower = higher priority
    Paths    []string // Glob patterns for path-specific rules (empty = global)
}

// Load resolves all memory files from the hierarchy, expands imports,
// and returns entries sorted by priority.
func Load(projectDir, homeDir string) ([]MemoryEntry, error)

// InjectForFile filters memory entries relevant to a specific file path
// (matching path-specific rules) and formats them for the system prompt.
func InjectForFile(entries []MemoryEntry, filePath string) string
```

**Phase**: 4 (Interactive Agent); auto-memory in Phase 5

---

### 3. Sub-Agent / Task System

**What Claude Code does**: Spawns specialized sub-agents (Explore, Plan, Bash, General-purpose) with isolated context. Custom agents defined as Markdown files with YAML frontmatter. Background execution, worktree isolation, resumable, persistent memory.

**What PIGO-PORT.md has**: Single agent loop. No sub-agent spawning.

**Why P0**: Sub-agents are how Claude Code handles complex tasks without polluting the main context. The Explore agent keeps research out of the conversation; the Bash agent isolates long-running commands. Without this, pi-go's context window fills up faster, compaction triggers more often, and complex tasks become unreliable.

**Lean implementation**:

```
internal/agent/
├── agent.go           # (existing) Agent loop
├── subagent.go        # Sub-agent spawner: fork context, restrict tools, collect result
├── registry.go        # Built-in agent types + custom agent loader
└── agents/            # Built-in agent definitions
    ├── explore.md     # Read-only codebase exploration
    ├── plan.md        # Research for plan mode
    └── bash.md        # Isolated shell execution
```

```go
// internal/agent/subagent.go

type SubAgentConfig struct {
    Name            string
    Description     string
    Model           string   // Override model (e.g., "haiku" for Explore)
    Tools           []string // Allowed tools (nil = inherit)
    DisallowedTools []string
    MaxTurns        int
    Background      bool     // Run in background goroutine
    Isolation       string   // "worktree" for git worktree isolation
    Memory          string   // "user", "project", "local"
}

// Spawn creates a sub-agent with restricted tools and isolated context.
// Returns the agent's final response. If Background=true, returns immediately
// with a handle for later retrieval.
func Spawn(ctx context.Context, config SubAgentConfig, prompt string,
           session *session.Session) (*SubAgentHandle, error)

type SubAgentHandle struct {
    ID     string
    Done   <-chan struct{}
    Result func() (*ai.AssistantMessage, error)
}
```

**Custom agents**: Load from `.pi-go/agents/*.md` and `~/.pi-go/agents/*.md`. Claude Code compat: also read `.claude/agents/*.md`.

**Phase**: 3 (Agent Core)

---

### 4. OS-Level Sandboxing

**What Claude Code does**: macOS Seatbelt profiles, Linux bubblewrap, network domain filtering via proxy. Write access restricted to working directory. Commands can opt out via `sandbox.excludedCommands`.

**What PIGO-PORT.md has**: `filepath.Abs` + prefix check. Path traversal rejection. No OS-level isolation.

**Why P0**: Without sandboxing, a malicious LLM response could `rm -rf /` or exfiltrate data via `curl`. Path prefix checks are necessary but insufficient; they don't protect against shell escapes, symlink attacks, or network exfiltration.

**Lean implementation**:

```
internal/sandbox/
├── sandbox.go         # Sandbox interface + factory (detect OS, capabilities)
├── seatbelt.go        # macOS: sandbox-exec with custom profile
├── bwrap.go           # Linux: bubblewrap (bwrap) subprocess wrapping
├── noop.go            # Fallback: path-only checks (Windows, or when bwrap unavailable)
└── network.go         # Domain-based network filtering (optional proxy)
```

```go
// internal/sandbox/sandbox.go

type Sandbox interface {
    // WrapCommand modifies an exec.Cmd to run inside the sandbox.
    // Adds sandbox-exec (macOS) or bwrap (Linux) wrapper.
    WrapCommand(cmd *exec.Cmd, opts SandboxOpts) *exec.Cmd

    // ValidatePath checks if a path is accessible under current sandbox rules.
    ValidatePath(path string, write bool) error

    // Available returns true if OS-level sandboxing is supported.
    Available() bool
}

type SandboxOpts struct {
    WorkDir         string   // Primary writable directory
    AdditionalDirs  []string // Extra writable directories
    AllowNetwork    bool     // Allow network access
    AllowedDomains  []string // If AllowNetwork, restrict to these domains
}
```

**Key decisions**:
- Seatbelt on macOS (built-in, zero dependencies)
- bubblewrap on Linux (widely available; graceful degradation to path checks)
- Windows: path checks only (no good lightweight sandbox)
- Network filtering: optional; only when `sandbox.network.allowedDomains` is set
- `sandbox.excludedCommands`: commands like `git` that need wider access

**Phase**: 3 (Agent Core; wraps bash tool)

---

## P1: Critical Gaps

### 5. Web Tools (WebFetch + WebSearch)

**What Claude Code does**: `WebFetch` fetches URLs, converts HTML to markdown, processes with small model. `WebSearch` performs web searches with domain filtering.

**What PIGO-PORT.md has**: Nothing.

**Why P1**: Without web tools, the agent cannot look up documentation, check API references, or research solutions. Users must manually paste content.

**Lean implementation**:

```
internal/tools/
├── webfetch.go        # HTTP GET -> readability extraction -> markdown
└── websearch.go       # Search API wrapper (configurable backend)
```

```go
// internal/tools/webfetch.go

// WebFetch fetches a URL, extracts readable content, converts to markdown,
// and optionally processes it with a prompt using a fast model.
// Uses golang.org/x/net/html for parsing; no headless browser.
func WebFetch(ctx context.Context, url string, prompt string) (string, error)
```

**Key decisions**:
- HTML-to-markdown: use `golang.org/x/net/html` + custom readability extractor (no Chromium)
- Search backend: configurable; default to Brave Search API or SearXNG
- Cache: 15-minute TTL, in-memory LRU
- No Chrome integration (P3; see below)

**New dependency**: `golang.org/x/net/html` (stdlib-adjacent)

**Phase**: 3 (Tools)

---

### 6. Glob-Based Permission Rules

**What Claude Code does**: Permission rules with glob patterns and tool specifiers. Deny > Ask > Allow priority. Per-tool granularity: `Bash(npm run *)`, `Edit(/src/**)`, `WebFetch(domain:example.com)`.

**What PIGO-PORT.md has**: Three modes (Normal, Yolo, Plan). No glob-based rules; no per-tool granularity.

**Why P1**: Without granular permissions, users must choose between "approve everything" (yolo) and "approve every single action" (normal). Claude Code users have trained muscle memory for "Yes, don't ask again for this command" patterns.

**Lean implementation**:

```go
// internal/permission/permission.go (extends existing)

type Rule struct {
    Tool    string // "Bash", "Edit", "Read", "mcp__github__*"
    Pattern string // Glob: "npm run *", "/src/**", "domain:example.com"
    Action  Action // Allow, Deny, Ask
}

type Action int

const (
    ActionAllow Action = iota
    ActionDeny
    ActionAsk
)

// Evaluate checks rules in order: deny first, then ask, then allow.
// First match wins within each priority level.
func (c *Checker) Evaluate(tool string, specifier string) Action
```

**Settings integration**: Rules defined in `settings.json` under `allow`, `deny`, `ask` arrays:
```json
{
  "allow": ["Read", "Glob", "Grep", "Bash(go test *)"],
  "deny": ["Bash(rm -rf *)"],
  "ask": ["Edit", "Write", "Bash"]
}
```

**Phase**: 4 (Interactive Agent)

---

### 7. Headless / SDK Mode

**What Claude Code does**: `claude -p "query"` for non-interactive use. Output formats: text, json, stream-json. Structured output via `--json-schema`. Streaming input via `--input-format stream-json`. Budget/turn limits. System prompt override.

**What PIGO-PORT.md has**: `internal/mode/print/print.go` (placeholder). `--print` flag defined.

**Why P1**: SDK mode is how pi-go integrates into CI/CD, scripts, and automation. Without it, pi-go is interactive-only.

**Lean implementation**:

```go
// internal/mode/print/print.go

type PrintMode struct {
    outputFormat string // "text", "json", "stream-json"
    jsonSchema   string // Optional JSON Schema for structured output
    maxTurns     int
    maxBudgetUSD float64
    systemPrompt string // Override or append
    session      *session.Session
}

func (m *PrintMode) Run(ctx context.Context, prompt string) error
```

**CLI flags** (extend `cmd/pi-go/flags.go`):

| Flag | Purpose |
|------|---------|
| `--output-format` | text / json / stream-json |
| `--json-schema` | Structured output schema |
| `--max-turns` | Limit agentic turns |
| `--max-budget-usd` | Dollar limit |
| `--system-prompt` | Replace system prompt |
| `--append-system-prompt` | Append to system prompt |
| `--continue` / `-c` | Resume last session |
| `--resume` / `-r` | Resume specific session |

**Phase**: 4 (Interactive Agent)

---

### 8. Settings Precedence Chain

**What Claude Code does**: Five-level precedence: Managed > CLI > Local > Project > User. Managed settings are IT-deployed and cannot be overridden.

**What PIGO-PORT.md has**: Global + project config with deep merge.

**Why P1**: Enterprise adoption requires managed settings. Without precedence, organizations cannot enforce security policies.

**Lean implementation**:

```go
// internal/config/config.go (extends existing)

type SettingsLevel int

const (
    LevelUser     SettingsLevel = iota // ~/.pi-go/settings.json
    LevelProject                        // .pi-go/settings.json (committable)
    LevelLocal                          // .pi-go/settings.local.json (gitignored)
    LevelCLI                            // Command-line flags
    LevelManaged                        // /etc/pi-go/managed-settings.json (admin)
)

// Load merges settings from all levels. Higher levels override lower.
// Managed settings cannot be overridden by any other level.
func Load(projectDir string, flags *Flags) (*Settings, error)
```

**Managed settings paths**:
- macOS: `/Library/Application Support/pi-go/managed-settings.json`
- Linux: `/etc/pi-go/managed-settings.json`

**Phase**: 4 (Config)

---

## P2: Important Gaps (Ecosystem Compatibility)

### 9. Hooks System

**What Claude Code does**: 16 hook events with command, prompt, and agent types. Pre/post tool execution. Can block, modify input, or provide feedback. Async support.

**What PIGO-PORT.md has**: Nothing.

**Why P2**: Hooks are how power users customize behavior without modifying source. CI/CD gates, code quality checks, custom notifications. Claude Code's hook ecosystem is growing; compatibility matters.

**Lean implementation** (start with 6 core events, not 16):

```
internal/hooks/
├── hooks.go           # Hook engine: load, match, execute
├── command.go         # Shell command hooks (JSON on stdin)
└── types.go           # Event types, matcher patterns
```

**Core events (Phase 4)**:

| Event | Can Block? | Use Case |
|-------|-----------|----------|
| `PreToolUse` | Yes | Lint before write; block dangerous commands |
| `PostToolUse` | No | Log tool results; trigger side effects |
| `UserPromptSubmit` | Yes | Validate prompt; inject context |
| `Stop` | Yes | Force continue; quality gate |
| `SessionStart` | No | Set environment; load context |
| `SessionEnd` | No | Cleanup; metrics |

**Deferred events**: `Notification`, `SubagentStart/Stop`, `TeammateIdle`, `TaskCompleted`, `ConfigChange`, `PreCompact`, `PermissionRequest`, `PostToolUseFailure`. Add when demand exists.

**Hook config** (in `settings.json`):
```json
{
  "hooks": {
    "PreToolUse": [{
      "matcher": "Edit|Write",
      "type": "command",
      "command": "golangci-lint run --new-from-rev=HEAD"
    }]
  }
}
```

**No prompt or agent hook types initially**. Command hooks cover 90% of use cases.

**Phase**: 4 (Interactive Agent)

---

### 10. Slash Commands System

**What Claude Code does**: 35+ built-in slash commands. Custom skills become slash commands. Interactive pickers for `/model`, `/resume`, `/mcp`.

**What PIGO-PORT.md has**: No slash command system.

**Why P2**: Slash commands are the primary UX for configuration and session management. Claude Code users expect them.

**Lean implementation**:

```go
// internal/commands/commands.go

type Command struct {
    Name        string
    Aliases     []string
    Description string
    Execute     func(ctx context.Context, args string, app *App) error
}

var builtinCommands = []Command{
    {Name: "clear",    Description: "Clear conversation",          Execute: cmdClear},
    {Name: "compact",  Description: "Compact context",             Execute: cmdCompact},
    {Name: "config",   Description: "Open settings",               Execute: cmdConfig},
    {Name: "context",  Description: "Show context usage",          Execute: cmdContext},
    {Name: "cost",     Description: "Token usage stats",           Execute: cmdCost},
    {Name: "help",     Description: "Show help",                   Execute: cmdHelp},
    {Name: "init",     Description: "Bootstrap PIGOMD.md",         Execute: cmdInit},
    {Name: "memory",   Description: "Edit memory files",           Execute: cmdMemory},
    {Name: "model",    Description: "Switch model",                Execute: cmdModel},
    {Name: "plan",     Description: "Enter plan mode",             Execute: cmdPlan},
    {Name: "rename",   Description: "Rename session",              Execute: cmdRename},
    {Name: "resume",   Description: "Resume session",              Execute: cmdResume},
    {Name: "rewind",   Description: "Rewind to checkpoint",        Execute: cmdRewind},
    {Name: "sandbox",  Description: "Configure sandbox",           Execute: cmdSandbox},
    {Name: "status",   Description: "Version and account info",    Execute: cmdStatus},
    {Name: "vim",      Description: "Toggle vim mode",             Execute: cmdVim},
    {Name: "mcp",      Description: "Manage MCP servers",          Execute: cmdMCP},
    {Name: "export",   Description: "Export conversation",         Execute: cmdExport},
    {Name: "copy",     Description: "Copy last response",          Execute: cmdCopy},
}
```

**Custom skills as commands**: Any skill at `.pi-go/skills/<name>/SKILL.md` registers as `/<name>`.

**Phase**: 4 (Interactive Agent)

---

### 11. Git Worktree Isolation

**What Claude Code does**: `--worktree` / `-w` flag creates isolated git worktree at `.claude/worktrees/<name>`. Sub-agents can use `isolation: worktree`. Cleanup on session end.

**What PIGO-PORT.md has**: Nothing.

**Why P2**: Worktrees enable parallel sessions on the same repo without conflicts. Sub-agents with worktree isolation can make changes without polluting the main tree.

**Lean implementation**:

```go
// internal/git/worktree.go

// Create creates a new git worktree at .pi-go/worktrees/<name>
// with a new branch based on HEAD.
func Create(repoDir, name string) (worktreeDir string, branch string, err error)

// Remove cleans up a worktree and its branch.
func Remove(repoDir, name string) error

// List returns active worktrees.
func List(repoDir string) ([]WorktreeInfo, error)
```

**Phase**: 5 (Polish)

---

### 12. Notebook Support

**What Claude Code does**: `NotebookEdit` tool for Jupyter `.ipynb` files. Cell-level editing (replace, insert, delete). Reads notebooks with full output rendering.

**What PIGO-PORT.md has**: Nothing.

**Why P2**: Data science workflows rely on notebooks. Without support, pi-go cannot help with ML/AI development.

**Lean implementation**:

```go
// internal/tools/notebook.go

// NotebookEdit modifies a specific cell in a Jupyter notebook.
// Supports replace, insert, delete operations.
func NotebookEdit(ctx context.Context, path string, cellNumber int,
                  editMode string, newSource string, cellType string) error

// NotebookRead reads a notebook and returns a formatted representation
// with all cells, their types, and outputs.
func NotebookRead(path string) (string, error)
```

**Key decision**: Parse `.ipynb` JSON directly with `encoding/json`. No Jupyter dependency.

**Phase**: 5 (Polish)

---

### 13. Session Forking and PR Linking

**What Claude Code does**: `--fork-session` branches a conversation. `--from-pr 123` resumes sessions linked to a PR.

**What PIGO-PORT.md has**: Session persistence but no forking or PR linking.

**Why P2**: Forking enables experimentation without losing the original thread. PR linking enables multi-session workflows on the same feature.

**Lean implementation**:

```go
// internal/session/persistence.go (extends existing)

// Fork creates a copy of the session up to the current point,
// with a new session ID.
func (s *Session) Fork() (*Session, error)

// LinkPR associates the session with a PR number.
func (s *Session) LinkPR(prNumber int, repo string) error

// FindByPR returns sessions linked to a specific PR.
func FindByPR(prNumber int, repo string) ([]SessionInfo, error)
```

**Phase**: 5 (Polish)

---

## P3: Nice-to-Have (Defer Unless Trivial)

### 14. Vim Mode

**What Claude Code does**: `/vim` toggles vim-style editing in the input buffer.

**Why P3**: Power user feature. The input component (`pkg/tui/component/input.go`) can support this later via a mode flag on the key handler. Estimate: ~200 LOC.

**Phase**: 5+

---

### 15. Chrome / Browser Integration

**What Claude Code does**: `@browser` context, `--chrome` flag, screenshot capture, console logs, network traces.

**Why P3**: Valuable for web development but adds significant complexity (Chromium DevTools Protocol). Use MCP server for browser automation instead (`mcp__puppeteer`).

**Decision**: Recommend MCP-based browser tools rather than native integration. Zero code.

---

### 16. Agent Teams

**What Claude Code does**: Multiple Claude instances coordinated by a lead. Shared task list. Experimental feature behind flag.

**Why P3**: Experimental even in Claude Code. High complexity, marginal benefit for solo developers. Revisit when MCP-based coordination matures.

**Decision**: Defer entirely. Sub-agents (P0 #3) cover 90% of the use case.

---

### 17. Plugin System

**What Claude Code does**: Structured plugins with agents, skills, hooks, MCP servers. Marketplace with GitHub/npm sources.

**Why P3**: MCP + hooks + skills cover the same ground with less abstraction. Plugins are a convenience wrapper.

**Decision**: Defer. The combination of MCP servers + hooks + skills + custom agents provides equivalent extensibility without a plugin abstraction layer.

---

### 18. Auto-Updates

**What Claude Code does**: Self-update via `autoUpdatesChannel` setting.

**Why P3**: Can be added trivially later. Check GitHub Releases API, download binary, replace self.

**Phase**: 5 (already in PIGO-PORT.md as `cmd/pi-go/update.go`)

---

### 19. Telemetry / OpenTelemetry

**What Claude Code does**: Statsig, Sentry, OpenTelemetry export.

**Why P3**: Not needed for MVP. Add opt-in OpenTelemetry when enterprise customers request it.

**Decision**: Defer. Add `OTEL_METRICS_EXPORTER` support in Phase 5+ if needed.

---

### 20. Output Styles

**What Claude Code does**: Configurable `outputStyle` setting (explanatory mode, learning mode).

**Why P3**: System prompt variations. Trivial to add: a `--style` flag that appends instructions to the system prompt. ~20 LOC.

**Phase**: 5+

---

### 21. Prompt Suggestions

**What Claude Code does**: AI-generated follow-up suggestions after each response.

**Why P3**: Nice UX touch. Adds one extra LLM call (using fast model). ~50 LOC.

**Phase**: 5+

---

## Revised Architecture (Additions to PIGO-PORT.md)

### New Packages

```
pi-coding-agent-go/
├── internal/
│   ├── mcp/                    # NEW: MCP client + server
│   │   ├── client.go
│   │   ├── transport.go
│   │   ├── stdio.go
│   │   ├── http.go
│   │   ├── server.go
│   │   ├── config.go
│   │   └── oauth.go
│   ├── memory/                 # NEW: Memory hierarchy (CLAUDE.md compat)
│   │   ├── memory.go
│   │   ├── rules.go
│   │   └── auto.go
│   ├── sandbox/                # NEW: OS-level sandboxing
│   │   ├── sandbox.go
│   │   ├── seatbelt.go
│   │   ├── bwrap.go
│   │   ├── noop.go
│   │   └── network.go
│   ├── hooks/                  # NEW: Lifecycle hooks
│   │   ├── hooks.go
│   │   ├── command.go
│   │   └── types.go
│   ├── commands/               # NEW: Slash commands
│   │   └── commands.go
│   ├── git/                    # NEW: Git operations (worktrees, PR linking)
│   │   └── worktree.go
│   ├── agent/
│   │   ├── agent.go
│   │   ├── subagent.go         # NEW: Sub-agent spawning
│   │   ├── registry.go         # NEW: Agent type registry
│   │   ├── types.go
│   │   └── tool.go
│   ├── tools/
│   │   ├── ... (existing)
│   │   ├── webfetch.go         # NEW
│   │   ├── websearch.go        # NEW
│   │   └── notebook.go         # NEW
│   └── ... (existing packages)
```

### New Dependencies

| Dependency | Purpose | Phase |
|------------|---------|-------|
| `golang.org/x/net/html` | HTML parsing for WebFetch | 3 |
| `github.com/gorilla/websocket` | MCP WebSocket transport (if needed) | 3b |

**Total new deps**: 1-2. We reuse stdlib and existing deps aggressively.

---

## Revised Phase Plan

### Phase 3b: MCP Foundation (parallel with Phase 3)

1. `internal/mcp/transport.go` — Transport interface
2. `internal/mcp/stdio.go` — Stdio transport (spawn process, JSON-RPC)
3. `internal/mcp/client.go` — MCP client (capability negotiation, tool listing)
4. `internal/mcp/config.go` — `.mcp.json` parsing, scope resolution
5. `internal/mcp/http.go` — Streamable HTTP transport
6. `internal/mcp/server.go` — Expose pi-go tools as MCP server
7. `internal/mcp/oauth.go` — OAuth 2.0 browser flow for remote servers

### Phase 3 Additions (Agent Core)

8. `internal/agent/subagent.go` — Sub-agent spawning with isolated context
9. `internal/agent/registry.go` — Built-in + custom agent type registry
10. `internal/sandbox/sandbox.go` — Sandbox interface + OS detection
11. `internal/sandbox/seatbelt.go` — macOS sandbox-exec profiles
12. `internal/sandbox/bwrap.go` — Linux bubblewrap wrapping
13. `internal/tools/webfetch.go` — URL fetch + HTML-to-markdown
14. `internal/tools/websearch.go` — Web search API wrapper

### Phase 4 Additions (Interactive Agent)

15. `internal/memory/memory.go` — Memory hierarchy loader
16. `internal/memory/rules.go` — Rules directory walker
17. `internal/hooks/hooks.go` — Hook engine (6 core events)
18. `internal/commands/commands.go` — Slash command registry
19. Permission rules with glob patterns (extend `internal/permission/`)
20. Settings precedence chain (extend `internal/config/`)
21. SDK/print mode with JSON output (extend `internal/mode/print/`)

### Phase 5 Additions (Polish)

22. `internal/memory/auto.go` — Auto-memory
23. `internal/git/worktree.go` — Git worktree isolation
24. `internal/tools/notebook.go` — Jupyter notebook support
25. Session forking + PR linking
26. Vim mode
27. Prompt suggestions

---

## CLI Flags Additions

Flags to add to `cmd/pi-go/flags.go`:

```go
// SDK mode
flag.StringVar(&f.OutputFormat, "output-format", "text", "Output: text, json, stream-json")
flag.StringVar(&f.JSONSchema, "json-schema", "", "Structured output JSON Schema")
flag.IntVar(&f.MaxTurns, "max-turns", 0, "Max agentic turns (0=unlimited)")
flag.Float64Var(&f.MaxBudgetUSD, "max-budget-usd", 0, "Max spend in USD")
flag.StringVar(&f.SystemPrompt, "system-prompt", "", "Override system prompt")
flag.StringVar(&f.AppendSystemPrompt, "append-system-prompt", "", "Append to system prompt")

// Session management
flag.BoolVar(&f.Continue, "continue", false, "Resume last session")
flag.BoolVar(&f.Continue, "c", false, "Resume last session (short)")
flag.StringVar(&f.Resume, "resume", "", "Resume specific session")
flag.StringVar(&f.Resume, "r", "", "Resume specific session (short)")
flag.BoolVar(&f.ForkSession, "fork-session", false, "Fork session on resume")

// Working context
flag.BoolVar(&f.Worktree, "worktree", false, "Start in git worktree")
flag.BoolVar(&f.Worktree, "w", false, "Start in git worktree (short)")
flag.StringVar(&f.AddDir, "add-dir", "", "Additional working directory")

// MCP
flag.StringVar(&f.MCPConfig, "mcp-config", "", "MCP server config JSON file")

// Permissions
flag.StringVar(&f.PermissionMode, "permission-mode", "", "Permission mode")
flag.StringVar(&f.AllowedTools, "allowedTools", "", "Auto-approve tools (comma-sep)")
flag.StringVar(&f.DisallowedTools, "disallowedTools", "", "Remove tools (comma-sep)")

// Debug
flag.BoolVar(&f.Debug, "debug", false, "Debug mode")
flag.BoolVar(&f.Verbose, "verbose", false, "Verbose output")
```

---

## Keyboard Shortcut Additions

| Shortcut | Action | Source |
|----------|--------|--------|
| `Ctrl+B` | Background running task | Claude Code |
| `Ctrl+T` | Toggle task list | Claude Code |
| `Alt+P` | Switch model | Claude Code |
| `Alt+T` | Toggle extended thinking | Claude Code |
| `!command` | Direct bash execution | Claude Code |
| `@file` | File path autocomplete | Claude Code (extends existing @file#L-L) |

---

## What We Deliberately Skip

These Claude Code features do NOT belong in pi-go:

| Feature | Why Skip |
|---------|----------|
| **Plugin marketplace** | MCP + hooks + skills = equivalent. No registry overhead. |
| **Agent teams** | Experimental; sub-agents cover 90% of use cases |
| **Desktop app** | pi-go is terminal-first. IDE extensions are separate repos. |
| **Slack integration** | MCP server; not a core feature |
| **Chrome integration** | MCP server (`puppeteer`); not built-in |
| **Statsig telemetry** | No tracking. Opt-in OTEL only if enterprise demands. |
| **Sentry error reporting** | Crash recovery + local logs. No cloud dependency. |
| **Web sessions** | Terminal-first. No cloud infrastructure. |
| **Managed MCP** | Enterprise complexity. Add when first enterprise customer needs it. |
| **Status line customization** | Simple default. No JSON template engine. |
| **Theme system** | Single dark-on-terminal theme. Respect terminal colors. |
| **Survey system** | `/bug` command with local export instead. |
| **Keybindings JSON** | Hardcoded defaults with override flag. Add file-based config in 5+. |

---

## Summary: Effort Estimate

| Priority | Items | New LOC (est.) | New Files |
|----------|-------|----------------|-----------|
| **P0** | MCP, Memory, Sub-agents, Sandbox | ~3,500 | ~20 |
| **P1** | Web tools, Permissions, SDK mode, Settings | ~2,000 | ~8 |
| **P2** | Hooks, Commands, Worktrees, Notebooks, Forking | ~1,500 | ~10 |
| **P3** | Vim, Suggestions, Output styles, OTEL | ~500 | ~5 |
| **Total** | | **~7,500** | **~43** |

For context: Claude Code (TypeScript) is estimated at 50,000+ LOC. pi-go targets the same capability surface in ~10,000 LOC total (existing ~3,100 + ~7,500 new). That's the lean advantage of Go + intentional scope control.

---

## Compatibility Matrix

| Claude Code Feature | pi-go Status | Compatibility |
|---------------------|-------------|---------------|
| CLAUDE.md files | Read natively | Full |
| .claude/rules/ | Read natively | Full |
| .claude/agents/ | Read natively | Full |
| .claude/settings.json | Read natively | Full |
| .mcp.json | Read natively | Full |
| Skills (SKILL.md) | Read natively | Full (superset) |
| Hooks (settings.json) | Command type only | Partial (no prompt/agent types initially) |
| Permission rules | Glob syntax | Full |
| MCP stdio transport | Supported | Full |
| MCP HTTP transport | Supported | Full |
| Session format | Own JSONL format | Not compatible (by design) |
| Plugin system | Not supported | None (use MCP + hooks + skills instead) |
| Agent teams | Not supported | None |

**Goal**: A user with an existing `.claude/` directory can run `pi-go` and have their memory, rules, agents, MCP servers, and skills work immediately. Session history is separate (different format, different storage).
