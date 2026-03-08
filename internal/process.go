package internal

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"time"
)

const (
	gracefulShutdownTimeout = 5 * time.Second
	crashLoopThreshold      = 2 * time.Second
	restartCooldown         = 3 * time.Second
)

// Process manages the server process lifecycle with cross-platform process group killing.
type Process struct {
	cmdStr    string
	cmd       *exec.Cmd
	startedAt time.Time
	lastUptime time.Duration // set when process exits, for crash loop detection
	mu        sync.Mutex
}

// NewProcess creates a Process for the given exec command.
func NewProcess(cmdStr string) *Process {
	return &Process{cmdStr: cmdStr}
}

// Start launches the server. Stdout/stderr stream to the terminal in real time.
func (p *Process) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	shell, args := shellArgs(p.cmdStr)
	cmd := exec.Command(shell, args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	initProcess(cmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	p.cmd = cmd
	p.startedAt = time.Now()
	slog.Info("Server started", "pid", cmd.Process.Pid)

	go p.watchExit()
	return nil
}

// Stop terminates the server and its entire process tree.
func (p *Process) Stop() {
	p.mu.Lock()
	cmd := p.cmd
	p.cmd = nil
	p.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return
	}

	pid := cmd.Process.Pid
	slog.Info("Stopping server", "pid", pid)

	killProcessTree(pid)

	done := make(chan struct{})
	go func() {
		cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("Server exited")
	case <-time.After(gracefulShutdownTimeout):
		slog.Warn("Server did not exit in time, force killing")
		killProcessTreeForce(pid)
		<-done
		slog.Info("Server force killed")
	}
}

// initProcess, killProcessTree, killProcessTreeForce are implemented in process_*.go

// IsRunning returns true if the server is currently running.
func (p *Process) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cmd != nil && p.cmd.Process != nil
}

// Uptime returns how long the server has been running.
func (p *Process) Uptime() time.Duration {
	p.mu.Lock()
	t := p.startedAt
	p.mu.Unlock()
	return time.Since(t)
}

// WasCrashLoop returns true if the server exited within the crash threshold.
func (p *Process) WasCrashLoop() bool {
	p.mu.Lock()
	uptime := p.lastUptime
	p.mu.Unlock()
	return uptime > 0 && uptime < crashLoopThreshold
}

// RestartCooldown returns the cooldown duration after a crash.
func RestartCooldown() time.Duration {
	return restartCooldown
}

func (p *Process) watchExit() {
	cmd := p.cmd
	if cmd == nil {
		return
	}
	cmd.Wait()

	p.mu.Lock()
	if p.cmd == nil {
		p.mu.Unlock()
		return
	}
	uptime := time.Since(p.startedAt)
	p.lastUptime = uptime
	p.cmd = nil
	p.mu.Unlock()

	if uptime < crashLoopThreshold {
		slog.Warn("Server crashed very quickly – possible crash loop, will not auto-restart",
			"uptime", uptime.Round(time.Millisecond))
	} else {
		slog.Warn("Server exited unexpectedly", "uptime", uptime.Round(time.Millisecond))
	}
}
