# Data races in scenario handling: rate/limit decisions and "direct" switching are unreliable under load

**Component:** `runner/scenario.go`, `runner/runner.go`

Every request that hits a gaos path runs on its own goroutine, but the scenario state is shared
and unsynchronised. Building gaos with `-race` and sending 400 requests from 32 clients to a
path with a `rate` scenario prints 7–8 `WARNING: DATA RACE` reports. In practice:

- `Rate.Execute` and `Limit.Execute` increment their counter without synchronisation, so under
  concurrency a `"rate": 1` scenario does not reject every second request and `limit` lets more or
  fewer requests through than configured.
- `Method.Handler` increments its request counter from every request goroutine.
- When an action has `"direct"`, `Method.Execute` replaces the method's current scenario while
  other requests are reading it (its executables, `Accept`, `Ignore`, `Name`).

## Expected

- gaos is free of data races when one path is served concurrently, including while `direct`
  actions switch the path between scenarios (`go test -race` stays clean).
- `rate` and `limit` give exactly the same accept/reject counts under concurrent requests as the
  same number of sequential requests (for any configured rate or limit).
- Behaviour for sequential requests does not change.
- Keep `Method`'s embedded `Scenario` and its `runner` field: tests construct
  `&Method{Scenario: ..., runner: ...}` directly.

Only the concurrency problem is in scope; other bugs in this package are tracked separately.
