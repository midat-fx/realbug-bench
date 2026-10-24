# realbug-bench

Coding-agent evaluation tasks cut from **real bugs in real Go projects**, in the format used by
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
| [`gaos-scenario-races`](tasks/gaos-scenario-races/issue.md) | [Trendyol/gaos@771d620](https://github.com/Trendyol/gaos/tree/771d620499118e64e048de918103bcc404bda74f): data races on rate/limit counters and on `direct` scenario switching; found and fixed in the pg-search-sync project, not fixed upstream | concurrency | 2 | 4 |

The first three are bugs fixed upstream in October 2026; the base commit is the parent of the fix
commit. These libraries are the Kafka/CDC stack used by the ledger-saga and pg-search-sync
projects. `gaos-scenario-races` comes from pg-search-sync, which uses gaos for fault injection: it
found the races (`docs/upstream.md`, reproduced with `-race`) and fixed them in a local gaos commit
that is not upstream yet. The base is the pinned upstream commit, the reference patch is the race
part of that fix (the same commit also fixes two unrelated gaos bugs, left out of this task).
Tasks from the projects' own code are added as their fixes land.

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
`evil-*.patch` must fail. The layer is the one reported by the verifier; every solution was graded
3 times locally with identical verdicts.

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

**gaos-scenario-races**

| Solution | Verdict | What it does |
|---|---|---|
| `reference` | PASS | the pg-search-sync fix: mutexes in `Rate`/`Limit`, a mutex around the `Method` scenario, atomic request counter |
| `ok-atomic-counters-rwmutex` | PASS | lock-free counters (CAS loop that increments and wraps in one step) and an `RWMutex` snapshot of the scenario |
| `evil-partial-counters-only` | FAIL hidden-tests | locks the counters, the `direct` scenario switch still races |
| `evil-hardcode-rate-one` | FAIL hidden-tests | a global toggle for `rate: 1`, the visible test's configuration |
| `evil-atomic-check-then-reset` | FAIL visible-tests | atomic increment and a separate atomic reset: no race report, but counts drift under concurrency |
| `evil-delete-visible-tests` | FAIL visible-tests | deletes the visible test file |

## Coding-agent results

Agents get the issue text and a workspace with the base commit and the visible tests
(`./bench workspace`), run headless with full permissions in a directory outside this repository,
and their diff is graded by the same verifier. Transcripts are checked for reads of this
repository (hidden tests, solutions); none were found.

| Task | Agent | Model | Time | Verdict |
|---|---|---|---|---|
| `konsumer-shared-broker-race` | Codex CLI 0.149.0-alpha.4 | gpt-5.6-terra | 6 min 49 s | **PASS** (all layers; it also edited two existing tests, which layer 2 discarded) |
| `pgcdc-truncated-pgoutput` | Codex CLI 0.149.0-alpha.4 | gpt-5.6-terra | 7 min 56 s | **PASS** (all layers; it found the Relation decoder on its own) |
| `konsumer-batch-retry-prebatch` | Codex CLI | — | — | not run: the account hit its Codex usage limit |
| `gaos-scenario-races` | — | — | — | not run yet |
| all tasks | Claude Code 2.1.220 | — | — | not run yet: the CLI's login on the run machine had expired |

Both passing patches were graded three times with the same verdict. Run 2026-10-24.

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

The harness, issues and tests are MIT. Patches under `tasks/*/solutions` modify code by Trendyol:
go-pq-cdc and kafka-konsumer (MIT), gaos (Apache-2.0). `reference.patch` files are the upstream
fixes by their authors (linked above), extended where noted; the gaos reference is the
pg-search-sync project's fix.
