package internal

import (
	"sync"
	"time"
)

// Debouncer batches rapid events into a single signal after a quiet period.
// If a new event arrives while a batch is pending, the timer resets.
type Debouncer struct {
	interval time.Duration
	mu       sync.Mutex
	timer    *time.Timer
	ch       chan struct{}
	stopped  bool
}

// NewDebouncer creates a Debouncer with the given interval.
// Typical values: 200-500ms to collapse editor write storms.
func NewDebouncer(interval time.Duration) *Debouncer {
	return &Debouncer{
		interval: interval,
		ch:       make(chan struct{}, 1),
	}
}

// Trigger records that an event occurred. After the debounce interval
// with no further events, a signal is sent on Events().
func (d *Debouncer) Trigger() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.stopped {
		return
	}

	if d.timer != nil {
		d.timer.Stop()
	}

	d.timer = time.AfterFunc(d.interval, func() {
		d.mu.Lock()
		defer d.mu.Unlock()
		if d.stopped {
			return
		}
		d.timer = nil
		select {
		case d.ch <- struct{}{}:
		default:
			// Channel already has a pending signal
		}
	})
}

// Events returns the channel that emits debounced signals.
func (d *Debouncer) Events() <-chan struct{} {
	return d.ch
}

// Stop stops the debouncer and releases resources.
func (d *Debouncer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.stopped = true
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
	close(d.ch)
}
