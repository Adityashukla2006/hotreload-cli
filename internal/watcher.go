package internal

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// ErrWatcherLimit is returned when the OS file watcher limit is reached.
var ErrWatcherLimit = errors.New("file watcher limit reached")

// ignoredDirs lists directory names we never watch.
var ignoredDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	"bin":          true,
	"out":          true,
	"target":       true,
	".idea":        true,
	".vscode":      true,
}

// ignoredExtensions lists file suffixes we ignore.
var ignoredExtensions = map[string]bool{
	".tmp": true,
	".temp": true,
	".swp": true,
	".swo": true,
	".swx": true,
	".orig": true,
	".bak": true,
	".rej": true,
	".old": true,
}

// Watcher recursively monitors a directory tree for file changes.
type Watcher struct {
	root    string
	fsw     *fsnotify.Watcher
	eventCh chan struct{}
	stopCh  chan struct{}
	mu      sync.Mutex
	stopped bool
}

// NewWatcher creates a Watcher for the given root directory.
func NewWatcher(root string) *Watcher {
	return &Watcher{
		root:    root,
		eventCh: make(chan struct{}, 1),
		stopCh:  make(chan struct{}),
	}
}

// Start initializes the watcher and recursively adds all subdirectories.
func (w *Watcher) Start() error {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	w.fsw = fsw

	if err := w.addDirRecursive(w.root); err != nil {
		fsw.Close()
		return err
	}

	go w.loop()
	return nil
}

// Stop shuts down the watcher.
func (w *Watcher) Stop() {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return
	}
	w.stopped = true
	w.mu.Unlock()

	close(w.stopCh)
	if w.fsw != nil {
		w.fsw.Close()
	}
}

// Events returns the channel that emits a signal on relevant file changes.
func (w *Watcher) Events() <-chan struct{} {
	return w.eventCh
}

func (w *Watcher) addDirRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				slog.Debug("Skipping directory (permission denied)", "path", path)
				return nil
			}
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if ignoredDirs[d.Name()] {
			return filepath.SkipDir
		}
		if err := w.fsw.Add(path); err != nil {
			if isWatcherLimitError(err) {
				return ErrWatcherLimit
			}
			slog.Debug("Could not watch directory", "path", path, "error", err)
		}
		return nil
	})
}

func isWatcherLimitError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "too many open files") ||
		strings.Contains(s, "no space left") ||
		strings.Contains(s, "inotify") && strings.Contains(s, "limit")
}

func (w *Watcher) relevant(event fsnotify.Event) bool {
	if !event.Has(fsnotify.Write) &&
		!event.Has(fsnotify.Create) &&
		!event.Has(fsnotify.Remove) &&
		!event.Has(fsnotify.Rename) {
		return false
	}
	base := filepath.Base(event.Name)
	if strings.HasPrefix(base, ".") {
		return false
	}
	ext := filepath.Ext(event.Name)
	if ignoredExtensions[ext] || strings.HasSuffix(event.Name, "~") {
		return false
	}
	for _, part := range strings.Split(filepath.ToSlash(event.Name), "/") {
		if ignoredDirs[part] {
			return false
		}
	}
	return true
}

func (w *Watcher) loop() {
	for {
		select {
		case <-w.stopCh:
			return

		case event, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if !w.relevant(event) {
				continue
			}

			if event.Has(fsnotify.Create) {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					_ = w.addDirRecursive(event.Name)
				}
			}

			slog.Debug("File event", "op", event.Op, "path", event.Name)

			select {
			case w.eventCh <- struct{}{}:
			default:
				// Already has pending signal
			}

		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			if isWatcherLimitError(err) {
				slog.Error("File watcher limit reached",
					"error", err,
					"hint", "On Linux, increase inotify limits: echo fs.inotify.max_user_watches=524288 | sudo tee -a /etc/sysctl.conf && sudo sysctl -p")
			} else {
				slog.Warn("Watcher error", "error", err)
			}
		}
	}
}
