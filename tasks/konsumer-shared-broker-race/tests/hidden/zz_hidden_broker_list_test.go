package kafka

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

func hiddenBrokers() []string {
	return []string{" 127.0.0.1:1", "127.0.0.2:1 ", "\t127.0.0.3:1\n"}
}

func TestHiddenNewProducerKeepsCallerSliceAndTrims(t *testing.T) {
	brokers := hiddenBrokers()
	p, err := NewProducer(&ProducerConfig{Writer: WriterConfig{Brokers: brokers, Topic: "t"}})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	if !reflect.DeepEqual(brokers, hiddenBrokers()) {
		t.Fatalf("NewProducer wrote into the caller's broker slice: %q", brokers)
	}
	w, ok := p.(*producer).w.(*kafka.Writer)
	if !ok {
		t.Fatalf("unexpected writer type %T", p.(*producer).w)
	}
	if got := w.Addr.String(); got != "127.0.0.1:1,127.0.0.2:1,127.0.0.3:1" {
		t.Fatalf("writer must use trimmed brokers, got %q", got)
	}
}

func TestHiddenNewKafkaReaderKeepsCallerSliceAndTrims(t *testing.T) {
	brokers := hiddenBrokers()
	// no GroupID: a partition reader does not dial until it is read from, so Close returns at once
	cfg := ConsumerConfig{Reader: ReaderConfig{Brokers: brokers, Topic: "t"}}
	r, err := cfg.newKafkaReader(NewZapLogger(LogLevelError))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	if !reflect.DeepEqual(brokers, hiddenBrokers()) {
		t.Fatalf("newKafkaReader wrote into the caller's broker slice: %q", brokers)
	}
	got := r.(*readerWrapper).r.Config().Brokers
	if !reflect.DeepEqual(got, []string{"127.0.0.1:1", "127.0.0.2:1", "127.0.0.3:1"}) {
		t.Fatalf("reader must use trimmed brokers, got %q", got)
	}
}

// The broker slice is shared with a reader that is already running; building producers from it
// concurrently must not race (the grader runs this with -race).
func TestHiddenConcurrentProducersOnSharedBrokerSlice(t *testing.T) {
	shared := hiddenBrokers()
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: shared,
		GroupID: "g",
		Topic:   "t",
		Dialer:  &kafka.Dialer{Timeout: 10 * time.Millisecond},
	})
	defer reader.Close()

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				p, err := NewProducer(&ProducerConfig{Writer: WriterConfig{Brokers: shared, Topic: "dlq"}})
				if err != nil {
					t.Error(err)
					return
				}
				_ = p.Close()
				time.Sleep(2 * time.Millisecond)
			}
		}()
	}
	wg.Wait()
	if !reflect.DeepEqual(shared, hiddenBrokers()) {
		t.Fatalf("shared broker slice changed: %q", shared)
	}
}

func TestHiddenDeadLetterProducerUsesTrimmedReaderBrokers(t *testing.T) {
	cfg := ConsumerConfig{
		DeadLetterTopic: "dlq",
		Reader:          ReaderConfig{Brokers: hiddenBrokers(), GroupID: "g", Topic: "t"},
	}
	p, err := initializeDeadLetterProducer(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if !reflect.DeepEqual(cfg.Reader.Brokers, hiddenBrokers()) {
		t.Fatalf("reader broker list changed: %q", cfg.Reader.Brokers)
	}
	if got := p.(*producer).w.(*kafka.Writer).Addr.String(); got != "127.0.0.1:1,127.0.0.2:1,127.0.0.3:1" {
		t.Fatalf("dead letter producer must use trimmed brokers, got %q", got)
	}
}
