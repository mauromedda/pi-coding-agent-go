# Bubble Tea Integration for .pi

## Overview

Bubble Tea has been added as a dependency to .pi. While the current TUI implementation (`pkg/tui`) remains the primary rendering engine, Bubble Tea (`github.com/charmbracelet/bubbletea`) is now available for future enhancements and as a potential migration path.

## Current State

### Bubble Tea Dependency
- Added as an optional dependency via `go get github.com/charmbracelet/bubbletea@v1.3.10`
- Available for use in new features or full migration
- Provides additional TUI components and patterns

### Existing TUI (`pkg/tui`)
- Custom TUI implementation following the Component pattern
- Thread-safe containers with RWMutex
- Differential rendering with buffer pooling
- Overlay support for modals and dialogs
- Component interface: `Render(out *RenderBuffer, width int)`, `Invalidate()`

## Migration Path (Future Work)

### Phase 1: Dependency Addition ✓
- Add Bubble Tea to dependencies ✓
- Verify existing code still works ✓

### Phase 2: Selective Integration (Future)
- Use Bubble Tea for specific UI elements ( dialogs, loaders, selectors)
- Gradually migrate components
- Preserve existing rendering patterns

### Phase 3: Full Migration (Future)
- Replace `pkg/tui` with Bubble Tea
- Update all components to Bubble Tea's model-view-update pattern
- Implement message-passing architecture

## Key Differences: Custom TUI vs Bubble Tea

### Custom TUI (Current)
```go
type Component interface {
    Render(out *RenderBuffer, width int)
    Invalidate()
}
```

- Direct rendering to buffer
- Immutable state during render
- Container-based composition

### Bubble Tea
```go
type Msg interface{}
type Model interface {
    Init() Cmd
    Update(Msg) (Model, Cmd)
    View() string
}
```

- Message-passing architecture
- State is updated via messages
- View is computed from current state

## Current Usage

Bubble Tea is available for import:
```go
import tea "github.com/charmbracelet/bubbletea"
```

## Benefits of Bubble Tea

1. **Mature Framework**: Well-tested, production-ready
2. **Rich Components**: Built-in components for common patterns
3. **Threading Model**: Simplified concurrent updates
4. **Community**: Large ecosystem and documentation
5. **Features**: Built-in support for:
   - Mouse input
   - Mouse selection
   - Progress bars
   - Spinners
   - List selectors
   - Text inputs

## Building

```bash
go build ./...
```

All tests pass:
```bash
go test ./...
```

## Files Modified

1. `go.mod` - Added Bubble Tea dependency
2. `internal/mode/interactive/interactive_test.go` - Updated test to use ToolCall

## Next Steps

1. Evaluate Bubble Tea components for replacement
2. Create wrapper components if needed
3. Plan incremental migration
4. Update documentation with migration plan
