package render

import (
	"fmt"
	"os"
	"sync"
	"time"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner displays an animated loading indicator.
type Spinner struct {
	mu      sync.Mutex
	running bool
	stop    chan struct{}
	msg     string
}

// NewSpinner creates a new Spinner.
func NewSpinner() *Spinner {
	return &Spinner{}
}

// Start begins the spinner animation with the given message.
func (s *Spinner) Start(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	s.running = true
	s.msg = msg
	s.stop = make(chan struct{})

	go func() {
		i := 0
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-s.stop:
				// Clear the spinner line
				_, _ = fmt.Fprint(os.Stdout, "\r\033[2K")
				return
			case <-ticker.C:
				frame := spinnerFrames[i%len(spinnerFrames)]
				_, _ = fmt.Fprintf(os.Stdout, "\r\033[2K\033[1;36m%s\033[0m \033[2m%s\033[0m", frame, s.msg)
				i++
			}
		}
	}()
}

// Stop stops the spinner animation.
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	close(s.stop)
}
