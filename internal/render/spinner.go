package render

import (
	"fmt"
	"io"
	"sync"
	"time"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner displays an animated loading indicator on its writer.
type Spinner struct {
	out     io.Writer
	mu      sync.Mutex
	running bool
	stop    chan struct{}
}

// NewSpinner creates a Spinner that draws to out.
func NewSpinner(out io.Writer) *Spinner {
	return &Spinner{out: out}
}

// Start begins the spinner animation with the given message. If a spinner is
// already running, this is a no-op.
func (s *Spinner) Start(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	s.running = true
	s.stop = make(chan struct{})

	stop := s.stop
	out := s.out
	go func() {
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()

		for i := 0; ; i++ {
			select {
			case <-stop:
				fmt.Fprint(out, "\r\033[2K")
				return
			case <-ticker.C:
				frame := spinnerFrames[i%len(spinnerFrames)]
				fmt.Fprintf(out, "\r\033[2K\033[1;36m%s\033[0m \033[2m%s\033[0m", frame, msg)
			}
		}
	}()
}

// Stop stops the spinner animation. Safe to call when not running.
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	close(s.stop)
}
