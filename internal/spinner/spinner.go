package spinner

import (
	"context"
	"fmt"
	"io"
)

type Spinner struct {
	cnt    int
	out    io.Writer
	format string
	done   chan struct{}
	tick   chan struct{}
}

func New(out io.Writer, format string) *Spinner {
	s := &Spinner{
		out:    out,
		format: format,
		done:   make(chan struct{}),
		tick:   make(chan struct{}),
	}
	return s
}

func (s *Spinner) Start(ctx context.Context) {
	spinningCharacters := []rune("⣾⣽⣻⢿⡿⣟⣯⣷")
	go func() {
		for {
			select {
			case <-s.done:
				return

			case <-ctx.Done():
				return
			case <-s.tick:
				s.cnt++
				pos := s.cnt % len(spinningCharacters)
				fmt.Fprint(s.out, "\r"+string(spinningCharacters[pos]))
				fmt.Fprintf(s.out, " "+s.format+"…", s.cnt)
			}
		}
	}()
}

func (s *Spinner) Inc() {
	s.tick <- struct{}{}
}

func (s *Spinner) Done() {
	fmt.Fprintf(s.out, "\r✔️ "+s.format+"       \n\n", s.cnt)
	close(s.done)
	close(s.tick)
}
