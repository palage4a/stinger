package stinger

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func p[T any](v T) *T {
	return &v
}

type Request struct {
	Key   string
	Value string
}

func TestFileReaderNext(t *testing.T) {
	for _, tc := range []struct {
		name     string
		expected []*Request
	}{
		{
			name: "default",
			expected: []*Request{
				{
					Key:   "k1",
					Value: "v1",
				},
				{
					Key:   "k2",
					Value: "v2",
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, err := os.CreateTemp("", "*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(f.Name())

			for _, r := range tc.expected {
				b, err := json.Marshal(r)
				assert.Nil(t, err)
				b = append(b, '\n')

				if _, err := f.Write(b); err != nil {
					t.Fatal("write err:", err)
				}
			}

			p := NewFileReader[Request](
				context.Background(),
				FileReaderConfig{
					P:    1,
					Path: f.Name(),
					Size: 1,
				},
			)

			p.Generate()
			assert.Nil(t, err)
			assert.Equal(t, tc.expected[0], p.Next())
			assert.Equal(t, tc.expected[1], p.Next())
			assert.Nil(t, p.Next())
		})
	}
}
