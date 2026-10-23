# Batch consumer: retried messages skip PreBatchFn and ignore SendDirectToDeadLetter

**Component:** batch consumer retry path (`batch_consumer.go`, the cronsumer callback `runKonsumerFn`)

With `RetryEnabled: true`, messages that come back from the retry topic are handed to
`BatchConsumeFn` by the cronsumer callback of the batch consumer. That callback is not the same
pipeline as the main flow:

1. **`PreBatchFn` is never called on the retry path.** Our `PreBatchFn` parses the payload into
   `WriterData`; on every retry attempt `BatchConsumeFn` gets `WriterData == nil` and fails again,
   so the message burns all of its retries and ends up in the dead letter topic for no reason.
2. **`SendDirectToDeadLetter` is ignored on the retry path.** On the main flow, a message that
   failed with `SendDirectToDeadLetter = true` goes straight to the dead letter topic. On the retry
   path the flag has no effect and the message is retried `MaxRetry` times.

## Expected

For a message consumed from the retry topic by the batch consumer:

- `PreBatchFn` (when configured) runs once per retry attempt and `BatchConsumeFn` receives
  **its result** (it may return new `*Message` instances). If `PreBatchFn` returns an empty slice,
  there is nothing to consume and the attempt counts as successful.
- A message that failed (the `BatchConsumeFn` error, or its own `ErrDescription`) and has
  `SendDirectToDeadLetter = true` is produced to the dead letter topic once and is **not** retried
  (the callback returns `nil` for it). It is produced the same way the main flow does it:
  key, value and headers preserved, the `x-error-message` header set to the error
  (`ErrDescription` wins over the returned error) and replaced rather than duplicated, and `Topic`
  left empty because the dead letter producer is bound to its topic.
- If there is no dead letter producer, or publishing to the dead letter topic fails, nothing is
  lost: the callback returns an error and the message follows the normal retry flow.
- Ordinary failures (no `SendDirectToDeadLetter`) behave as today: the error is returned
  (`ErrDescription` wins) and the message is retried.
