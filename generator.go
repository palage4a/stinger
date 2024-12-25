package stinger

import (
	"context"
	"fmt"
	"time"
)

type BufferedFaker[T any] struct {
	ctx    context.Context
	cancel context.CancelFunc
	f      Generator[T]
	p      int
	ch     chan T
}

func NewBufferedFaker[T any](ctx context.Context, f Generator[T], s uint, p int) *BufferedFaker[T] {
	c, cancel := context.WithCancel(ctx)

	return &BufferedFaker[T]{c, cancel, f, p, make(chan T, s)}
}

func (b *BufferedFaker[T]) Generate() {
	for range b.p {
		go func() {
			for {
				select {
				case <-b.ctx.Done():
					return
				case b.ch <- b.f.Next():
				}
			}
		}()
	}
}

func (b *BufferedFaker[T]) Next() T {
	return <-b.ch
}

func (b *BufferedFaker[T]) Wait(stop bool) {
	timer := time.NewTicker(1000 * time.Millisecond)

	func() {
		for range timer.C {
			select {
			case <-b.ctx.Done():
				return
			default:
			}

			fmt.Printf("buffer size: %d of %d\n", len(b.ch), cap(b.ch))
			if len(b.ch) == cap(b.ch) {
				if stop {
					b.cancel()
					close(b.ch)
				}

				return
			}
		}
	}()

	timer.Stop()
}
