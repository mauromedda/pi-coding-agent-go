// ABOUTME: CLI entry point for pi-go with terminal crash recovery
// ABOUTME: Parses flags, loads config, registers providers/tools, dispatches to mode

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/internal/config"
	"github.com/mauromedda/pi-coding-agent-go/internal/intent"
	pilog "github.com/mauromedda/pi-coding-agent-go/internal/log"
	"github.com/mauromedda/pi-coding-agent-go/internal/memory"
	"github.com/mauromedda/pi-coding-agent-go/internal/mode/interactive/btea"
	"github.com/mauromedda/pi-coding-agent-go/internal/mode/print"
	"github.com/mauromedda/pi-coding-agent-go/internal/permission"
	"github.com/mauromedda/pi-coding-agent-go/internal/personality"
	"github.com/mauromedda/pi-coding-agent-go/internal/personality/checks"
	"github.com/mauromedda/pi-coding-agent-go/internal/pkgmanager"
	"github.com/mauromedda/pi-coding-agent-go/internal/prompt"
	"github.com/mauromedda/pi-coding-agent-go/internal/statusline"
	"github.com/mauromedda/pi-coding-agent-go/internal/telemetry"
	"github.com/mauromedda/pi-coding-agent-go/internal/tools"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai/provider/anthropic"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai/provider/google"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai/provider/openai"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai/provider/vertex"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/theme"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	// Intercept package subcommands before flag parsing.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "install", "remove", "update", "list":
			cwd, _ := os.Getwd()
			if err := pkgmanager.RunCLI(os.Args[1:], config.PackagesDir(), config.PackagesDirLocal(cwd)); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		}
	}

	args := parseFlags()

	if args.version {
		fmt.Printf("pi-go %s (%s) built %s\n", version, commit, date)
		os.Exit(0)
	}

	if args.update {
		if err := runSelfUpdate(version); err != nil {
			fmt.Fprintf(os.Stderr, "update failed: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if err := run(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// run performs the full initialization sequence and dispatches to the selected mode.
func run(args cliArgs) error {
	if args.verbose {
		pilog.SetLevel(pilog.LevelDebug)
	}

	registerProviders()

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	home, _ := os.UserHomeDir()

	// Load auth first so picompat keys are available before provider registration
	auth, err := config.LoadAuth()
	if err != nil {
		return fmt.Errorf("loading auth: %w", err)
	}

	// Merge API keys from ~/.pi/agent/ (does not overwrite existing keys)
	config.MergePiAuth(auth, config.PiAgentDir())

	// Load config with CLI overrides; picompat is Level -1 inside LoadAll
	cfg, err := config.LoadAll(cwd, buildCLIOverrides(args))
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Resolve and activate theme from config
	resolveTheme(cfg, cwd)

	model, err := resolveModel(args, cfg)
	if err != nil {
		return fmt.Errorf("resolving model: %w", err)
	}

	baseURL := resolveBaseURL(args, cfg)

	// W4: Register providers with auth keys
	registerProvidersWithAuth(auth, baseURL)

	provider := ai.GetProvider(model.Api, baseURL)
	if provider == nil {
		return fmt.Errorf("no provider registered for API %q", model.Api)
	}

	pathSandbox, err := permission.NewSandbox([]string{cwd})
	if err != nil {
		return fmt.Errorf("creating path sandbox: %w", err)
	}

	// W1/W3: Registry with sandbox registers all builtins including web tools
	toolRegistry := tools.NewRegistryWithSandbox(pathSandbox)

	// Apply --disallowedTools: remove tools before creating checker
	if args.disallowedTools != "" {
		for spec := range strings.SplitSeq(args.disallowedTools, ",") {
			spec = strings.TrimSpace(spec)
			if spec != "" {
				toolRegistry.Remove(spec)
			}
		}
	}

	// W7: Create checker from settings with glob rules using effective permissions
	permMode := resolvePermissionMode(args, cfg)
	allow, deny, ask := cfg.EffectivePermissions()
	checker := permission.NewCheckerFromSettings(permMode, nil, allow, deny, ask)

	// Apply --allowedTools: add as glob allow rules
	if args.allowedTools != "" {
		for spec := range strings.SplitSeq(args.allowedTools, ",") {
			spec = strings.TrimSpace(spec)
			if spec != "" {
				checker.AddAllowRule(permission.Rule{Tool: spec})
			}
		}
	}

	// W2: Load memory hierarchy and format for system prompt
	memEntries, _ := memory.Load(cwd, home)
	memSection := memory.FormatForPrompt(memEntries, nil)

	// Initialize telemetry tracker
	var tracker *telemetry.Tracker
	if cfg.Telemetry.IsEnabled() {
		tracker = telemetry.NewTracker(cfg.Telemetry.BudgetUSD, cfg.Telemetry.EffectiveWarnAtPct())
	}
	_ = tracker // Will be wired into agent loop in a future phase

	// Initialize personality engine
	var personalityPrompt string
	if cfg.Personality != nil {
		engine, err := personality.NewEngine("")
		if err == nil {
			if err := engine.SetProfile(cfg.Personality.EffectiveProfile()); err != nil {
				pilog.Debug("personality: profile %q not found, using base", cfg.Personality.EffectiveProfile())
			}
			ctx := checks.CheckContext{} // Empty context; populated per-request later
			personalityPrompt = engine.ComposePrompt(ctx)
		}
	}

	// Initialize intent classifier (for future use; not wired into agent loop yet)
	var intentClassifier *intent.Classifier
	if cfg.Intent.IsEnabled() {
		intentClassifier = intent.NewClassifier(intent.ClassifierConfig{
			HeuristicThreshold: cfg.Intent.EffectiveHeuristicThreshold(),
			AutoPlanFileCount:  cfg.Intent.EffectiveAutoPlanFileCount(),
		})
	}
	_ = intentClassifier // Will be wired into agent loop in a future phase

	// Build system prompt with memory and tool names
	toolNames := make([]string, 0, len(toolRegistry.All()))
	for _, t := range toolRegistry.All() {
		toolNames = append(toolNames, t.Name)
	}
	systemPrompt := prompt.BuildSystem(prompt.SystemOpts{
		CWD:               cwd,
		PlanMode:           args.plan,
		ToolNames:          toolNames,
		MemorySection:      memSection,
		ContextFiles:       prompt.LoadContextFiles(cwd),
		Style:              args.style,
		PersonalityPrompt:  personalityPrompt,
		PromptVersion:      promptVersion(cfg),
	})

	// -p "prompt" shorthand: non-interactive mode with inline prompt
	if args.prompt != "" {
		return print.RunWithConfig(context.Background(), print.Config{
			OutputFormat: args.outputFormat,
			MaxTurns:     args.maxTurns,
			MaxBudgetUSD: args.maxBudget,
			SystemPrompt: systemPrompt,
			InputFormat:  args.inputFormat,
			JSONSchema:   args.jsonSchema,
		}, print.Deps{
			Provider: provider,
			Model:    model,
			Tools:    toolRegistry.All(),
		}, args.prompt)
	}

	// Print mode: non-interactive, streams to stdout
	if args.print {
		outputFormat := args.outputFormat
		if outputFormat == "" {
			outputFormat = "text"
		}

		promptText := strings.Join(args.remaining(), " ")
		return print.RunWithConfig(context.Background(), print.Config{
			OutputFormat: outputFormat,
			MaxTurns:     args.maxTurns,
			MaxBudgetUSD: args.maxBudget,
			SystemPrompt: systemPrompt,
			InputFormat:  args.inputFormat,
			JSONSchema:   args.jsonSchema,
		}, print.Deps{
			Provider: provider,
			Model:    model,
			Tools:    toolRegistry.All(),
		}, promptText)
	}

	// Build optional status line engine from config
	var statusEngine *statusline.Engine
	if cfg.StatusLine != nil && cfg.StatusLine.Command != "" {
		statusEngine = statusline.New(cfg.StatusLine.Command, cfg.StatusLine.Padding)
	}

	// Interactive mode (default)
	return runInteractive(model, checker, provider, toolRegistry, systemPrompt, statusEngine, cfg.AutoCompactThreshold)
}

// registerProviders registers all built-in AI provider factories with no auth.
func registerProviders() {
	ai.RegisterProvider(ai.ApiAnthropic, func(baseURL string) ai.ApiProvider {
		return anthropic.New("", baseURL)
	})
	ai.RegisterProvider(ai.ApiOpenAI, func(baseURL string) ai.ApiProvider {
		return openai.New("", baseURL)
	})
	ai.RegisterProvider(ai.ApiGoogle, func(baseURL string) ai.ApiProvider {
		return google.New("", baseURL)
	})
	ai.RegisterProvider(ai.ApiVertex, func(baseURL string) ai.ApiProvider {
		return vertex.New("", "", baseURL)
	})
}

// registerProvidersWithAuth re-registers providers with auth keys from the store.
func registerProvidersWithAuth(auth *config.AuthStore, _ string) {
	if key := auth.GetKey("anthropic"); key != "" {
		ai.RegisterProvider(ai.ApiAnthropic, func(baseURL string) ai.ApiProvider {
			return anthropic.New(key, baseURL)
		})
	}

	// OpenAI-compatible: also check vllm and ollama keys
	openaiKey := auth.GetKey("openai")
	if openaiKey == "" {
		openaiKey = auth.GetKey("vllm")
	}
	if openaiKey == "" {
		openaiKey = auth.GetKey("ollama")
	}
	if openaiKey != "" {
		ai.RegisterProvider(ai.ApiOpenAI, func(baseURL string) ai.ApiProvider {
			return openai.New(openaiKey, baseURL)
		})
	}

	if key := auth.GetKey("google"); key != "" {
		ai.RegisterProvider(ai.ApiGoogle, func(baseURL string) ai.ApiProvider {
			return google.New(key, baseURL)
		})
	}
}

// buildCLIOverrides maps CLI flags to a Settings struct for LoadAll.
func buildCLIOverrides(args cliArgs) *config.Settings {
	s := &config.Settings{}
	if args.model != "" {
		s.Model = args.model
	}
	if args.baseURL != "" {
		s.BaseURL = args.baseURL
	}
	if args.thinking {
		s.Thinking = true
	}
	if args.yolo {
		s.Yolo = true
	}
	return s
}

// resolveModel determines the model from CLI flag, config, or default.
func resolveModel(args cliArgs, cfg *config.Settings) (*ai.Model, error) {
	modelID := args.model
	if modelID == "" {
		modelID = cfg.Model
	}
	return config.ResolveModel(modelID)
}

// resolveBaseURL picks the API base URL from CLI flag or config.
func resolveBaseURL(args cliArgs, cfg *config.Settings) string {
	if args.baseURL != "" {
		return args.baseURL
	}
	return cfg.BaseURL
}

// resolvePermissionMode maps CLI flags and config to a permission.Mode.
// Priority: --dangerously-skip-permissions > --permission-mode > --yolo/--plan > config > normal.
func resolvePermissionMode(args cliArgs, cfg *config.Settings) permission.Mode {
	if args.dangerouslySkip {
		return permission.ModeYolo
	}
	if args.permissionMode != "" {
		if mode, err := permission.ParseMode(args.permissionMode); err == nil {
			return mode
		}
	}
	switch {
	case args.yolo || cfg.Yolo:
		return permission.ModeYolo
	case args.plan:
		return permission.ModePlan
	}
	if cfgMode := cfg.EffectiveDefaultMode(); cfgMode != "" {
		if mode, err := permission.ParseMode(cfgMode); err == nil {
			return mode
		}
	}
	return permission.ModeNormal
}

// runInteractive starts the Bubble Tea interactive TUI.
func runInteractive(model *ai.Model, checker *permission.Checker, provider ai.ApiProvider, toolReg *tools.Registry, systemPrompt string, statusEngine *statusline.Engine, autoCompactThreshold int) error {
	return btea.Run(btea.AppDeps{
		Provider:             provider,
		Model:                model,
		Tools:                toolReg.All(),
		Checker:              checker,
		SystemPrompt:         systemPrompt,
		Version:              version,
		StatusEngine:         statusEngine,
		AutoCompactThreshold: autoCompactThreshold,
		PermissionMode:       checker.Mode(),
	})
}

// promptVersion returns the active prompt version from config, or empty for hardcoded fallback.
func promptVersion(cfg *config.Settings) string {
	if cfg.Prompts != nil && cfg.Prompts.ActiveVersion != "" {
		return cfg.Prompts.ActiveVersion
	}
	return ""
}

// resolveTheme loads the theme from config. It checks:
// 1. Built-in theme names (default, dark, light, monochrome)
// 2. JSON file in theme directories
// Falls back to "default" if not set or not found.
func resolveTheme(cfg *config.Settings, cwd string) {
	name := cfg.Theme
	if name == "" {
		return // already initialized to default
	}

	// Try built-in first
	if th := theme.Builtin(name); th != nil {
		theme.Set(th)
		return
	}

	// Try loading from theme directories
	for _, dir := range config.ThemesDirs(cwd) {
		path := filepath.Join(dir, name+".json")
		if th, err := theme.LoadFile(path); err == nil {
			theme.Set(th)
			return
		}
	}

	// Unknown theme; keep default
	fmt.Fprintf(os.Stderr, "warning: unknown theme %q, using default\n", name)
}
