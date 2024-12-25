package stinger

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
)

type LatencyPercentile struct {
	Success    bool
	Percentile int
	Value      time.Duration
}

type Response struct {
	Code    string
	Success bool
	Count   int64
}

type Metrics struct {
	// NOTE: enabled introduced as a hack for not observing traffic before test start
	enabled       bool
	latency       *prometheus.SummaryVec
	delivery      prometheus.Summary
	requests      prometheus.Counter
	messages      prometheus.Counter
	responses     *prometheus.CounterVec
	consumed      prometheus.Counter
	sentBytes     prometheus.Gauge
	receivedBytes prometheus.Gauge

	start    time.Time
	duration time.Duration
}

func NewMetrics() *Metrics {
	m := new(Metrics)

	m.latency = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Name:       "latency",
		Help:       "request latency",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
	}, []string{"success"})

	m.requests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "requests_total",
		Help: "total requests number (grpc/iproto)",
	})

	m.responses = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "responses_total",
		Help: "total response number (grpc/iproto)",
	}, []string{"code", "success"})

	m.sentBytes = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sent_bytes",
		Help: "sent bytes from client to service",
	})

	m.receivedBytes = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "received_bytes",
		Help: "received bytes from client to service",
	})

	return m
}

func (m *Metrics) Enable() {
	m.enabled = true
}

func (m *Metrics) Disable() {
	m.enabled = false
}

func (m *Metrics) StartTimer() {
	m.start = time.Now()
}

func (m *Metrics) StopTimer() {
	m.duration = time.Since(m.start)
}

func (m *Metrics) Requests() int64 {
	var metric dto.Metric

	err := m.requests.Write(&metric)
	if err != nil {
		panic(err)
	}

	return int64(metric.GetCounter().GetValue())
}

func (m *Metrics) IncReq(i int64) {
	m.requests.Add(float64(i))
}

func (m *Metrics) ObserveRequest(f func() (string, bool, error)) error {
	s := time.Now()
	m.IncReq(1)
	code, success, err := f()
	m.latency.WithLabelValues(strconv.FormatBool(success)).Observe(float64(time.Since(s).Nanoseconds()))
	m.IncResponses(code, success, 1)

	return err
}

func (m *Metrics) SentBytes() uint64 {
	var metric dto.Metric

	err := m.sentBytes.Write(&metric)
	if err != nil {
		panic(err)
	}

	return uint64(metric.GetGauge().GetValue())
}

func (m *Metrics) AddSentBytes(i uint64) {
	if m.enabled {
		m.sentBytes.Add(float64(i))
	}
}

func (m *Metrics) ReceivedBytes() uint64 {
	var metric dto.Metric

	err := m.receivedBytes.Write(&metric)
	if err != nil {
		panic(err)
	}

	return uint64(metric.GetGauge().GetValue())
}

func (m *Metrics) AddReceivedBytes(i uint64) {
	if m.enabled {
		m.receivedBytes.Add(float64(i))
	}
}

func (m *Metrics) Latency() ([]LatencyPercentile, error) {
	ch := make(chan prometheus.Metric)
	go func() {
		m.latency.Collect(ch)
		close(ch)
	}()

	res := make([]LatencyPercentile, 0)
	for m := range ch {
		var metric dto.Metric

		err := m.Write(&metric)
		if err != nil {
			return nil, fmt.Errorf("get latency err: %w", err)
		}
		summary := metric.GetSummary()

		if summary.GetSampleSum() > 0 {
			quantiles := summary.GetQuantile()

			var success bool
			var err error
			for _, l := range metric.Label {
				switch l.GetName() {
				case "success":
					success, err = strconv.ParseBool(l.GetValue())
					if err != nil {
						return nil, fmt.Errorf("get latency err: %w", err)
					}
				default:
				}
			}

			for _, q := range quantiles {
				v := time.Duration(q.GetValue())
				res = append(res, LatencyPercentile{
					Success:    success,
					Percentile: int(q.GetQuantile() * 100),
					Value:      v,
				})
			}
		}
	}

	return res, nil
}

func (m *Metrics) Delivery() (*dto.Metric, error) {
	var metric dto.Metric
	err := m.delivery.Write(&metric)
	if err != nil {
		return nil, err
	}

	return &metric, nil
}

func (m *Metrics) IncResponses(code string, success bool, i int64) {
	m.responses.WithLabelValues(code, strconv.FormatBool(success)).Add(float64(i))
}

func (m *Metrics) Responses() []Response {
	ch := make(chan prometheus.Metric)
	go func() {
		m.responses.Collect(ch)
		close(ch)
	}()

	res := make([]Response, 0)
	for m := range ch {
		var metric dto.Metric

		err := m.Write(&metric)
		if err != nil {
			panic(err)
		}

		resp := Response{}
		resp.Count = int64(metric.GetCounter().GetValue())

		for _, l := range metric.Label {
			switch l.GetName() {
			case "code":
				resp.Code = l.GetValue()
			case "success":
				b, err := strconv.ParseBool(l.GetValue())
				if err != nil {
					panic(fmt.Errorf("strconv parse bool err: %s", err))
				}
				resp.Success = b
			}
		}

		res = append(res, resp)
	}

	return res
}

func (m *Metrics) Result() *Result {
	latency, err := m.Latency()
	if err != nil {
		panic(err)
	}

	return &Result{
		latency:       latency,
		requests:      m.Requests(),
		responses:     m.Responses(),
		duration:      m.duration,
		sentBytes:     m.SentBytes(),
		receivedBytes: m.ReceivedBytes(),
	}
}

type Result struct {
	latency       []LatencyPercentile
	duration      time.Duration
	requests      int64
	responses     []Response
	sentBytes     uint64
	receivedBytes uint64
}

func getSpacer(s string, l int) string {
	b := make([]byte, l)
	for i := range l {
		b[i] = '.'
	}

	return string(b[len(s):])
}

func (r *Result) Print() {
	fmt.Println("\nRESULTS:")
	fmt.Printf("elapsed ....................... %s\n", r.duration)

	responsesCount := int64(0)
	errorsCount := int64(0)
	for _, r := range r.responses {
		responsesCount += r.Count
		if !r.Success {
			errorsCount += r.Count
		}
	}

	if r.requests > 0 {
		fmt.Println("\nREQUESTS:")
		fmt.Printf("responses ..................... %d\n", responsesCount)
		fmt.Printf("errors ........................ %d\n", errorsCount)
		fmt.Printf("total ......................... %d\n", r.requests)
		fmt.Printf("throughput .................... %0.2f %s\n", float64(r.requests)/r.duration.Seconds(), "req/s")

		failedRequests := make([]LatencyPercentile, 0)
		successedRequests := make([]LatencyPercentile, 0)
		for _, r := range r.latency {
			if r.Success {
				successedRequests = append(successedRequests, r)
			} else {
				failedRequests = append(failedRequests, r)
			}
		}

		if len(successedRequests) > 0 {
			fmt.Printf("SUCCESSED:\n")
			for _, p := range successedRequests {
				fmt.Printf("  latency p(%d) ................. %s\n", p.Percentile, p.Value)
			}
		}

		if len(failedRequests) > 0 {
			fmt.Printf("FAILED:\n")
			for _, p := range failedRequests {
				fmt.Printf("  latency p(%d) ................. %s\n", p.Percentile, p.Value)
			}
		}
	}

	fmt.Println("\nCODES:")
	for _, r := range r.responses {
		fmt.Printf("%s %s %d\n", r.Code, getSpacer(r.Code, 30), r.Count)
	}

	data := r.receivedBytes + r.sentBytes
	if data > 0 {
		fmt.Println("\nDATA:")
		fmt.Printf("sent .......................... %s\n", ByteCountIEC(r.sentBytes))
		fmt.Printf("received ...................... %s\n", ByteCountIEC(r.receivedBytes))
		fmt.Printf("total ......................... %s\n", ByteCountIEC(data))
		fmt.Printf("throughput .................... %s/s\n", ByteCountIEC(uint64(float64(data)/r.duration.Seconds())))
	}
}

func (m *Metrics) Serve() {
	http.Handle("/metrics", promhttp.Handler())
}
