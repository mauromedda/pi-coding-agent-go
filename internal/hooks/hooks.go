// ABOUTME: Hook engine that fires lifecycle hooks matching tool events
// ABOUTME: Pre-compiles regex matchers; runs matching hooks sequentially per event

package hooks

import (
	"context"
	"fmt"
	"regexp"

	"github.com/mauromedda/pi-coding-agent-go/internal/config"
)

// compiledHook pairs a hook definition with its pre-compiled regex matcher.
type compiledHook struct {
	def   config.HookDef
	regex *regexp.Regexp // nil means match-all
}

// Engine holds registered hooks and fires them on lifecycle events.
type Engine struct {
	hooks map[string][]compiledHook
}

// NewEngine creates a hook engine from the hooks configuration map.
// It pre-compiles all regex matchers and returns an error if any are invalid.
func NewEngine(hooks map[string][]config.HookDef) (*Engine, error) {
	compiled := make(map[string][]compiledHook, len(hooks))

	for event, defs := range hooks {
		for _, def := range defs {
			ch := compiledHook{def: def}
			if def.Matcher != "" {
				re, err := regexp.Compile(def.Matcher)
				if err != nil {
					return nil, fmt.Errorf("invalid hook matcher %q for event %s: %w", def.Matcher, event, err)
				}
				ch.regex = re
			}
			compiled[event] = append(compiled[event], ch)
		}
	}

	return &Engine{hooks: compiled}, nil
}

// Fire runs all hooks registered for the given event.
// Hooks are matched by regex against the tool name and run sequentially.
// If any hook blocks execution (non-zero exit), Fire returns immediately
// with Blocked=true. Env maps from all hooks are merged into the output.
func (e *Engine) Fire(ctx context.Context, input HookInput) (HookOutput, error) {
	defs, ok := e.hooks[string(input.Event)]
	if !ok {
		return HookOutput{}, nil
	}

	var merged HookOutput

	for _, hook := range defs {
		if hook.regex != nil && !hook.regex.MatchString(input.Tool) {
			continue
		}

		out, err := runHookCommand(ctx, hook.def.Command, input)
		if err != nil {
			return HookOutput{}, fmt.Errorf("hook %q: %w", hook.def.Command, err)
		}

		// Merge env from this hook into the accumulated output.
		if len(out.Env) > 0 {
			if merged.Env == nil {
				merged.Env = make(map[string]string)
			}
			for k, v := range out.Env {
				merged.Env[k] = v
			}
		}

		if out.Blocked {
			merged.Blocked = true
			merged.Message = out.Message
			return merged, nil
		}

		// Keep last non-empty message.
		if out.Message != "" {
			merged.Message = out.Message
		}
	}

	return merged, nil
}
