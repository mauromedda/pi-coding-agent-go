// ABOUTME: CLI entry point for pi-go with terminal crash recovery
// ABOUTME: Parses flags, loads config, registers providers/tools, dispatches to mode

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mauromedda/pi-coding-agent-go/internal/config"
	"github.com/mauromedda/pi-coding-agent-go/internal/mode/interactive"
	"github.com/mauromedda/pi-coding-agent-go/internal/mode/print"
	"github.com/mauromedda/pi-coding-agent-go/internal/permission"
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

	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	auth, err := config.LoadAuth()
	if err != nil {
		return fmt.Errorf("loading auth: %w", err)
	}

	model, err := resolveModel(args, cfg)
	if err != nil {
		return fmt.Errorf("resolving model: %w", err)
	}

	baseURL := resolveBaseURL(args, cfg)

	provider := ai.GetProvider(model.Api, baseURL)
	if provider == nil {
		return fmt.Errorf("no provider registered for API %q", model.Api)
	}

	toolRegistry := tools.NewRegistry()

	permMode := resolvePermissionMode(args, cfg)
	checker := permission.NewChecker(permMode, nil)

	// Print mode: non-interactive, streams to stdout
	if args.print {
		prompt := strings.Join(args.remaining(), " ")
		return print.Run(context.Background(), provider, model, prompt)
	}

	// Interactive mode (default)
	return runInteractive(model, checker, auth, toolRegistry)
}

// registerProviders registers all built-in AI provider factories.
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

// runInteractive sets up the terminal, defers crash recovery, and starts the TUI.
func runInteractive(model *ai.Model, checker *permission.Checker, auth *config.AuthStore, toolReg *tools.Registry) error {
	vt := terminal.NewVirtualTerminal(120, 40)
	defer terminal.RestoreOnPanic(vt)

	app := interactive.New(vt, 120, 40, checker)

	if checker.Mode() == permission.ModeYolo {
		app.SetYoloMode()
	}

	_ = auth     // Auth store available for key retrieval during session
	_ = toolReg  // Tool registry available for agent loop
	_ = model    // Model available for session creation

	app.Start()
	defer app.Stop()

	// The interactive loop will be driven by the agent loop (future wiring).
	// For now, indicate successful startup.
	fmt.Printf("pi-go %s | model: %s | mode: %s\n", version, model.Name, checker.Mode())

	return nil
}
