package stinger

import (
	"context"
	"fmt"
	"time"
)

type PublishGenerator struct {
	ctx    context.Context
	cancel context.CancelFunc
	rf     *RequestFaker
	p      int
	ch     chan *Request
}

func NewPublishGenerator(ctx context.Context, rf *RequestFaker, s uint, p int) *PublishGenerator {
	c, cancel := context.WithCancel(ctx)

	return &PublishGenerator{c, cancel, rf, p, make(chan *Request, s)}
}

func (g *PublishGenerator) Generate() {
	for range g.p {
		go func() {
			for {
				select {
				case <-g.ctx.Done():
					return
				case g.ch <- g.rf.Next():
				}
			}
		}()
	}
}

func (g *PublishGenerator) Next() *Request {
	return <-g.ch
}

func (g *PublishGenerator) Wait(stop bool) {
	timer := time.NewTicker(1000 * time.Millisecond)

	func() {
		for range timer.C {
			select {
			case <-g.ctx.Done():
				return
			default:
			}

			fmt.Printf("buffer size: %d of %d\n", len(g.ch), cap(g.ch))
			if len(g.ch) == cap(g.ch) {
				if stop {
					g.cancel()
					close(g.ch)
				}

				return
			}
		}
	}()

	timer.Stop()
}

type PublishBatchGenerator struct {
	ctx    context.Context
	cancel context.CancelFunc
	rf     *BatchRequestFaker
	p      int
	ch     chan []*Request
}

func NewPublishBatchGenerator(ctx context.Context, rf *BatchRequestFaker, s uint, p int) *PublishBatchGenerator {
	c, cancel := context.WithCancel(ctx)

	return &PublishBatchGenerator{c, cancel, rf, p, make(chan []*Request, s)}
}

func (g *PublishBatchGenerator) Generate() {
	for range g.p {
		go func() {
			for {
				select {
				case <-g.ctx.Done():
					return
				case g.ch <- g.rf.Next():
				}
			}
		}()
	}
}

func (g *PublishBatchGenerator) Next() []*Request {
	return <-g.ch
}

func (g *PublishBatchGenerator) Wait(stop bool) {
	timer := time.NewTicker(1000 * time.Millisecond)

	func() {
		for range timer.C {
			select {
			case <-g.ctx.Done():
				return
			default:
			}

			fmt.Printf("buffer size: %d of %d\n", len(g.ch), cap(g.ch))
			if len(g.ch) == cap(g.ch) {
				if stop {
					g.cancel()
					close(g.ch)
				}

				return
			}
		}
	}()

	timer.Stop()
}
