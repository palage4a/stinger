package stinger

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type FileWriter[T any] struct {
	ctx context.Context
	f   Generator[*T]

	cfg FileWriterConfig

	w *bufio.Writer
}

type FileWriterConfig struct {
	Path string
	Size uint
}

func NewFileWriter[T any](ctx context.Context, f Generator[*T], cfg FileWriterConfig) *FileWriter[T] {
	return &FileWriter[T]{
		ctx: ctx,
		f:   f,
		cfg: cfg,
	}
}

func (p *FileWriter[T]) Generate() {
	ticker := time.NewTicker(1 * time.Second)
	for i := range p.cfg.Size {
		select {
		case <-p.ctx.Done():
			return
		default:
			p.write()
			select {
			case <-ticker.C:
				fmt.Printf("wrote %d records...\n", i)
			default:
			}
		}
	}
	err := p.w.Flush()
	if err != nil {
		fatal("flush err:", err)
	}

	fmt.Printf("successfully wrote %d records...\n", p.cfg.Size)
}

func (p *FileWriter[T]) writer(path string) (*bufio.Writer, error) {
	if p.w != nil {
		return p.w, nil
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	p.w = bufio.NewWriter(f)

	return p.w, nil
}

func (p *FileWriter[T]) write() {
	e := p.f.Next()

	b, err := json.Marshal(e)
	if err != nil {
		fatal("json marshal err:", err)
	}
	b = append(b, '\n')

	w, err := p.writer(p.cfg.Path)
	if err != nil {
		fatal(fmt.Errorf("get writer err: %w", err))
	}

	if _, err := w.Write(b); err != nil {
		fatal("write err:", err)
	}
}

func (p *FileWriter[T]) Next() *T {
	return nil
}

func (p *FileWriter[T]) Wait(_ bool) {
	return
}
