# Data race: NewProducer / NewConsumer write into the caller's broker slice

**Component:** `producer.go`, `consumer_config.go`, `producer_config.go`

`go test -race` on a service that uses a dead letter topic reports a data race in
`initializeDeadLetterProducer`. The dead letter producer is built from the reader's broker list
(`cfg.Reader.Brokers`), and `NewProducer` normalizes broker addresses with `strings.TrimSpace`
**in place**, writing into the backing array of that slice. At that point the kafka-go reader is
already running and its consumer-group discovery reads the very same slice from another
goroutine.

The same in-place normalization happens when the consumer builds its kafka-go reader from
`ConsumerConfig.Reader.Brokers`, so any broker slice a user passes in (or shares between a
consumer and a producer) can be rewritten underneath whoever else holds it.

## Expected

- Neither `NewProducer` nor consumer/reader construction writes into the broker slice it was given.
  A slice shared with a running reader, or with several goroutines that build producers at the
  same time, must stay untouched and must not race.
- Broker addresses are still trimmed: the kafka-go writer and reader receive the trimmed
  addresses (`" 127.0.0.1:9092 "` → `"127.0.0.1:9092"`), for any number of brokers.
- The dead letter producer keeps using the reader's (trimmed) brokers.
