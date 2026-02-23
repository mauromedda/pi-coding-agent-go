// ABOUTME: Agent loop: prompt -> stream -> tool execution -> repeat
// ABOUTME: Orchestrates LLM calls and tool invocations with concurrent read-only execution

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	pilog "github.com/mauromedda/pi-coding-agent-go/internal/log"
	"github.com/mauromedda/pi-coding-agent-go/internal/perf"
	"github.com/mauromedda/pi-coding-agent-go/internal/session"
	"github.com/mauromedda/pi-coding-agent-go/pkg/ai"
	"golang.org/x/sync/errgroup"
)

// PermCheckFunc validates whether a tool is allowed to execute.
// Returns nil if allowed, error with reason if blocked.
type PermCheckFunc func(tool string, args map[string]any) error

// AdaptiveConfig holds optional adaptive performance parameters for the agent.
type AdaptiveConfig struct {
	Profile    perf.ModelProfile
	Compaction session.CompactionConfig
	Summarizer session.SummarizerFunc // optional; if nil, compaction is skipped
}

// Agent orchestrates the prompt-stream-tool loop against an LLM provider.
type Agent struct {
	provider  ai.ApiProvider
	model     *ai.Model
	tools     map[string]*AgentTool
	permCheck PermCheckFunc
	adaptive  *AdaptiveConfig
	state     atomic.Int32 // stores AgentState
	events    chan AgentEvent
	steerCh   chan ai.Message
	cancelFn  context.CancelFunc
}

// New creates an Agent wired to the given provider, model, and tool set.
func New(provider ai.ApiProvider, model *ai.Model, tools []*AgentTool) *Agent {
	return NewWithPermissions(provider, model, tools, nil)
}

// NewWithPermissions creates an Agent with an optional permission checker.
func NewWithPermissions(provider ai.ApiProvider, model *ai.Model, tools []*AgentTool, permCheck PermCheckFunc) *Agent {
	tm := make(map[string]*AgentTool, len(tools))
	for _, t := range tools {
		tm[t.Name] = t
	}

	return &Agent{
		provider:  provider,
		model:     model,
		tools:     tm,
		permCheck: permCheck,
		steerCh:   make(chan ai.Message, 8),
	}
}

// SetAdaptive configures adaptive performance for the agent.
func (a *Agent) SetAdaptive(cfg *AdaptiveConfig) {
	a.adaptive = cfg
}

// Prompt starts the agent loop in a goroutine and returns an event channel.
// The channel is closed when the loop terminates (end-turn, error, or cancel).
func (a *Agent) Prompt(ctx context.Context, llmCtx *ai.Context, opts *ai.StreamOptions) <-chan AgentEvent {
	ctx, cancel := context.WithCancel(ctx)
	a.cancelFn = cancel
	a.events = make(chan AgentEvent, 1024)
	a.state.Store(int32(StateRunning))

	go a.loop(ctx, llmCtx, opts)

	return a.events
}

// Steer injects a steering message that will be appended before the next LLM call.
func (a *Agent) Steer(msg ai.Message) {
	select {
	case a.steerCh <- msg:
	default:
	}
}

// Abort cancels the current agent loop.
func (a *Agent) Abort() {
	a.state.Store(int32(StateCancelled))
	if a.cancelFn != nil {
		a.cancelFn()
	}
}

// State returns the current lifecycle state.
func (a *Agent) State() AgentState {
	return AgentState(a.state.Load())
}

// loop is the core prompt-stream-tool cycle.
func (a *Agent) loop(ctx context.Context, llmCtx *ai.Context, opts *ai.StreamOptions) {
	defer close(a.events)
	defer func() {
		// Preserve StateCancelled if Abort() was called.
		a.state.CompareAndSwap(int32(StateRunning), int32(StateIdle))
	}()
	// Terminal events (start, end, errors that break the loop) use emitFinal
	// so they are delivered even after context cancellation.
	pilog.Debug("agent: loop start model=%s tools=%d", a.model.Name, len(a.tools))
	a.emitFinal(AgentEvent{Type: EventAgentStart})

	for {
		if err := ctx.Err(); err != nil {
			a.emitFinal(AgentEvent{Type: EventError, Error: fmt.Errorf("agent cancelled: %w", err)})
			break
		}

		a.drainSteeringMessages(llmCtx)
		a.applyAdaptive(ctx, llmCtx, opts)

		msg, err := a.streamResponse(ctx, llmCtx, opts)
		if err != nil {
			a.emitFinal(AgentEvent{Type: EventError, Error: fmt.Errorf("streaming response: %w", err)})
			break
		}

		toolCalls, parseErrResults := extractToolCalls(msg)
		llmCtx.Messages = append(llmCtx.Messages, assistantMessage(msg))

		if len(toolCalls) == 0 && len(parseErrResults) == 0 {
			break
		}

		var results []toolExecResult

		// Include parse-error results so the LLM can self-correct.
		results = append(results, parseErrResults...)

		if len(toolCalls) > 0 {
			execResults, err := a.executeTools(ctx, toolCalls)
			if err != nil {
				a.emitFinal(AgentEvent{Type: EventError, Error: fmt.Errorf("executing tools: %w", err)})
				break
			}
			results = append(results, execResults...)
		}

		llmCtx.Messages = append(llmCtx.Messages, toolResultMessage(results))
	}

	a.emitFinal(AgentEvent{Type: EventAgentEnd})
}

// drainSteeringMessages appends any pending steering messages to the context.
func (a *Agent) drainSteeringMessages(llmCtx *ai.Context) {
	for {
		select {
		case msg := <-a.steerCh:
			llmCtx.Messages = append(llmCtx.Messages, msg)
		default:
			return
		}
	}
}

// applyAdaptive runs pre-flight decisions: compaction and adaptive MaxOutputTokens.
func (a *Agent) applyAdaptive(ctx context.Context, llmCtx *ai.Context, opts *ai.StreamOptions) {
	if a.adaptive == nil {
		return
	}

	inputTokens := session.EstimateMessagesTokens(llmCtx.Messages)
	contextWindow := a.adaptive.Profile.ContextWindow
	params := perf.Decide(a.adaptive.Profile, inputTokens, contextWindow)

	// Pre-flight compaction
	if params.CompactBeforeCall && a.adaptive.Summarizer != nil {
		if session.ShouldCompact(llmCtx.Messages, contextWindow, a.adaptive.Compaction) {
			pilog.Debug("agent: pre-flight compaction triggered (input=%d, window=%d)", inputTokens, contextWindow)
			result, err := session.CompactWithLLM(ctx, llmCtx.Messages, a.adaptive.Compaction, a.adaptive.Summarizer)
			if err != nil {
				pilog.Debug("agent: pre-flight compaction failed: %v", err)
			} else {
				llmCtx.Messages = result.Messages
				pilog.Debug("agent: compacted %d→%d tokens", result.TokensBefore, session.EstimateMessagesTokens(result.Messages))
			}
		}
	}

	// Adaptive MaxOutputTokens
	opts.MaxTokens = params.MaxOutputTokens

	// Prompt caching
	if params.UsePromptCaching {
		ai.ApplyPromptCaching(llmCtx, a.model.Api)
	}
}

// streamResponse streams a single LLM response, emitting text/thinking events.
func (a *Agent) streamResponse(ctx context.Context, llmCtx *ai.Context, opts *ai.StreamOptions) (*ai.AssistantMessage, error) {
	pilog.Debug("agent: streaming model=%s messages=%d", a.model.Name, len(llmCtx.Messages))
	stream := a.provider.Stream(ctx, a.model, llmCtx, opts)

	for evt := range stream.Events() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context cancelled during stream: %w", ctx.Err())
		}
		a.forwardStreamEvent(ctx, evt)
	}

	result := stream.Result()
	if result == nil {
		return nil, fmt.Errorf("stream completed without result")
	}

	// Emit token usage stats
	usage := result.Usage
	a.emit(ctx, AgentEvent{Type: EventUsageUpdate, Usage: &usage})

	return result, nil
}

// forwardStreamEvent translates an ai.StreamEvent into an AgentEvent.
func (a *Agent) forwardStreamEvent(ctx context.Context, evt ai.StreamEvent) {
	switch evt.Type {
	case ai.EventContentDelta:
		a.emit(ctx, AgentEvent{Type: EventAssistantText, Text: evt.Text})
	case ai.EventThinkingDelta:
		a.emit(ctx, AgentEvent{Type: EventAssistantThinking, Text: evt.Text})
	case ai.EventError:
		a.emit(ctx, AgentEvent{Type: EventError, Error: evt.Error})
	}
}

// emit sends an event; blocks until delivered or context is cancelled.
func (a *Agent) emit(ctx context.Context, evt AgentEvent) {
	select {
	case a.events <- evt:
	case <-ctx.Done():
	}
}

// emitFinal sends a lifecycle event unconditionally.
// Used for start, end, and loop-terminating errors that must be delivered
// even after context cancellation. Safe because the loop is the sole producer
// and the channel is buffered.
func (a *Agent) emitFinal(evt AgentEvent) {
	a.events <- evt
}

// toolCall holds a parsed tool invocation from the model's response.
type toolCall struct {
	ID   string
	Name string
	Args map[string]any
}

// extractToolCalls pulls tool-use content blocks from the assistant message.
// Returns valid parsed calls and error results for any calls whose input
// could not be parsed, so the LLM receives feedback and can self-correct.
func extractToolCalls(msg *ai.AssistantMessage) ([]toolCall, []toolExecResult) {
	var calls []toolCall
	var errResults []toolExecResult
	for _, c := range msg.Content {
		if c.Type != ai.ContentToolUse {
			continue
		}

		args, err := ParseToolArgs(c.Input)
		if err != nil {
			errResults = append(errResults, toolExecResult{
				ID: c.ID,
				Result: ToolResult{
					Content: fmt.Sprintf("failed to parse arguments for tool %s: %v", c.Name, err),
					IsError: true,
				},
			})
			continue
		}

		calls = append(calls, toolCall{ID: c.ID, Name: c.Name, Args: args})
	}
	return calls, errResults
}

// toolExecResult pairs a tool call ID with its execution result.
type toolExecResult struct {
	ID     string
	Result ToolResult
}

// executeTools runs tool calls: read-only tools concurrently, write tools sequentially.
func (a *Agent) executeTools(ctx context.Context, calls []toolCall) ([]toolExecResult, error) {
	readOnly, write := partitionToolCalls(a.tools, calls)

	results := make([]toolExecResult, 0, len(calls))

	roResults, err := a.executeReadOnlyTools(ctx, readOnly)
	if err != nil {
		return nil, fmt.Errorf("read-only tool execution: %w", err)
	}
	results = append(results, roResults...)

	wResults, err := a.executeWriteTools(ctx, write)
	if err != nil {
		return nil, fmt.Errorf("write tool execution: %w", err)
	}
	results = append(results, wResults...)

	return results, nil
}

// executeReadOnlyTools runs read-only tool calls concurrently via errgroup.
func (a *Agent) executeReadOnlyTools(ctx context.Context, calls []toolCall) ([]toolExecResult, error) {
	if len(calls) == 0 {
		return nil, nil
	}

	results := make([]toolExecResult, len(calls))
	g, gCtx := errgroup.WithContext(ctx)

	for i, tc := range calls {
		g.Go(func() error {
			res, err := a.executeSingleTool(gCtx, tc)
			if err != nil {
				return err
			}
			results[i] = res
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("concurrent tool execution: %w", err)
	}

	return results, nil
}

// executeWriteTools runs write tool calls sequentially.
func (a *Agent) executeWriteTools(ctx context.Context, calls []toolCall) ([]toolExecResult, error) {
	results := make([]toolExecResult, 0, len(calls))
	for _, tc := range calls {
		res, err := a.executeSingleTool(ctx, tc)
		if err != nil {
			return nil, fmt.Errorf("sequential tool %s execution: %w", tc.Name, err)
		}
		results = append(results, res)
	}
	return results, nil
}

// executeSingleTool runs one tool call, emitting start/update/end events.
// Checks permissions before execution if a permission checker is configured.
func (a *Agent) executeSingleTool(ctx context.Context, tc toolCall) (toolExecResult, error) {
	tool, ok := a.tools[tc.Name]
	if !ok {
		return toolExecResult{
			ID:     tc.ID,
			Result: ToolResult{Content: fmt.Sprintf("unknown tool: %s", tc.Name), IsError: true},
		}, nil
	}

	// Permission check before execution
	if a.permCheck != nil {
		if err := a.permCheck(tc.Name, tc.Args); err != nil {
			result := ToolResult{Content: err.Error(), IsError: true}
			a.emit(ctx, AgentEvent{
				Type: EventToolEnd, ToolID: tc.ID, ToolName: tc.Name, ToolResult: &result,
			})
			return toolExecResult{ID: tc.ID, Result: result}, nil
		}
	}

	pilog.Debug("agent: tool %s (id=%s)", tc.Name, tc.ID)
	a.emit(ctx, AgentEvent{
		Type: EventToolStart, ToolID: tc.ID, ToolName: tc.Name, ToolArgs: tc.Args,
	})

	start := time.Now()
	onUpdate := func(u ToolUpdate) {
		a.emit(ctx, AgentEvent{Type: EventToolUpdate, ToolID: tc.ID, ToolName: tc.Name, Text: u.Output})
	}

	result, err := tool.Execute(ctx, tc.ID, tc.Args, onUpdate)
	result.Duration = time.Since(start)

	if err != nil {
		result.Content = err.Error()
		result.IsError = true
	}

	a.emit(ctx, AgentEvent{
		Type: EventToolEnd, ToolID: tc.ID, ToolName: tc.Name, ToolResult: &result,
	})

	return toolExecResult{ID: tc.ID, Result: result}, nil
}

// partitionToolCalls splits calls into read-only and write groups.
func partitionToolCalls(tools map[string]*AgentTool, calls []toolCall) (readOnly, write []toolCall) {
	for _, tc := range calls {
		tool, ok := tools[tc.Name]
		if ok && tool.ReadOnly {
			readOnly = append(readOnly, tc)
		} else {
			write = append(write, tc)
		}
	}
	return readOnly, write
}

// assistantMessage converts an AssistantMessage into a conversation Message.
func assistantMessage(msg *ai.AssistantMessage) ai.Message {
	return ai.Message{Role: ai.RoleAssistant, Content: msg.Content}
}

// toolResultMessage builds a user message containing tool results.
func toolResultMessage(results []toolExecResult) ai.Message {
	contents := make([]ai.Content, 0, len(results))
	for _, r := range results {
		contents = append(contents, ai.Content{
			Type:    ai.ContentToolResult,
			ID:      r.ID,
			ResultText: r.Result.Content,
			IsError: r.Result.IsError,
		})
	}
	return ai.Message{Role: ai.RoleUser, Content: contents}
}

// aiTools converts registered AgentTools into ai.Tool definitions for the LLM context.
func aiTools(tools map[string]*AgentTool) []ai.Tool {
	out := make([]ai.Tool, 0, len(tools))
	for _, t := range tools {
		schema := t.Parameters
		if schema == nil {
			schema = json.RawMessage(`{}`)
		}
		out = append(out, ai.Tool{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  schema,
		})
	}
	return out
}
