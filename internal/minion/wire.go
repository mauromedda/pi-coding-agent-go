// ABOUTME: Factory functions that build minion protocol functions from mode + model + provider
// ABOUTME: BuildIngester for pre-turn context ingest, BuildResultCompressor for sub-agent output

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

// IngestFunc distills initial context before the first agent turn.
// Returns a summary string suitable for prepending to the conversation.
type IngestFunc func(ctx context.Context, msgs []ai.Message) (string, error)

// ResultCompressorFunc compresses sub-agent output text.
// Returns the original text if it's within maxLen characters.
type ResultCompressorFunc func(ctx context.Context, text string, maxLen int) (string, error)

// WireConfig holds everything needed to build minion functions.
type WireConfig struct {
	Mode     MinionMode
	Model    *ai.Model
	Provider ai.ApiProvider
}

// BuildIngester creates a function that distills initial context (pre-first-turn).
func BuildIngester(cfg WireConfig) IngestFunc {
	d := New(Config{
		Provider: cfg.Provider,
		Model:    cfg.Model,
	})
	return d.IngestDistill
}

// BuildResultCompressor creates a function that compresses sub-agent output.
func BuildResultCompressor(cfg WireConfig) ResultCompressorFunc {
	d := New(Config{
		Provider: cfg.Provider,
		Model:    cfg.Model,
	})
	return d.CompressResult
}
