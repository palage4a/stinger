package stinger

import (
	"context"
	"os"
	"testing"
	"time"
)

func BenchmarkFileWriter_write(b *testing.B) {
	f, err := os.CreateTemp("", "*")
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	defer os.Remove(f.Name())

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	rf := NewRequestFaker(&RequestFakerConfig{
		Queue:       "queue",
		RkCount:     -1,
		SkCount:     -1,
		DkCount:     -1,
		PayloadSize: 64,
	})

	p := NewFileWriter[Request](ctx, rf, FileWriterConfig{
		size: 1, // whatever
		path: f.Name(),
	},
	)

	b.ResetTimer()

	for range b.N {
		p.write()
	}
}
