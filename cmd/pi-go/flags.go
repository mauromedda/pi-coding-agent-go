// ABOUTME: CLI flag parsing using stdlib flag package
// ABOUTME: Supports --yolo, --model, --plan, --print, -p, --permission-mode, --allowedTools, --disallowedTools, SDK flags

package main

import "flag"

type cliArgs struct {
	yolo             bool
	model            string
	plan             bool
	print            bool
	prompt           string // -p "prompt" non-interactive mode
	thinking         bool
	version          bool
	update           bool
	baseURL          string
	maxTurns         int
	maxBudget        float64
	outputFormat     string
	inputFormat      string
	jsonSchema       string
	style            string
	permissionMode   string // --permission-mode
	allowedTools     string // --allowedTools (comma-separated)
	disallowedTools  string // --disallowedTools (comma-separated)
	lean             bool   // --lean minimal system prompt
	dangerouslySkip  bool   // --dangerously-skip-permissions
	verbose          bool   // -v / --verbose debug output
	noWorktree       bool   // --no-worktree disable session worktree
}

func parseFlags() cliArgs {
	var args cliArgs

	flag.BoolVar(&args.yolo, "yolo", false, "Skip all permission prompts")
	flag.StringVar(&args.model, "model", "", "Model to use (e.g., claude-sonnet-4-20250514)")
	flag.BoolVar(&args.plan, "plan", false, "Start in plan mode")
	flag.BoolVar(&args.print, "print", false, "Non-interactive print mode")
	flag.BoolVar(&args.thinking, "thinking", false, "Enable thinking/reasoning")
	flag.BoolVar(&args.version, "version", false, "Show version and exit")
	flag.BoolVar(&args.update, "update", false, "Self-update to latest version")
	flag.StringVar(&args.baseURL, "base-url", "", "Custom API base URL")
	flag.IntVar(&args.maxTurns, "max-turns", 0, "Maximum agent turns (0 = unlimited)")
	flag.Float64Var(&args.maxBudget, "max-budget-usd", 0.0, "Maximum budget in USD (0 = unlimited)")
	flag.StringVar(&args.outputFormat, "output-format", "text", "Output format: text, json, stream-json")
	flag.StringVar(&args.inputFormat, "input-format", "", "Input format: empty = plain text, stream-json = JSONL from stdin")
	flag.StringVar(&args.jsonSchema, "json-schema", "", "Path to JSON schema file for output validation")
	flag.StringVar(&args.style, "style", "", "Output style: concise, verbose, formal, casual")
	flag.StringVar(&args.prompt, "p", "", "Non-interactive mode: run prompt and exit")
	flag.StringVar(&args.permissionMode, "permission-mode", "", "Permission mode: default, acceptEdits, plan, dontAsk, bypassPermissions")
	flag.StringVar(&args.allowedTools, "allowedTools", "", "Comma-separated list of allowed tools")
	flag.StringVar(&args.disallowedTools, "disallowedTools", "", "Comma-separated list of disallowed tools")
	flag.BoolVar(&args.lean, "lean", false, "Use minimal system prompt (no memory, personality, context)")
	flag.BoolVar(&args.dangerouslySkip, "dangerously-skip-permissions", false, "Skip all permission checks (alias for bypassPermissions)")
	flag.BoolVar(&args.verbose, "v", false, "Enable verbose debug output")
	flag.BoolVar(&args.verbose, "verbose", false, "Enable verbose debug output")
	flag.BoolVar(&args.noWorktree, "no-worktree", false, "Disable session worktree isolation")

	flag.Parse()
	return args
}

// remaining returns the non-flag command-line arguments.
func (a cliArgs) remaining() []string {
	return flag.Args()
}
