package stinger

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"os"
	"strconv"
	"testing"
	"time"
)

func p[T any](v T) *T {
	return &v
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
					Queue:            "queue",
					BucketId:         12,
					RoutingKey:       p("rk"),
					ShardingKey:      p("sk"),
					DeduplicationKey: p("dk"),
					Payload:          []byte("qwer"),
					Metadata: map[string]string{
						"x-timestamp": strconv.FormatInt(time.Now().UnixNano(), 10),
					},
				},
				{
					Queue:            "queue",
					BucketId:         13,
					RoutingKey:       p("rk"),
					ShardingKey:      p("sk"),
					DeduplicationKey: p("dk"),
					Payload:          []byte("qwer"),
					Metadata: map[string]string{
						"x-timestamp": strconv.FormatInt(time.Now().UnixNano(), 10),
					},
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
					p:    1,
					path: f.Name(),
					size: 1,
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
