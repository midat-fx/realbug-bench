package kafka

import (
	"errors"
	"testing"

	kcronsumer "github.com/Trendyol/kafka-cronsumer/pkg/kafka"
)

func retriedMessage() kcronsumer.Message {
	return kcronsumer.Message{
		Topic: "orders.retry",
		Key:   []byte("order-1"),
		Value: []byte(`{"id":1}`),
		Headers: []kcronsumer.Header{
			{Key: "x-retry-count", Value: []byte("1")},
			{Key: errMessageKey, Value: []byte("previous error")},
		},
	}
}

func Test_batchConsumer_retry_runs_PreBatchFn(t *testing.T) {
	var seen interface{}
	bc := batchConsumer{
		preBatchFn: func(messages []*Message) []*Message {
			return []*Message{{Key: messages[0].Key, Value: messages[0].Value, WriterData: "parsed"}}
		},
		consumeFn: func(messages []*Message) error {
			seen = messages[0].WriterData
			return nil
		},
	}

	if err := bc.runKonsumerFn(retriedMessage()); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if seen != "parsed" {
		t.Fatalf("BatchConsumeFn must see the WriterData set by PreBatchFn, got %v", seen)
	}
}

func Test_batchConsumer_retry_honors_SendDirectToDeadLetter(t *testing.T) {
	dlq := &mockDeadLetterProducer{}
	bc := batchConsumer{
		base: &base{metric: &ConsumerMetric{}, logger: NewZapLogger(LogLevelDebug), deadLetterProducer: dlq},
		consumeFn: func(messages []*Message) error {
			messages[0].SendDirectToDeadLetter = true
			return errors.New("schema mismatch")
		},
	}

	if err := bc.runKonsumerFn(retriedMessage()); err != nil {
		t.Fatalf("message was dead-lettered, retry callback must return nil, got %v", err)
	}
	if len(dlq.received) != 1 {
		t.Fatalf("expected one dead-lettered message, got %d", len(dlq.received))
	}
	if dlq.received[0].Topic != "" {
		t.Fatalf("dead-lettered message must leave Topic empty, got %q", dlq.received[0].Topic)
	}
	assertErrHeader(t, dlq.received[0], "schema mismatch")
}
