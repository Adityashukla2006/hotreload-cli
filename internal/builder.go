package internal

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
)

// Builder executes the user-supplied build command.
type Builder struct {
	cmdStr string
	mu     sync.Mutex
	cancel context.CancelFunc
}

// NewBuilder creates a Builder for the given command string.
func NewBuilder(cmdStr string) *Builder {
	return &Builder{cmdStr: cmdStr}
}

// Build runs the build command. It can be cancelled by calling Cancel.
// Output is streamed to the logger in real time.
func (b *Builder) Build(log *Logger) error {
	ctx, cancel := context.WithCancel(context.Background())
	b.mu.Lock()
	b.cancel = cancel
	b.mu.Unlock()

	shell, args := shellArgs(b.cmdStr)
	cmd := exec.CommandContext(ctx, shell, args...)
	cmd.Stdout = log.StreamPipe("[build] ")
	cmd.Stderr = log.StreamPipe("[build] ")

	err := cmd.Run()

	b.mu.Lock()
	b.cancel = nil
	b.mu.Unlock()

	if err != nil {
		if ctx.Err() != nil {
			return nil // Cancelled, not a real failure
		}
		return fmt.Errorf("build failed: %w", err)
	}
	return nil
}

// Cancel stops the current build if one is running.
func (b *Builder) Cancel() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cancel != nil {
		b.cancel()
		b.cancel = nil
	}
}
