// ABOUTME: CLI entry point for pi-go with terminal crash recovery
// ABOUTME: Parses flags, loads config, registers providers/tools, dispatches to mode

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/internal/config"
	"github.com/mauromedda/pi-coding-agent-go/internal/memory"
	"github.com/mauromedda/pi-coding-agent-go/internal/mode/interactive"
	"github.com/mauromedda/pi-coding-agent-go/internal/mode/print"
	"github.com/mauromedda/pi-coding-agent-go/internal/permission"
	"github.com/mauromedda/pi-coding-agent-go/internal/prompt"
	"github.com/mauromedda/pi-coding-agent-go/internal/tools"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai/provider/anthropic"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai/provider/google"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai/provider/openai"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai/provider/vertex"
	"github.com/mauromedda/pi-coding-agent-go/pkg/tui/terminal"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
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

	// W7: Create checker from settings with glob rules
	permMode := resolvePermissionMode(args, cfg)
	checker := permission.NewCheckerFromSettings(permMode, nil, cfg.Allow, cfg.Deny, cfg.Ask)

	// W2: Load memory hierarchy and format for system prompt
	memEntries, _ := memory.Load(cwd, home)
	memSection := memory.FormatForPrompt(memEntries, nil)

	// Build system prompt with memory and tool names
	toolNames := make([]string, 0, len(toolRegistry.All()))
	for _, t := range toolRegistry.All() {
		toolNames = append(toolNames, t.Name)
	}
	systemPrompt := prompt.BuildSystem(prompt.SystemOpts{
		CWD:           cwd,
		PlanMode:      args.plan,
		ToolNames:     toolNames,
		MemorySection: memSection,
		ContextFiles:  prompt.LoadContextFiles(cwd),
	})

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

	// Interactive mode (default)
	return runInteractive(model, checker, provider, toolRegistry, systemPrompt)
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
func resolvePermissionMode(args cliArgs, cfg *config.Settings) permission.Mode {
	switch {
	case args.yolo || cfg.Yolo:
		return permission.ModeYolo
	case args.plan:
		return permission.ModePlan
	default:
		return permission.ModeNormal
	}
}

// runInteractive sets up the real terminal, defers crash recovery, and runs the TUI.
func runInteractive(model *ai.Model, checker *permission.Checker, provider ai.ApiProvider, toolReg *tools.Registry, systemPrompt string) error {
	term := terminal.NewProcessTerminal()
	defer terminal.RestoreOnPanic(term)

	app := interactive.NewFromDeps(interactive.AppDeps{
		Terminal:     term,
		Provider:     provider,
		Model:        model,
		Tools:        toolReg.All(),
		Checker:      checker,
		SystemPrompt: systemPrompt,
		Version:      version,
	})

	if checker.Mode() == permission.ModeYolo {
		app.SetYoloMode()
	}

	return app.Run()
}
