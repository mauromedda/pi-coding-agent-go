// ABOUTME: Factory function that builds a TransformContextFunc from mode + model + provider
// ABOUTME: Used by CLI flag wiring to create the minion distillation pipeline

package minion

import (
	"context"

	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
)

// MinionMode selects singular or plural protocol.
type MinionMode string

const (
	ModeSingular MinionMode = "singular"
	ModePlural   MinionMode = "plural"
)

// WireConfig holds everything needed to build a TransformContextFunc.
type WireConfig struct {
	Mode     MinionMode
	Model    *ai.Model
	Provider ai.ApiProvider
}

// BuildTransform creates a context transform function from the given config.
// The returned function is compatible with agent.AdaptiveConfig.TransformContext.
func BuildTransform(cfg WireConfig) func(ctx context.Context, msgs []ai.Message) ([]ai.Message, error) {
	switch cfg.Mode {
	case ModePlural:
		d := NewDistributor(DistributorConfig{
			Provider: cfg.Provider,
			Model:    cfg.Model,
		})
		return d.Distribute
	default: // singular
		d := New(Config{
			Provider: cfg.Provider,
			Model:    cfg.Model,
		})
		return d.Distill
	}
}
