// ABOUTME: Dependency injection struct for the Bubble Tea interactive app
// ABOUTME: Mirrors interactive.AppDeps; adapted for the btea architecture

package btea

import (
	"github.com/mauromedda/pi-coding-agent-go/internal/agent"
	"github.com/mauromedda/pi-coding-agent-go/internal/config"
	"github.com/mauromedda/pi-coding-agent-go/internal/permission"
	"github.com/mauromedda/pi-coding-agent-go/internal/statusline"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// AppDeps bundles all dependencies for the Bubble Tea interactive app.
type AppDeps struct {
	Provider             ai.ApiProvider
	Model                *ai.Model
	Tools                []*agent.AgentTool
	Checker              *permission.Checker
	SystemPrompt         string
	Version              string
	StatusEngine         *statusline.Engine
	AutoCompactThreshold int
	Hooks                map[string][]config.HookDef
	ScopedModels         *config.ScopedModelsConfig
	PermissionMode       permission.Mode
}
