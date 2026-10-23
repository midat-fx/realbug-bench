package kafka

import (
	"bytes"
	"context"
	"errors"
	"testing"

	kcronsumer "github.com/Trendyol/kafka-cronsumer/pkg/kafka"
)

type hiddenDLQ struct {
	fail     bool
	calls    int
	received []Message
}

func (h *hiddenDLQ) Produce(_ context.Context, m Message) error {
	return h.ProduceBatch(context.Background(), []Message{m})
}

func (h *hiddenDLQ) ProduceBatch(_ context.Context, ms []Message) error {
	h.calls++
	if h.fail {
		return errors.New("broker unavailable")
	}
	h.received = append(h.received, ms...)
	return nil
}

func (h *hiddenDLQ) Close() error { return nil }

func hiddenRetried() kcronsumer.Message {
	return kcronsumer.Message{
		Topic: "payments.retry",
		Key:   []byte("p-77"),
		Value: []byte(`{"amount":10}`),
		Headers: []kcronsumer.Header{
			{Key: "x-retry-count", Value: []byte("2")},
			{Key: errMessageKey, Value: []byte("timeout")},
			{Key: "trace-id", Value: []byte("abc")},
		},
	}
}

func hiddenBase(p Producer) *base {
	b := &base{metric: &ConsumerMetric{}, logger: NewZapLogger(LogLevelDebug)}
	if p != nil {
		b.deadLetterProducer = p
	}
	return b
}

func TestHiddenRetryPreBatchCalledOncePerAttemptWithRetriedMessage(t *testing.T) {
	calls := 0
	bc := batchConsumer{
		base: hiddenBase(nil),
		preBatchFn: func(ms []*Message) []*Message {
			calls++
			if len(ms) != 1 || string(ms[0].Key) != "p-77" || string(ms[0].Value) != `{"amount":10}` {
				t.Errorf("PreBatchFn must receive the retried message, got %+v", ms)
			}
			for _, m := range ms {
				m.WriterData = 10
			}
			return ms
		},
		consumeFn: func(ms []*Message) error {
			if ms[0].WriterData != 10 {
				return errors.New("WriterData missing")
			}
			return nil
		},
	}
	if err := bc.runKonsumerFn(hiddenRetried()); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("PreBatchFn must run exactly once per retry attempt, got %d", calls)
	}
}

func TestHiddenRetryPreBatchEmptyResultIsSuccess(t *testing.T) {
	consumed := 0
	bc := batchConsumer{
		base:       hiddenBase(nil),
		preBatchFn: func([]*Message) []*Message { return []*Message{} },
		consumeFn: func([]*Message) error {
			consumed++
			return errors.New("must not be called")
		},
	}
	if err := bc.runKonsumerFn(hiddenRetried()); err != nil {
		t.Fatalf("filtered-out message must count as handled, got %v", err)
	}
	if consumed != 0 {
		t.Fatalf("BatchConsumeFn must not be called for an empty PreBatchFn result, got %d calls", consumed)
	}
}

func TestHiddenRetryPreBatchCanDeadLetterWithoutConsumeError(t *testing.T) {
	dlq := &hiddenDLQ{}
	bc := batchConsumer{
		base: hiddenBase(dlq),
		preBatchFn: func(ms []*Message) []*Message {
			ms[0].SendDirectToDeadLetter = true
			ms[0].ErrDescription = "unparseable amount"
			return ms
		},
		consumeFn: func([]*Message) error { return nil },
	}
	if err := bc.runKonsumerFn(hiddenRetried()); err != nil {
		t.Fatalf("dead-lettered message must not be retried, got %v", err)
	}
	if len(dlq.received) != 1 {
		t.Fatalf("expected 1 dead-lettered message, got %d", len(dlq.received))
	}
	got := dlq.received[0]
	if got.Topic != "" {
		t.Fatalf("Topic must be empty, got %q", got.Topic)
	}
	if !bytes.Equal(got.Key, []byte("p-77")) || !bytes.Equal(got.Value, []byte(`{"amount":10}`)) {
		t.Fatalf("key/value must be preserved, got %q/%q", got.Key, got.Value)
	}
	assertErrHeader(t, got, "unparseable amount")
	if v, _ := getHeaderValue(got, "trace-id"); v != "abc" {
		t.Fatalf("other headers must be preserved, trace-id=%q", v)
	}
	if v, _ := getHeaderValue(got, "x-retry-count"); v != "2" {
		t.Fatalf("other headers must be preserved, x-retry-count=%q", v)
	}
	n := 0
	for _, h := range got.Headers {
		if h.Key == errMessageKey {
			n++
		}
	}
	if n != 1 || len(got.Headers) != 3 {
		t.Fatalf("%s must be replaced, not duplicated: %s", errMessageKey, got.Headers.Pretty())
	}
}

func TestHiddenRetryWithoutDeadLetterProducerKeepsRetrying(t *testing.T) {
	consumeErr := errors.New("validation failed")
	bc := batchConsumer{
		base: hiddenBase(nil),
		consumeFn: func(ms []*Message) error {
			ms[0].SendDirectToDeadLetter = true
			return consumeErr
		},
	}
	if err := bc.runKonsumerFn(hiddenRetried()); !errors.Is(err, consumeErr) {
		t.Fatalf("without a dead letter producer the consume error must be returned, got %v", err)
	}
}

func TestHiddenRetryFailedDeadLetterPublishKeepsRetrying(t *testing.T) {
	consumeErr := errors.New("validation failed")
	dlq := &hiddenDLQ{fail: true}
	bc := batchConsumer{
		base: hiddenBase(dlq),
		consumeFn: func(ms []*Message) error {
			ms[0].SendDirectToDeadLetter = true
			return consumeErr
		},
	}
	err := bc.runKonsumerFn(hiddenRetried())
	if err == nil {
		t.Fatal("a message that could not be dead-lettered must be retried (non-nil error)")
	}
	if dlq.calls == 0 {
		t.Fatal("dead letter publish must be attempted")
	}
}

func TestHiddenRetryOrdinaryFailureIsNotDeadLettered(t *testing.T) {
	dlq := &hiddenDLQ{}
	bc := batchConsumer{
		base:       hiddenBase(dlq),
		preBatchFn: func(ms []*Message) []*Message { return ms },
		consumeFn: func(ms []*Message) error {
			ms[0].IsFailed = true
			ms[0].ErrDescription = "db is down"
			return errors.New("generic")
		},
	}
	err := bc.runKonsumerFn(hiddenRetried())
	if err == nil || err.Error() != "db is down" {
		t.Fatalf("expected the message ErrDescription as error, got %v", err)
	}
	if dlq.calls != 0 {
		t.Fatalf("ordinary failures must go through retry, not the dead letter topic (%d publishes)", dlq.calls)
	}
}

func TestHiddenRetryPlainConsumeErrorIsReturned(t *testing.T) {
	dlq := &hiddenDLQ{}
	consumeErr := errors.New("downstream 503")
	bc := batchConsumer{base: hiddenBase(dlq), consumeFn: func([]*Message) error { return consumeErr }}
	if err := bc.runKonsumerFn(hiddenRetried()); err == nil || err.Error() != "downstream 503" {
		t.Fatalf("expected consume error, got %v", err)
	}
	if dlq.calls != 0 {
		t.Fatalf("no dead letter publish expected, got %d", dlq.calls)
	}
}

func TestHiddenRetrySendDirectFlagWithoutFailureIsSuccess(t *testing.T) {
	dlq := &hiddenDLQ{}
	bc := batchConsumer{
		base: hiddenBase(dlq),
		consumeFn: func(ms []*Message) error {
			ms[0].SendDirectToDeadLetter = true
			return nil
		},
	}
	if err := bc.runKonsumerFn(hiddenRetried()); err != nil {
		t.Fatalf("successful consume must return nil, got %v", err)
	}
	if dlq.calls != 0 {
		t.Fatalf("nothing failed, nothing must be dead-lettered (%d publishes)", dlq.calls)
	}
}

func TestHiddenRetryWithoutPreBatchConsumesOnce(t *testing.T) {
	calls := 0
	bc := batchConsumer{consumeFn: func(ms []*Message) error {
		calls++
		if len(ms) != 1 || string(ms[0].Key) != "p-77" {
			t.Errorf("unexpected batch %+v", ms)
		}
		return nil
	}}
	if err := bc.runKonsumerFn(hiddenRetried()); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("BatchConsumeFn must be called once, got %d", calls)
	}
}
