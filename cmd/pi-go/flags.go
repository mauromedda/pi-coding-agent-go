// ABOUTME: CLI flag parsing using stdlib flag package
// ABOUTME: Supports --yolo, --model, --plan, --print, --thinking, --version, --update

package main

import "flag"

type cliArgs struct {
	yolo     bool
	model    string
	plan     bool
	print    bool
	thinking bool
	version  bool
	update   bool
	baseURL  string
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

	flag.Parse()
	return args
}

// remaining returns the non-flag command-line arguments.
func (a cliArgs) remaining() []string {
	return flag.Args()
}
