package internal

import (
	"io"
	"log/slog"
	"os"
	"sync"
)

// Logger provides structured logging for hotreload.
type Logger struct {
	*slog.Logger
}

// NewLogger creates a Logger that writes to stdout with info level.
func NewLogger() *Logger {
	return &Logger{
		Logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})),
	}
}

// StreamPipe creates an io.Writer that forwards data to the logger in real time.
// Each line is logged with the given prefix. Used for streaming build/server output.
func (l *Logger) StreamPipe(prefix string) io.Writer {
	return &streamWriter{logger: l, prefix: prefix}
}

type streamWriter struct {
	logger *Logger
	prefix string
	mu     sync.Mutex
	buf    []byte
}

func (s *streamWriter) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buf = append(s.buf, p...)

	// Emit complete lines
	for {
		i := 0
		for i < len(s.buf) && s.buf[i] != '\n' {
			i++
		}
		if i >= len(s.buf) {
			break
		}
		line := string(s.buf[:i])
		s.buf = s.buf[i+1:]
		if line != "" {
			s.logger.Info(s.prefix + line)
		}
	}

	return len(p), nil
}
