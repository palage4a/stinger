package stinger

import (
	"context"
	"os"
	"testing"
	"time"
)

type RequestFaker struct{}

func (f *RequestFaker) Next() *Request {
	return &Request{
		Key:   "key",
		Value: "value",
	}
}

func BenchmarkFileWriter_write(b *testing.B) {
	f, err := os.CreateTemp("", "*")
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	defer os.Remove(f.Name())

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	rf := &RequestFaker{}

	p := NewFileWriter[Request](ctx, rf, FileWriterConfig{
		Size: 1, // whatever
		Path: f.Name(),
	})

	b.ResetTimer()

	for range b.N {
		p.write()
	}
}
