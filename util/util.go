package util

import (
	"io"
	"math/rand"
	"sync"

	"github.com/palage4a/stinger/metrics"
)

type RRContainer[T any] struct {
	next int
	pool []T
}

func NewRRContainer[T any](pool []T) *RRContainer[T] {
	return &RRContainer[T]{
		0, pool,
	}
}

func (p *RRContainer[T]) Next() T {
	var cur T
	if len(p.pool) != 0 {
		cur = p.pool[p.next%len(p.pool)]
		p.next++
	}

	return cur
}

type SafeRRContainer[T any] struct {
	c  *RRContainer[T]
	mu *sync.Mutex
}

func NewSafeRRContainer[T any](c *RRContainer[T]) *SafeRRContainer[T] {
	return &SafeRRContainer[T]{
		c,
		&sync.Mutex{},
	}
}

func (p *SafeRRContainer[T]) Next() T {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.c.Next()
}

func MultiplySlice[T any](s []T, c int) []T {
	var out []T
	if c == 0 || len(s) == 0 {
		return out
	}

	out = make([]T, len(s)*c)
	i := 0
	for _, v := range s {
		for range c {
			out[i] = v
			i++
		}
	}

	return out
}

func Shuffle[T any](s []T) {
	rand.Shuffle(len(s), func(i, j int) { s[i], s[j] = s[j], s[i] })
}

func SplitSlice[T any](s []T, sliceSize int) [][]T {
	var slices [][]T
	if len(s) == 0 {
		return slices
	}

	if len(s) < sliceSize {
		return [][]T{s}
	}

	if sliceSize < 2 {
		sliceSize = 1
	}

	slices = make([][]T, len(s)/sliceSize)
	if len(s)%sliceSize > 0 {
		slices = append(slices, nil) //nolint:makezero
	}

	offset := 0
	for i := range slices {
		var ss []T
		if offset+sliceSize > len(s) {
			ss = s[offset:]
		} else {
			ss = s[offset : offset+sliceSize]
		}

		slices[i] = ss
		offset += sliceSize
	}

	return slices
}

func ObservingSizeRead(m *metrics.Metrics, r io.Reader, b []byte) (int, error) {
	n, err := r.Read(b)
	if err != nil {
		return 0, err
	}
	//nolint:gosec
	m.AddReceivedBytes(uint64(n))

	return n, err
}

func ObservingSizeWrite(m *metrics.Metrics, w io.Writer, b []byte) (int, error) {
	n, err := w.Write(b)
	if err != nil {
		return 0, err
	}
	//nolint:gosec
	m.AddSentBytes(uint64(n))

	return n, err
}
