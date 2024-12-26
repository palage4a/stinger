package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "google.golang.org/grpc/examples/helloworld/helloworld"

	"github.com/palage4a/stinger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jaswdr/faker"
	mathrand "math/rand"
)

var (
	uriFlag      = flag.String("uri", "", "comma separated list of uris")
	procsFlag    = flag.Int("procs", 0, "number of procs")
	durationFlag = flag.Duration("d", time.Second, "test duration")
	verboseFlag  = flag.Bool("v", false, "verbose output")

	concurrencyFlag           = flag.Int("concurrency", 1, "concurrency")
	clientsFlag               = flag.Int("clients", 1, "count of grpc clients for single uri")
	grpcConnectionTimeoutFlag = flag.Duration("connection_timeout", 10*time.Second, "grpc connection timeout")
)

//nolint:funlen
func main() {
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	f := NewFaker()

	m := stinger.NewMetrics()
	runners := make([]stinger.Runnable, 0)

	gb := stinger.NewGrpcBencher(m, *concurrencyFlag, 1, *uriFlag)

	runner := NewSayHelloBencher(gb, f)
	runners = append(runners, runner)
	r := stinger.Benchmark(ctx, m, stinger.BenchmarkConfig{
		Procs:    *procsFlag,
		Duration: *durationFlag,
		Verbose:  *verboseFlag,
	}, runners...)

	select {
	case <-ctx.Done():
		stop()
	default:
	}

	r.Print()
}

type SayHelloBencher struct {
	*stinger.GrpcBencher
	g stinger.Generator[*pb.HelloRequest]
}

func NewSayHelloBencher(b *stinger.GrpcBencher, g stinger.Generator[*pb.HelloRequest]) *SayHelloBencher {
	return &SayHelloBencher{b, g}
}

func (b *SayHelloBencher) ActorSetup(ctx context.Context, id int) (stinger.Actor, error) {
	clients, err := b.CreateClients(ctx, id)
	if err != nil {
		return nil, err
	}

	greeters := NewGreeterClient(clients)
	p := stinger.NewRRContainer(greeters)

	return &SayHelloActor{p, b.g}, nil
}

type SayHelloActor struct {
	p *stinger.RRContainer[pb.GreeterClient]
	g stinger.Generator[*pb.HelloRequest]
}

func (a *SayHelloActor) Run(m *stinger.Metrics) error {
	req := a.g.Next()
	if req == nil {
		return stinger.ErrEndOfData
	}

	err := m.ObserveRequest(func() (string, bool, error) {
		_, err := a.next().SayHello(context.Background(), &pb.HelloRequest{
			Name: req.Name,
		})
		if err != nil {
			st, ok := status.FromError(err)
			if ok {
				return st.Code().String(), false, err
			}
		}

		return codes.OK.String(), true, nil
	})

	return err
}

func (a *SayHelloActor) next() pb.GreeterClient {
	return a.p.Next()
}

type Faker struct {
	f faker.Faker
}

func NewFaker() *Faker {
	src := mathrand.NewSource(time.Now().Unix())

	rf := &Faker{
		f: faker.NewWithSeed(src),
	}

	return rf
}

func (rf *Faker) name() string {
	return "Ivan"
}

func (rf *Faker) Next() *pb.HelloRequest {
	return &pb.HelloRequest{
		Name: rf.name(),
	}
}

func NewGreeterClient(conns []*grpc.ClientConn) []pb.GreeterClient {
	clients := make([]pb.GreeterClient, len(conns))
	for i, conn := range conns {
		clients[i] = pb.NewGreeterClient(conn)
	}

	return clients
}
