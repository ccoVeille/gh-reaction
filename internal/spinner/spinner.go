package spinner

import (
	"context"
	"fmt"
	"io"
)

type Spinner struct {
	format string
	done   chan struct{}
	tick   chan struct{}
}

func New(format string) *Spinner {
	s := &Spinner{
		format: format,
		done:   make(chan struct{}),
		tick:   make(chan struct{}),
	}
	return s
}

func (s *Spinner) Start(ctx context.Context, out io.Writer) {
	spinningCharacters := []rune("⣾⣽⣻⢿⡿⣟⣯⣷")
	go func() {
		var i int
		for {
			select {
			case <-s.done:
				return

			case <-ctx.Done():
				return
			case <-s.tick:
				i++
				pos := i % len(spinningCharacters)

				fmt.Fprint(out, "\r"+string(spinningCharacters[pos]))
				fmt.Fprintf(out, " "+s.format, i)
			}
		}
	}()
}

func (s *Spinner) Inc() {
	s.tick <- struct{}{}
}

func (s *Spinner) Done() {
	close(s.done)
	close(s.tick)
}
