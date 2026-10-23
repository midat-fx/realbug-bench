# realbug-bench

Coding-agent evaluation tasks cut from **real bugs fixed in real Go projects**, in the format used by
agent-evaluation platforms (micro1, Mercor, AfterQuery): a maintainer-style issue, a pinned Docker
image of the repository at the commit before the fix, a reference patch, visible and hidden tests,
and a verifier that grades any patch.

Every task ships a set of **adversarial ("evil") solutions** that the verifier must reject and at
least **two different valid solutions** it must accept. CI grades all of them on every push, twice,
and fails if any verdict is wrong or changes between runs.

```bash
./bench check            # grade every solution of every task (needs Docker); exit 0 = all verdicts as expected
```

## Tasks

| Task | Upstream bug | Kind | Valid solutions | Evil solutions |
|---|---|---|---|---|
| [`pgcdc-truncated-pgoutput`](tasks/pgcdc-truncated-pgoutput/issue.md) | [Trendyol/go-pq-cdc@6110b91](https://github.com/Trendyol/go-pq-cdc/commit/6110b91596ea9597b588954f94a89a1b5f4cfcd6): pgoutput decoders panic on truncated messages | input hardening | 2 | 4 |
| [`konsumer-batch-retry-prebatch`](tasks/konsumer-batch-retry-prebatch/issue.md) | [Trendyol/kafka-konsumer@0555952](https://github.com/Trendyol/kafka-konsumer/commit/055595218bb0d7bd99f45bdafc7b5cbe39dbf639): retry path skips `PreBatchFn`, ignores `SendDirectToDeadLetter` | behaviour / message routing | 2 | 4 |
| [`konsumer-shared-broker-race`](tasks/konsumer-shared-broker-race/issue.md) | [Trendyol/kafka-konsumer@d80db5c](https://github.com/Trendyol/kafka-konsumer/commit/d80db5c55df7fa8c83e1da30b5bbae10c9c9720a): data race, broker slice trimmed in place | concurrency | 2 | 5 |

All three are bugs fixed upstream in October 2026; the base commit of each task is the parent of
the fix commit. These libraries are the Kafka/CDC stack used by the
[ledger-saga](https://github.com/midat-fx/ledger-saga) project; tasks cut from ledger-saga itself
and its sibling projects are added as their fixes land.

**A note on `pgcdc-truncated-pgoutput`.** The upstream fix bounds-checks Begin, Commit, tuple data
and the TOAST copy in Update, but not the Relation decoder: a Relation message cut after the table
name still panics at the fix commit and at upstream HEAD (checked 2026-10-24). The issue asks for
every decoder reachable from `message.New`, so the reference patch is the upstream fix plus the
Relation bounds checks, and the bare upstream fix is graded as an evil (incomplete) solution. It
fails exactly on the hidden Relation tests.

## Verification layers

`harness/grade.sh` runs inside the task image with `--network none`. A patch passes only if every
layer passes; the first failing layer is the verdict.

| # | Layer | Why it exists |
|---|---|---|
| 1 | **Patch applies** to the agent's workspace (base commit + visible tests) | Graders must see exactly what the agent produced; nothing is re-based or fixed up. |
| 2 | **Test isolation**: every change the patch makes to `*_test.go` / `testdata` is discarded and the hidden tests are copied in | Editing or deleting tests is the cheapest cheat. Discarding the edits (instead of rejecting the patch) means legitimate extra tests cost nothing, and a solution that only works because it rewrote a test fails on the real one. |
| 3 | **Build** (`go build ./...`) | A patch that breaks another package fails here, not as a confusing test error. |
| 4 | **Visible tests**, `-race`, `-count=5` | The tests the agent could run. Race detector and repetition turn "passes once" into "passes reliably". |
| 5 | **Hidden tests**, `-race`, `-count=5` | Catch hard-coding to the visible fixtures and partial fixes: different inputs, more brokers, every truncation point of every message type, the failure branches the issue describes but the visible tests do not exercise. |
| 6 | **Upstream regression suite** | The fix must not break existing behaviour. The tests are the project's own, unchanged. |

Images are pinned by digest (`golang:1.25.14-bookworm@sha256:3b4a…`) and the repository by commit;
all modules are downloaded at build time, so grading is offline and repeatable.

## Solutions and what catches them

Expected verdicts come from file names: `reference.patch` and `ok-*.patch` must pass,
`evil-*.patch` must fail. The layer is the one reported by the verifier (3 runs each, identical).

**pgcdc-truncated-pgoutput**

| Solution | Verdict | What it does |
|---|---|---|
| `reference` | PASS | upstream fix + Relation bounds checks |
| `ok-cursor-reader` | PASS | different design: a bounds-checked `tuple.Cursor` used by the tuple and Relation decoders |
| `evil-partial-upstream-only` | FAIL hidden-tests | the real upstream fix, which misses the Relation decoder |
| `evil-hardcode-length-threshold` | FAIL hidden-tests | rejects tuple data shorter than 12 bytes, tuned to the visible fixtures |
| `evil-recover-in-new` | FAIL visible-tests | `recover()` in `message.New` only; the decoders still panic when called directly |
| `evil-edit-visible-tests` | FAIL visible-tests | reverts the visible tests to the old expectations, no code change |

**konsumer-batch-retry-prebatch**

| Solution | Verdict | What it does |
|---|---|---|
| `reference` | PASS | upstream fix |
| `ok-partition-and-join` | PASS | partitions the batch into dead-letter and retry sets, joins retry errors with `errors.Join` |
| `evil-drop-on-dlq-failure` | FAIL hidden-tests | returns `nil` for flagged messages even when dead-lettering is impossible or failed: the message is lost |
| `evil-dlq-every-failure` | FAIL hidden-tests | dead-letters every failed retry, flag or not |
| `evil-hardcode-error-text` | FAIL hidden-tests | dead-letters only when the error text equals the visible fixture's |
| `evil-delete-visible-tests` | FAIL visible-tests | deletes the visible test file |

**konsumer-shared-broker-race**

| Solution | Verdict | What it does |
|---|---|---|
| `reference` | PASS | upstream fix: normalization builds a new slice |
| `ok-clone-at-call-sites` | PASS | different location: `slices.Clone` at the two call sites, normalization unchanged |
| `evil-clone-in-dlq-only` | FAIL hidden-tests | copies the slice only in `initializeDeadLetterProducer`, the path the visible test exercises |
| `evil-partial-writer-only` | FAIL hidden-tests | fixes the producer side, the reader still trims in place |
| `evil-hardcode-single-broker` | FAIL hidden-tests | copies only when there is exactly one broker, like the visible fixture |
| `evil-mutex-around-trim` | FAIL visible-tests | guards the in-place write with a mutex the reader never takes; the race detector still fires |
| `evil-delete-visible-tests` | FAIL visible-tests | deletes the visible test file |

## Coding-agent results

AGENT_RESULTS_PLACEHOLDER

## Layout

```
bench                       driver: check | verify | workspace | agent | build
harness/grade.sh            verifier, runs inside the task container
tasks/<id>/
  issue.md                  the problem statement the agent gets
  Dockerfile                pinned image: repository at the base commit, modules pre-fetched
  task.env                  upstream, base/fix commits, test packages, visible-test selector
  tests/visible/            copied into the agent's workspace
  tests/hidden/             never shown to the agent; hidden tests are named TestHidden*
  solutions/*.patch         reference, ok-*, evil-*
results/agents/<task>/<agent>-<time>/
  candidate.patch           the agent's diff
  grade.log, result.json    verifier output and verdict
```

Other commands:

```bash
./bench verify <task> <patch>        # grade any patch
./bench workspace <task> <dir>       # what an agent gets: upstream at the base commit + visible tests
./bench agent <task> claude|codex    # run an agent headless on the task and grade its diff
```

## License

The harness, issues and tests are MIT. Patches under `tasks/*/solutions` modify MIT-licensed code
by Trendyol (go-pq-cdc, kafka-konsumer); `reference.patch` files are the upstream fixes by their
authors (linked above), extended where noted.
