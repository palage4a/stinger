package stinger

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"
)

var ErrEndOfData = errors.New("end of data")

type Actor interface {
	Run(*Metrics) error
}

type Runnable interface {
	SetUp(context.Context)
	Parallelism() int
	ActorSetup(context.Context, int) (Actor, error)
}

type BaseGenerator interface {
	Generate()
	Wait(bool)
}

type Generator[T any] interface {
	Next() T
}

type BenchmarkConfig struct {
	Procs    int
	Duration time.Duration
	Verbose  bool
}

func Benchmark(ctx context.Context, m *Metrics, cfg BenchmarkConfig, runners ...Runnable) *Result {
	runtime.GOMAXPROCS(cfg.Procs)

	wg := &sync.WaitGroup{}
	gCtx, cancel := context.WithTimeout(ctx, cfg.Duration)
	defer cancel()

	for _, r := range runners {
		r.SetUp(ctx)
	}

	m.StartTimer()
	for _, r := range runners {
		for i := range r.Parallelism() {
			wg.Add(1)
			go func(ctx context.Context) {
				defer wg.Done()

				actor, err := r.ActorSetup(ctx, i)
				if err != nil {
					fatal(err)
				}

				for {
					select {
					case <-ctx.Done():
						return
					default:
					}

					err := actor.Run(m)
					if err != nil {
						if errors.Is(err, ErrEndOfData) {
							return
						}

						if cfg.Verbose {
							fmt.Printf("run err: %s\n", err)
						}
					}
				}
			}(gCtx)
		}
	}
	wg.Wait()
	m.StopTimer()

	return m.Result()
}

func fatal(a ...any) {
	fmt.Println(a...)
	os.Exit(1)
}
