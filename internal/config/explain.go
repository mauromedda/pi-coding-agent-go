// ABOUTME: Human-readable rendering of effective configuration
// ABOUTME: Used by "config explain" CLI subcommand to show merged settings

package config

import (
	"fmt"
	"strings"
)

// Explain renders a human-readable summary of the effective settings.
// Shows non-zero values grouped by section.
func Explain(s *Settings) string {
	if s == nil {
		s = &Settings{}
	}

	var b strings.Builder

	// General
	b.WriteString("=== General ===\n")
	if s.Model != "" {
		fmt.Fprintf(&b, "  Model:       %s\n", s.Model)
	}
	if s.BaseURL != "" {
		fmt.Fprintf(&b, "  BaseURL:     %s\n", s.BaseURL)
	}
	if s.Temperature != 0 {
		fmt.Fprintf(&b, "  Temperature: %.2f\n", s.Temperature)
	}
	if s.MaxTokens != 0 {
		fmt.Fprintf(&b, "  MaxTokens:   %d\n", s.MaxTokens)
	}
	if s.Yolo {
		b.WriteString("  Yolo:        true\n")
	}
	if s.Thinking {
		b.WriteString("  Thinking:    true\n")
	}
	if s.Theme != "" {
		fmt.Fprintf(&b, "  Theme:       %s\n", s.Theme)
	}
	b.WriteString("\n")

	// Permissions
	b.WriteString("=== Permissions ===\n")
	if s.DefaultMode != "" {
		fmt.Fprintf(&b, "  DefaultMode: %s\n", s.DefaultMode)
	}
	if len(s.Allow) > 0 {
		fmt.Fprintf(&b, "  Allow:       %s\n", strings.Join(s.Allow, ", "))
	}
	if len(s.Deny) > 0 {
		fmt.Fprintf(&b, "  Deny:        %s\n", strings.Join(s.Deny, ", "))
	}
	if len(s.Ask) > 0 {
		fmt.Fprintf(&b, "  Ask:         %s\n", strings.Join(s.Ask, ", "))
	}
	b.WriteString("\n")

	// Intent
	b.WriteString("=== Intent ===\n")
	if s.Intent != nil {
		fmt.Fprintf(&b, "  Enabled:            %v\n", s.Intent.IsEnabled())
		if s.Intent.HeuristicThreshold != 0 {
			fmt.Fprintf(&b, "  HeuristicThreshold: %.2f\n", s.Intent.HeuristicThreshold)
		}
		if s.Intent.AutoPlanFileCount != 0 {
			fmt.Fprintf(&b, "  AutoPlanFileCount:  %d\n", s.Intent.AutoPlanFileCount)
		}
	}
	b.WriteString("\n")

	// Prompts
	b.WriteString("=== Prompts ===\n")
	if s.Prompts != nil {
		if s.Prompts.ActiveVersion != "" {
			fmt.Fprintf(&b, "  ActiveVersion:         %s\n", s.Prompts.ActiveVersion)
		}
		if s.Prompts.OverridesDir != "" {
			fmt.Fprintf(&b, "  OverridesDir:          %s\n", s.Prompts.OverridesDir)
		}
		if s.Prompts.MaxSystemPromptTokens != 0 {
			fmt.Fprintf(&b, "  MaxSystemPromptTokens: %d\n", s.Prompts.MaxSystemPromptTokens)
		}
	}
	b.WriteString("\n")

	// Personality
	b.WriteString("=== Personality ===\n")
	if s.Personality != nil {
		if s.Personality.Profile != "" {
			fmt.Fprintf(&b, "  Profile: %s\n", s.Personality.Profile)
		}
		for name, check := range s.Personality.Checks {
			fmt.Fprintf(&b, "  Check[%s]: enabled=%v", name, check.IsEnabled())
			if check.Level != "" {
				fmt.Fprintf(&b, " level=%s", check.Level)
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")

	// Telemetry
	b.WriteString("=== Telemetry ===\n")
	if s.Telemetry != nil {
		fmt.Fprintf(&b, "  Enabled:   %v\n", s.Telemetry.IsEnabled())
		if s.Telemetry.BudgetUSD != 0 {
			fmt.Fprintf(&b, "  BudgetUSD: %.2f\n", s.Telemetry.BudgetUSD)
		}
		if s.Telemetry.WarnAtPct != 0 {
			fmt.Fprintf(&b, "  WarnAtPct: %d%%\n", s.Telemetry.WarnAtPct)
		}
	}
	b.WriteString("\n")

	// Safety
	b.WriteString("=== Safety ===\n")
	if s.Safety != nil {
		if len(s.Safety.NeverModify) > 0 {
			fmt.Fprintf(&b, "  NeverModify: %s\n", strings.Join(s.Safety.NeverModify, ", "))
		}
		if len(s.Safety.LockedKeys) > 0 {
			fmt.Fprintf(&b, "  LockedKeys:  %s\n", strings.Join(s.Safety.LockedKeys, ", "))
		}
	}
	b.WriteString("\n")

	// Compaction
	b.WriteString("=== Compaction ===\n")
	if s.Compaction != nil {
		fmt.Fprintf(&b, "  Enabled:          %v\n", s.Compaction.IsEnabled())
		if s.Compaction.ReserveTokens != 0 {
			fmt.Fprintf(&b, "  ReserveTokens:    %d\n", s.Compaction.ReserveTokens)
		}
		if s.Compaction.KeepRecentTokens != 0 {
			fmt.Fprintf(&b, "  KeepRecentTokens: %d\n", s.Compaction.KeepRecentTokens)
		}
	}
	b.WriteString("\n")

	// Retry
	b.WriteString("=== Retry ===\n")
	if s.Retry != nil {
		if s.Retry.MaxRetries != 0 {
			fmt.Fprintf(&b, "  MaxRetries: %d\n", s.Retry.MaxRetries)
		}
		if s.Retry.BaseDelay != 0 {
			fmt.Fprintf(&b, "  BaseDelay:  %dms\n", s.Retry.BaseDelay)
		}
		if s.Retry.MaxDelay != 0 {
			fmt.Fprintf(&b, "  MaxDelay:   %dms\n", s.Retry.MaxDelay)
		}
	}
	b.WriteString("\n")

	// Terminal
	b.WriteString("=== Terminal ===\n")
	if s.Terminal != nil {
		if s.Terminal.LineWidth != 0 {
			fmt.Fprintf(&b, "  LineWidth: %d\n", s.Terminal.LineWidth)
		}
		if s.Terminal.Pager {
			b.WriteString("  Pager:     true\n")
		}
	}
	b.WriteString("\n")

	// Sandbox
	b.WriteString("=== Sandbox ===\n")
	if len(s.Sandbox.ExcludedCommands) > 0 {
		fmt.Fprintf(&b, "  ExcludedCommands: %s\n", strings.Join(s.Sandbox.ExcludedCommands, ", "))
	}
	if len(s.Sandbox.AllowedDomains) > 0 {
		fmt.Fprintf(&b, "  AllowedDomains:   %s\n", strings.Join(s.Sandbox.AllowedDomains, ", "))
	}
	b.WriteString("\n")

	return b.String()
}
