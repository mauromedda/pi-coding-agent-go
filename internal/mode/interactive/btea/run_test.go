// ABOUTME: Tests for the Run entry point and program setup logic
// ABOUTME: Validates NewAppModel initialization and tea.Program options

package btea

import (
	"testing"

	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

func TestRun_NewAppModelForRun(t *testing.T) {
	deps := AppDeps{
		Model:   &ai.Model{Name: "test-model", MaxOutputTokens: 4096},
		Version: "0.1.0-test",
	}

	m := NewAppModel(deps)

	t.Run("shared is non-nil", func(t *testing.T) {
		if m.sh == nil {
			t.Fatal("sh = nil; want non-nil")
		}
	})

	t.Run("program ref initially nil", func(t *testing.T) {
		if m.sh.program != nil {
			t.Error("sh.program should be nil before Run sets it")
		}
	})

	t.Run("context is set", func(t *testing.T) {
		if m.sh.ctx == nil {
			t.Error("sh.ctx = nil; want non-nil")
		}
	})

	t.Run("cancel is set", func(t *testing.T) {
		if m.sh.cancel == nil {
			t.Error("sh.cancel = nil; want non-nil")
		}
	})

	t.Run("init returns git branch cmd", func(t *testing.T) {
		cmd := m.Init()
		if cmd == nil {
			t.Fatal("Init() = nil; want a command")
		}
	})
}

func TestRun_BuildAITools(t *testing.T) {
	t.Run("nil tools returns empty", func(t *testing.T) {
		tools := buildAITools(nil)
		if len(tools) != 0 {
			t.Errorf("len = %d; want 0", len(tools))
		}
	})

	t.Run("converts agent tools to ai tools", func(t *testing.T) {
		agentTools := []*agent.AgentTool{
			{Name: "bash", Description: "Run bash commands"},
		}
		tools := buildAITools(agentTools)
		if len(tools) != 1 {
			t.Fatalf("len = %d; want 1", len(tools))
		}
		if tools[0].Name != "bash" {
			t.Errorf("Name = %q; want bash", tools[0].Name)
		}
		if tools[0].Description != "Run bash commands" {
			t.Errorf("Description = %q; want 'Run bash commands'", tools[0].Description)
		}
	})
}
