// ABOUTME: TTFB network probe: measures time-to-first-byte of streaming responses
// ABOUTME: Classifies latency as Local (<50ms), Fast (<500ms), or Slow (>=500ms)

package perf

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"
)

// LatencyClass categorizes network latency.
type LatencyClass int

const (
	LatencyLocal LatencyClass = iota // <50ms TTFB
	LatencyFast                      // <500ms TTFB
	LatencySlow                      // >=500ms TTFB
)

// String returns the human-readable name of the latency class.
func (l LatencyClass) String() string {
	switch l {
	case LatencyLocal:
		return "local"
	case LatencyFast:
		return "fast"
	case LatencySlow:
		return "slow"
	default:
		return "unknown"
	}
}

// ProbeResult holds the outcome of a TTFB measurement.
type ProbeResult struct {
	TTFB    time.Duration
	Latency LatencyClass
}

// probeTimeout is the maximum time to wait for a probe response.
const probeTimeout = 5 * time.Second

// probePayload is a minimal chat completion request for measuring TTFB.
const probePayload = `{"model":"test","messages":[{"role":"user","content":"hi"}],"max_tokens":1,"stream":true}`

// ProbeTTFB sends a minimal completion request and measures time-to-first-byte.
// On any error (timeout, connection, server error), defaults to LatencySlow.
func ProbeTTFB(ctx context.Context, baseURL string, apiKey string) ProbeResult {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	url := strings.TrimRight(baseURL, "/") + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(probePayload))
	if err != nil {
		return ProbeResult{TTFB: probeTimeout, Latency: LatencySlow}
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	ttfb := time.Since(start)

	if err != nil {
		return ProbeResult{TTFB: probeTimeout, Latency: LatencySlow}
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	return ProbeResult{
		TTFB:    ttfb,
		Latency: classifyLatency(ttfb),
	}
}

// classifyLatency maps a TTFB duration to a LatencyClass.
func classifyLatency(ttfb time.Duration) LatencyClass {
	switch {
	case ttfb < 50*time.Millisecond:
		return LatencyLocal
	case ttfb < 500*time.Millisecond:
		return LatencyFast
	default:
		return LatencySlow
	}
}
