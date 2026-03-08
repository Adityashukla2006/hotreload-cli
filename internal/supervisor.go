package internal

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

const debounceInterval = 300 * time.Millisecond

// Supervisor orchestrates the rebuild pipeline: watcher → debounce → build → process.
type Supervisor struct {
	root     string
	buildCmd string
	execCmd  string
	log      *Logger

	watcher  *Watcher
	debouncer *Debouncer
	builder  *Builder
	process  *Process

	mu         sync.Mutex
	rebuildCh   chan struct{}
	lastCrashAt time.Time
}

// NewSupervisor creates a Supervisor with the given configuration.
func NewSupervisor(root, buildCmd, execCmd string) *Supervisor {
	log := NewLogger()
	slog.SetDefault(log.Logger)
	return &Supervisor{
		root:      root,
		buildCmd:  buildCmd,
		execCmd:   execCmd,
		log:       log,
		watcher:   NewWatcher(root),
		debouncer: NewDebouncer(debounceInterval),
		builder:   NewBuilder(buildCmd),
		process:   NewProcess(execCmd),
		rebuildCh: make(chan struct{}, 1),
	}
}

// Run starts the supervisor. It blocks until the watcher stops.
func (s *Supervisor) Run() error {
	if err := s.watcher.Start(); err != nil {
		if err == ErrWatcherLimit {
			s.log.Error("File watcher limit reached",
				"hint", "On Linux: echo fs.inotify.max_user_watches=524288 | sudo tee -a /etc/sysctl.conf && sudo sysctl -p")
		}
		return fmt.Errorf("failed to start watcher: %w", err)
	}
	defer s.watcher.Stop()
	defer s.debouncer.Stop()

	// Feed watcher events into debouncer
	go func() {
		for range s.watcher.Events() {
			s.debouncer.Trigger()
		}
	}()

	// Rebuild requests: when debouncer fires, cancel in-progress build and rebuild
	rebuildCh := make(chan struct{}, 1)
	go func() {
		for range s.debouncer.Events() {
			select {
			case rebuildCh <- struct{}{}:
			default:
				// Already have a pending rebuild
			}
		}
	}()

	// Initial build and start
	s.triggerRebuild()

	// Main loop: process rebuild requests (cancel in-progress, then rebuild)
	for range rebuildCh {
		s.builder.Cancel()
		s.triggerRebuild()
	}

	return nil
}

// triggerRebuild runs the build pipeline. If a rebuild is requested during
// execution, the current build is cancelled and we rebuild the latest state.
func (s *Supervisor) triggerRebuild() {
	s.mu.Lock()
	if time.Since(s.lastCrashAt) < RestartCooldown() {
		s.log.Info("Skipping rebuild (restart cooldown after crash)")
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	s.log.Info("Change detected, rebuilding...")

	// Build first; only stop/restart if build succeeds
	if err := s.builder.Build(s.log); err != nil {
		s.log.Error("Build failed", "error", err)
		return
	}

	// Build succeeded: stop old server, start new one
	s.process.Stop()
	if err := s.process.Start(); err != nil {
		s.log.Error("Failed to start server", "error", err)
		return
	}

	// Check for crash loop
	go func() {
		time.Sleep(crashLoopThreshold)
		if s.process.IsRunning() {
			return
		}
		if s.process.WasCrashLoop() {
			s.mu.Lock()
			s.lastCrashAt = time.Now()
			s.mu.Unlock()
		}
	}()
}
