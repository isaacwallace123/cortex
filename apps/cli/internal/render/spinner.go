package render

import (
	"fmt"
	"io"
	"sync"
	"time"
)

var spinFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner renders an animated braille spinner on a single terminal line.
// It uses \r to rewrite the line on each tick, and \033[K to clear the tail.
// Stop() waits for the goroutine to exit and clears the line before returning.
type Spinner struct {
	w     io.Writer
	mu    sync.Mutex
	label string
	stop  chan struct{}
	done  chan struct{}
}

// StartSpinner creates a spinner and begins animating immediately.
func StartSpinner(w io.Writer, label string) *Spinner {
	s := &Spinner{
		w:     w,
		label: label,
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}
	go s.run()
	return s
}

func (s *Spinner) run() {
	defer close(s.done)
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()
	i := 0
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.mu.Lock()
			label := s.label
			s.mu.Unlock()
			fmt.Fprintf(s.w, "\r  %s%s%s  %s%s",
				aCyan, spinFrames[i%len(spinFrames)], aReset,
				label, aClear,
			)
			i++
		}
	}
}

// Update sets a new spinner label mid-flight.
func (s *Spinner) Update(label string) {
	s.mu.Lock()
	s.label = label
	s.mu.Unlock()
}

// Stop halts the spinner and clears the line. Blocks until the goroutine exits.
func (s *Spinner) Stop() {
	close(s.stop)
	<-s.done
	fmt.Fprintf(s.w, "\r%s", aClear)
}
