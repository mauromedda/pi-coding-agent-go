// ABOUTME: Defines the Terminal interface for raw mode, size queries, and output.
// ABOUTME: Abstracts terminal operations so implementations can target real or virtual terminals.

package terminal

// Terminal abstracts low-level terminal operations: raw mode,
// size queries, output writing, and resize notifications.
type Terminal interface {
	EnterRawMode() error
	ExitRawMode() error
	Size() (width, height int, err error)
	Write(p []byte) (n int, err error)
	OnResize(fn func(width, height int))
}
