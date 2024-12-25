package stinger

import (
	"testing"
)

func BenchmarkNewRequestFactory(b *testing.B) {
	tcs := []struct {
		name string
		cfg  *RequestFakerConfig
	}{
		{
			name: "nil",
			cfg: &RequestFakerConfig{
				Queue:   "queue",
				RkCount: -1,
				SkCount: -1,
			},
		},
		{
			name: "unique",
			cfg: &RequestFakerConfig{
				Queue:   "queue",
				RkCount: 0,
				SkCount: 0,
			},
		},
		{
			name: "1000keys",
			cfg: &RequestFakerConfig{
				Queue:   "queue",
				RkCount: 1000,
				SkCount: 1000,
			},
		},
	}

	for _, tc := range tcs {
		b.Run(tc.name, func(b *testing.B) {
			b.RunParallel(func(p *testing.PB) {
				for p.Next() {
					NewRequestFaker(tc.cfg)
				}
			})
		})
	}
}

func BenchmarkNext(b *testing.B) {
	tcs := []struct {
		name string
		cfg  *RequestFakerConfig
	}{
		{
			name: "unique",
			cfg: &RequestFakerConfig{
				Queue:       "queue",
				RkCount:     0,
				SkCount:     0,
				PayloadSize: 1024,
			},
		},
		{
			name: "1key",
			cfg: &RequestFakerConfig{
				Queue:       "queue",
				RkCount:     1,
				SkCount:     1,
				PayloadSize: 1024,
			},
		},
		{
			name: "1000keys",
			cfg: &RequestFakerConfig{
				Queue:       "queue",
				RkCount:     1e3,
				SkCount:     1e3,
				PayloadSize: 1024,
			},
		},
		{
			name: "1b_payload",
			cfg: &RequestFakerConfig{
				Queue:       "queue",
				PayloadSize: 1,
			},
		},
		{
			name: "1mb_payload",
			cfg: &RequestFakerConfig{
				Queue:       "queue",
				PayloadSize: 1 << 20,
			},
		},
	}

	for _, tc := range tcs {
		b.SetParallelism(10)
		b.Run(tc.name, func(b *testing.B) {
			rf := NewRequestFaker(tc.cfg)
			b.ResetTimer()
			b.RunParallel(func(p *testing.PB) {
				for p.Next() {
					_ = rf.Next()
				}
			})
		})
	}
}
