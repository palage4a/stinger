package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof" //nolint:gosec
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
	uriFlag               = flag.String("uri", "", "comma separated list of uris")
	procsFlag             = flag.Int("procs", 0, "number of procs")
	durationFlag          = flag.Duration("d", time.Second, "test duration")
	verboseFlag           = flag.Bool("v", false, "verbose output")
	reqsBufferSizeFlag    = flag.Uint("bs", 1<<20, "requests buffer size")
	preGenerationOnlyFlag = flag.Bool("pre_generation_only", false, "stop generation data before test")

	pprofFlag         = flag.Bool("pprof", false, "enable pprof http handler")
	pprofUriFlag      = flag.String("pprof.uri", ":6060", "uri for serving pprof http handler (default localhost:6060)")
	prometheusUriFlag = flag.String("prometheus.uri", "", "uri for serving prometheus http handler (disabled by default)")

	publishersFlag        = flag.Int("publishers", 1, "number of parallel publishers")
	publishersUriFlag     = flag.String("publishers.uri", "", "comma separated list of uris for publishers")
	publishersClientsFlag = flag.Int("publishers.clients", 1, "number of clients for each publisher uri")

	clientsFlag               = flag.Int("grpc.clients", 1, "count of grpc clients for single uri")
	grpcConnectionTimeoutFlag = flag.Duration("grpc.connection_timeout", 10*time.Second, "grpc connection timeout")
)

func fatal(a ...any) {
	fmt.Println(a...)
	os.Exit(1)
}

//nolint:funlen
func main() {
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	f := NewFaker()

	m := stinger.NewMetrics()
	if *prometheusUriFlag != "" {
		go func() {
			m.Serve()
			log.Println(http.ListenAndServe(*prometheusUriFlag, nil)) //nolint:gosec
		}()
	}

	runners := make([]stinger.Runnable, 0)

	gb := stinger.NewGrpcBencher(m, *publishersFlag, 1, *uriFlag)
	runner := NewSayHelloBencher(gb, f)
	fmt.Printf("say hello runner created\n")

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

func (rf *Faker) Name() string {
	return "Ivan"
}

func (rf *Faker) Next() *pb.HelloRequest {
	return &pb.HelloRequest{
		Name: rf.Name(),
	}
}

func NewGreeterClient(conns []*grpc.ClientConn) []pb.GreeterClient {
	clients := make([]pb.GreeterClient, len(conns))
	for i, conn := range conns {
		clients[i] = pb.NewGreeterClient(conn)
	}

	return clients
}

type SayHelloGenerator struct {
	ctx    context.Context
	cancel context.CancelFunc
	rf     *Faker
	p      int
	ch     chan *pb.HelloRequest
}

func NewSayHelloGenerator(ctx context.Context, rf *Faker, s uint, p int) *SayHelloGenerator {
	c, cancel := context.WithCancel(ctx)

	return &SayHelloGenerator{c, cancel, rf, p, make(chan *pb.HelloRequest, s)}
}

func (g *SayHelloGenerator) Generate() {
	for range g.p {
		go func() {
			for {
				select {
				case <-g.ctx.Done():
					return
				case g.ch <- g.rf.Next():
				}
			}
		}()
	}
}

func (g *SayHelloGenerator) Next() *pb.HelloRequest {
	return <-g.ch
}

func (g *SayHelloGenerator) Wait(stop bool) {
	timer := time.NewTicker(1000 * time.Millisecond)

	func() {
		for range timer.C {
			select {
			case <-g.ctx.Done():
				return
			default:
			}

			fmt.Printf("buffer size: %d of %d\n", len(g.ch), cap(g.ch))
			if len(g.ch) == cap(g.ch) {
				if stop {
					g.cancel()
					close(g.ch)
				}

				return
			}
		}
	}()

	timer.Stop()
}
