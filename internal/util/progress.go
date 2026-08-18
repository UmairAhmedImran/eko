package util

import (
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"
)

// Progress tracks file operation progress and renders a progress bar to a writer.
type Progress struct {
	total   int64
	current atomic.Int64
	writer  io.Writer
	done    chan struct{}
	prefix  string
}

// NewProgress creates a new progress tracker.
// total is the expected number of files, w is where to render (typically os.Stderr).
func NewProgress(total int, w io.Writer, prefix string) *Progress {
	return &Progress{
		total:  int64(total),
		writer: w,
		done:   make(chan struct{}),
		prefix: prefix,
	}
}

// Increment advances the progress counter by one. Safe for concurrent use.
func (p *Progress) Increment() {
	p.current.Add(1)
}

// Current returns the current progress count.
func (p *Progress) Current() int64 {
	return p.current.Load()
}

// Start begins rendering the progress bar in a background goroutine.
// Call Stop() when the operation completes.
func (p *Progress) Start() {
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-p.done:
				p.render()
				fmt.Fprintln(p.writer) // final newline
				return
			case <-ticker.C:
				p.render()
			}
		}
	}()
}

// Stop signals the progress renderer to finish and print a final newline.
func (p *Progress) Stop() {
	close(p.done)
	// Give the goroutine a moment to render the final state
	time.Sleep(50 * time.Millisecond)
}

// render prints the current progress bar state using carriage return for in-place update.
func (p *Progress) render() {
	current := p.current.Load()
	total := p.total

	// Calculate percentage and bar width
	var percent float64
	if total > 0 {
		percent = float64(current) / float64(total)
	}
	if percent > 1.0 {
		percent = 1.0
	}

	barWidth := 20
	filled := int(percent * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}

	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	// Use carriage return to overwrite the line
	fmt.Fprintf(p.writer, "\r%s [%s] %d/%d files", p.prefix, bar, current, total)
}

// IsTTY reports whether w is a terminal device.
func IsTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
