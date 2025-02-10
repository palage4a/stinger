package stinger

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/palage4a/stinger/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var clientMetrics *grpcprom.ClientMetrics

func init() {
	clientMetrics = grpcprom.NewClientMetrics()
	prometheus.MustRegister(clientMetrics)
}

type GrpcBencher struct {
	uris        []string
	m           *metrics.Metrics
	parallelism int
	clients     int

	slices [][]string
}

func NewGrpcBencher(m *metrics.Metrics, parallelism int, clients int, uri string) *GrpcBencher {
	uris := strings.Split(uri, ",")

	return &GrpcBencher{uris, m, parallelism, clients, nil}
}

func (b *GrpcBencher) SetUp(_ context.Context) {
	uris := MultiplySlice(b.uris, b.clients*b.parallelism)
	Shuffle(uris)
	b.slices = SplitSlice(uris, b.clients)

	b.m.Enable()
}

func (b *GrpcBencher) Parallelism() int {
	return b.parallelism
}

func (b *GrpcBencher) CreateClients(ctx context.Context, id int) ([]*grpc.ClientConn, error) {
	conCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // FIXME: hardcode
	defer cancel()

	conns, err := NewGrpcConnections(conCtx, b.slices[id%len(b.slices)], b.m)
	if err != nil {
		return nil, err
	}

	return conns, err
}

func newGrpcClient(_ context.Context, uri string, m *metrics.Metrics) (*grpc.ClientConn, error) {
	return grpc.NewClient(
		uri,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(sizeObservation(m)),
		grpc.WithChainUnaryInterceptor(
			clientMetrics.UnaryClientInterceptor(),
		),
		grpc.WithChainStreamInterceptor(
			clientMetrics.StreamClientInterceptor(),
		),
	)
}

func NewGrpcConnections(ctx context.Context, uris []string, m *metrics.Metrics) ([]*grpc.ClientConn, error) {
	conns := make([]*grpc.ClientConn, len(uris))
	for i, u := range uris {
		// NOTE: make client creation blocking
		conn, err := newGrpcClient(ctx, u, m)
		if err != nil {
			return nil, fmt.Errorf("grpc new client for %s: %w", u, err)
		}

		conns[i] = conn
	}

	return conns, nil
}

type SizeObserverNetConn struct {
	c net.Conn
	m *metrics.Metrics
}

func (c *SizeObserverNetConn) Read(b []byte) (int, error) {
	return observingSizeRead(c.m, c.c, b)
}

func (c *SizeObserverNetConn) Write(b []byte) (int, error) {
	return observingSizeWrite(c.m, c.c, b)
}

func (c *SizeObserverNetConn) Close() error {
	return c.c.Close()
}

func (c *SizeObserverNetConn) LocalAddr() net.Addr {
	return c.c.LocalAddr()
}

func (c *SizeObserverNetConn) RemoteAddr() net.Addr {
	return c.c.RemoteAddr()
}

func (c *SizeObserverNetConn) SetDeadline(t time.Time) error {
	return c.c.SetDeadline(t)
}

func (c *SizeObserverNetConn) SetReadDeadline(t time.Time) error {
	return c.c.SetDeadline(t)
}

func (c *SizeObserverNetConn) SetWriteDeadline(t time.Time) error {
	return c.c.SetWriteDeadline(t)
}

// NOTE: very dirty.
func sizeObservation(m *metrics.Metrics) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, addr string) (net.Conn, error) {
		c, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, err
		}

		return &SizeObserverNetConn{c, m}, nil
	}
}
