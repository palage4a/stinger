package stinger

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type FileReader[T any] struct {
	ctx    context.Context
	cancel context.CancelFunc
	mu     *sync.Mutex

	cfg FileReaderConfig

	ch chan *T

	f *os.File
	r *SafeReader
}

type SafeReader struct {
	r  *bufio.Reader
	mu *sync.Mutex
}

func (r *SafeReader) ReadBytes(delim byte) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.r.ReadBytes(delim)
}

type FileReaderConfig struct {
	P    int
	Path string
	Size uint
}

func NewFileReader[T any](ctx context.Context, cfg FileReaderConfig) *FileReader[T] {
	c, cancel := context.WithCancel(ctx)

	return &FileReader[T]{
		ctx:    c,
		cancel: cancel,
		mu:     &sync.Mutex{},
		cfg:    cfg,
		ch:     make(chan *T, cfg.Size),
	}
}

func (p *FileReader[T]) Generate() {
	for range p.cfg.P {
		go func() {
			for {
				select {
				case <-p.ctx.Done():
					return
				case p.ch <- p.read():
				}
			}
		}()
	}
}

func (p *FileReader[T]) open(path string) (*os.File, error) {
	if p.f != nil {
		return p.f, nil
	}

	var err error
	p.f, err = os.Open(path)
	if err != nil {
		return nil, err
	}

	return p.f, nil
}

func (p *FileReader[T]) reader(path string) (*SafeReader, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.r != nil {
		return p.r, nil
	}

	f, err := p.open(path)
	if err != nil {
		return nil, fmt.Errorf("file reader: open err: %w", err)
	}

	p.r = &SafeReader{r: bufio.NewReaderSize(f, 1<<20), mu: &sync.Mutex{}}

	return p.r, nil
}

func (p *FileReader[T]) read() *T {
	var e T

	r, err := p.reader(p.cfg.Path)
	if err != nil {
		fatal(fmt.Errorf("file reader: get reader err: %w", err))
	}

	b, err := r.ReadBytes('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}

		fatal(fmt.Errorf("file reader: read bytes err: %w", err))
	}

	err = json.Unmarshal(b, &e)
	if err != nil {
		fatal(fmt.Errorf("file reader: unmarshal err: %w", err))
	}

	return &e
}

func (p *FileReader[T]) Next() *T {
	return <-p.ch
}

func (p *FileReader[T]) Wait(stop bool) {
	timer := time.NewTicker(1000 * time.Millisecond)

	func() {
		for range timer.C {
			fmt.Printf("file reader buffer size: %d of %d\n", len(p.ch), cap(p.ch))
			if len(p.ch) == cap(p.ch) {
				if stop {
					p.cancel()
					close(p.ch)
				}

				return
			}
		}
	}()

	timer.Stop()
}
