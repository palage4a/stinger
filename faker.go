package stinger

import (
	"crypto/rand"
	mathrand "math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/jaswdr/faker"
)

type safeSource struct {
	mx sync.Mutex
	mathrand.Source
}

func (s *safeSource) Int63() int64 {
	s.mx.Lock()
	defer s.mx.Unlock()

	return s.Source.Int63()
}

// NewSafeSource wraps an unsafe rand.Source with a mutex to guard the random source
// against concurrent access.
func NewSafeSource(in mathrand.Source) mathrand.Source {
	return &safeSource{
		Source: in,
	}
}

type Request struct {
	Queue            string            `json:"queue"             msgpack:"queue"`
	BucketId         uint64            `json:"bucket_id"         msgpack:"bucket_id"`
	RoutingKey       *string           `json:"routing_key"       msgpack:"routing_key"`
	ShardingKey      *string           `json:"sharding_key"      msgpack:"sharding_key"`
	DeduplicationKey *string           `json:"deduplication_key" msgpack:"deduplication_key"`
	Payload          []byte            `json:"payload"           msgpack:"payload"`
	Metadata         map[string]string `json:"metadata"          msgpack:"metadata"`
}

type RequestFaker struct {
	f   faker.Faker
	cfg *RequestFakerConfig
	rks []*string
	sks []*string
	dks []*string
}

type RequestFakerConfig struct {
	Queue       string
	RkCount     int
	SkCount     int
	DkCount     int
	PayloadSize int
	MdSize      int
}

func NewRequestFaker(cfg *RequestFakerConfig) *RequestFaker {
	seed := mathrand.NewSource(time.Now().Unix())
	src := NewSafeSource(seed)
	rf := &RequestFaker{
		f:   faker.NewWithSeed(src),
		cfg: cfg,
	}

	if cfg.RkCount > 0 {
		rf.rks = make([]*string, cfg.RkCount)
		for i := range cfg.RkCount {
			s := string(rf.binbytes(16))
			rf.rks[i] = &s
		}
	}

	if cfg.SkCount > 0 {
		rf.sks = make([]*string, cfg.SkCount)
		for i := range cfg.SkCount {
			rf.sks[i] = rf.uint16str()
		}
	}

	if cfg.DkCount > 0 {
		rf.dks = make([]*string, cfg.DkCount)
		for i := range cfg.DkCount {
			rf.dks[i] = rf.uuid()
		}
	}

	return rf
}

func (rf *RequestFaker) Queue() string {
	return rf.cfg.Queue
}

func (rf *RequestFaker) BucketId() uint64 {
	return uint64(rf.f.UInt16())
}

func (rf *RequestFaker) RoutingKey() *string {
	var rk *string
	if len(rf.rks) > 0 {
		rk = rf.rks[rf.f.Generator.Intn(len(rf.rks))]
	} else if rf.cfg.RkCount == 0 {
		rk = rf.uuid()
	}

	return rk
}

func (rf *RequestFaker) ShardingKey() *string {
	var sk *string
	if len(rf.sks) > 0 {
		sk = rf.sks[rf.f.Generator.Intn(len(rf.sks))]
	} else if rf.cfg.SkCount == 0 {
		sk = rf.uint16str()
	}

	return sk
}

func (rf *RequestFaker) DeduplicationKey() *string {
	var dk *string
	if len(rf.dks) > 0 {
		dk = rf.dks[rf.f.Generator.Intn(len(rf.dks))]
	} else if rf.cfg.DkCount == 0 {
		dk = rf.uuid()
	}

	return dk
}

func (rf *RequestFaker) Payload() []byte {
	return rf.binbytes(rf.cfg.PayloadSize)
}

func (rf *RequestFaker) Metadata() map[string]string {
	md := make(map[string]string, rf.cfg.MdSize)
	for range rf.cfg.MdSize {
		key := rf.uuid()
		value := rf.uuid()
		md[*key] = *value
	}

	return md
}

func (rf *RequestFaker) Next() *Request {
	return &Request{
		Queue:            rf.Queue(),
		BucketId:         rf.BucketId(),
		RoutingKey:       rf.RoutingKey(),
		ShardingKey:      rf.ShardingKey(),
		DeduplicationKey: rf.DeduplicationKey(),
		Payload:          rf.Payload(),
		Metadata:         rf.Metadata(),
	}
}

func (rf *RequestFaker) binbytes(n int) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		fatal(err)
	}

	return b
}

func (rf *RequestFaker) uuid() *string {
	uuid := rf.f.UUID().V4()

	return &uuid
}

func (rf *RequestFaker) uint16str() *string {
	n := strconv.FormatUint(uint64(rf.f.UInt16()), 10)

	return &n
}

type BatchRequestFaker struct {
	rf        *RequestFaker
	batchSize uint
}

func NewBatchRequestFaker(rf *RequestFaker, batchSize uint) *BatchRequestFaker {
	return &BatchRequestFaker{rf, batchSize}
}

func (f *BatchRequestFaker) Next() []*Request {
	batch := make([]*Request, f.batchSize)

	for i := range f.batchSize {
		batch[i] = f.rf.Next()
	}

	return batch
}
