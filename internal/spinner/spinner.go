package spinner

import (
	"context"
	"fmt"
	"io"
)

// Spinner is a terminal spinner that provides visual feedback during long-running operations.
type Spinner struct {
	out      io.Writer
	done     chan struct{}
	tick     chan string
	maxChars int
}

// New creates a new Spinner that writes to the given output.
func New(out io.Writer) *Spinner {
	s := &Spinner{
		out:  out,
		done: make(chan struct{}),
		tick: make(chan string),
	}
	return s
}

// Start begins the spinner animation with the initial message.
func (s *Spinner) Start(ctx context.Context, str string) {
	s.print(str)
	spinningCharacters := []rune("⣾⣽⣻⢿⡿⣟⣯⣷")
	go func() {
		var pos int
		for {
			select {
			case <-s.done:
				return

			case <-ctx.Done():
				return
			case str := <-s.tick:
				pos = (pos + 1) % len(spinningCharacters)
				animated := "\r" + string(spinningCharacters[pos]) + " " + str
				s.print(animated)
			}
		}
	}()
}

// Progress updates the spinner with a new message.
func (s *Spinner) Progress(format string, args ...any) {
	s.tick <- fmt.Sprintf(format, args...)
}

// Done stops the spinner and prints the final message.
func (s *Spinner) Done(format string, args ...any) {
	s.print("\r" + fmt.Sprintf(format, args...) + "\n")
	close(s.done)
	close(s.tick)
}

// print prints the given string to the output, ensuring consistent width.
func (s *Spinner) print(str string) {
	s.maxChars = max(s.maxChars, len(str))
	fmt.Fprintf(s.out, "%*s", s.maxChars, str)
}
