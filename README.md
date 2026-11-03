# realbug-bench

Coding-agent evaluation tasks cut from **real bugs in real Go, Python and TypeScript projects**, in the format used by
agent-evaluation platforms (micro1, Mercor, AfterQuery): a maintainer-style issue, a pinned Docker
image of the repository at the commit before the fix, a reference patch, visible and hidden tests,
and a verifier that grades any patch.

Every task ships a set of **adversarial ("evil") solutions** that the verifier must reject and at
least **two different valid solutions** it must accept. CI grades all of them on every push, twice,
and fails if any verdict is wrong or changes between runs.

```bash
make test                # lint, leak-scan and self-test the harness, grade every solution of every task twice (needs Docker)
make test TASKS=immer-revoked-mapset-reads   # one task
./bench report           # table of every recorded agent run
```

Why it is built this way, including what was tried and dropped: [docs/DESIGN.md](docs/DESIGN.md).

## Tasks

| Task | Upstream bug | Language, kind | Valid solutions | Evil solutions |
|---|---|---|---|---|
| [`pgcdc-truncated-pgoutput`](tasks/pgcdc-truncated-pgoutput/issue.md) | [Trendyol/go-pq-cdc@6110b91](https://github.com/Trendyol/go-pq-cdc/commit/6110b91596ea9597b588954f94a89a1b5f4cfcd6): pgoutput decoders panic on truncated messages | Go, input hardening | 2 | 4 |
| [`konsumer-batch-retry-prebatch`](tasks/konsumer-batch-retry-prebatch/issue.md) | [Trendyol/kafka-konsumer@0555952](https://github.com/Trendyol/kafka-konsumer/commit/055595218bb0d7bd99f45bdafc7b5cbe39dbf639): retry path skips `PreBatchFn`, ignores `SendDirectToDeadLetter` | Go, behaviour / message routing | 2 | 4 |
| [`konsumer-shared-broker-race`](tasks/konsumer-shared-broker-race/issue.md) | [Trendyol/kafka-konsumer@d80db5c](https://github.com/Trendyol/kafka-konsumer/commit/d80db5c55df7fa8c83e1da30b5bbae10c9c9720a): data race, broker slice trimmed in place | Go, concurrency | 2 | 5 |
| [`gaos-scenario-races`](tasks/gaos-scenario-races/issue.md) | [Trendyol/gaos@771d620](https://github.com/Trendyol/gaos/tree/771d620499118e64e048de918103bcc404bda74f): data races on rate/limit counters and on `direct` scenario switching, not fixed upstream | Go, concurrency | 2 | 4 |
| [`sqlparse-dotted-real-name`](tasks/sqlparse-dotted-real-name/issue.md) | [andialbrecht/sqlparse@f66d12c](https://github.com/andialbrecht/sqlparse/commit/f66d12c245412f28c58f045b646eb53c0e691b8b): `get_real_name()` returns the schema for `db.schema.table` | Python, API behaviour | 3 | 4 |
| [`tomlkit-crlf-error-position`](tasks/tomlkit-crlf-error-position/issue.md) | [python-poetry/tomlkit@c07c0c1](https://github.com/python-poetry/tomlkit/commit/c07c0c15c49a893bb8a4c9f63401822d31e08942): parse errors in CRLF files point at the wrong line and column | Python, error reporting | 2 | 4 |
| [`click-show-envvar-sequence`](tasks/click-show-envvar-sequence/issue.md) | [pallets/click@c53d373](https://github.com/pallets/click/commit/c53d3732b98c764932e146c007146ad402acb229): with `show_envvar`, errors print the repr of an `envvar` list, and an empty `envvar` shows `[env var: ]` | Python, CLI error and help text | 2 | 4 |
| [`path-to-regexp-stringify-empty-text`](tasks/path-to-regexp-stringify-empty-text/issue.md) | [pillarjs/path-to-regexp@7dd6f8f](https://github.com/pillarjs/path-to-regexp/commit/7dd6f8f7bd63ae8bffca0968144508228dc61729): `stringify()` merges a param name with the text after an empty text token | TypeScript, round-trip / escaping | 2 | 4 |
| [`yaml-flow-key-after-blank-line`](tasks/yaml-flow-key-after-blank-line/issue.md) | [eemeli/yaml@55e2079](https://github.com/eemeli/yaml/commit/55e2079173b7389ef24e97592890c40f602078ad): a flow-collection map key after a blank line or comment fails to parse | TypeScript, parser state | 2 | 4 |
| [`immer-revoked-mapset-reads`](tasks/immer-revoked-mapset-reads/issue.md) | [immerjs/immer@8848a5b](https://github.com/immerjs/immer/commit/8848a5b16938e5681a890d73de7bda24b5080498): revoked Map/Set drafts still answer `has`, `size`, `keys`, `forEach` | TypeScript, API contract | 2 | 4 |

The first three Go tasks are bugs fixed upstream in October 2026, the Python and TypeScript ones
fixes from July to October 2026; the base commit is always the parent of the fix commit. `gaos-scenario-races` was found while using gaos for fault injection in a CDC pipeline: the races
reproduce with `-race`, and the fix is a local gaos commit that is not upstream yet. The base is the pinned upstream commit, the reference patch is the race
part of that fix (the same commit also fixes two unrelated gaos bugs, left out of this task).

**A note on `pgcdc-truncated-pgoutput`.** The upstream fix bounds-checks Begin, Commit, tuple data
and the TOAST copy in Update, but not the Relation decoder: a Relation message cut after the table
name still panics at the fix commit and at upstream HEAD (checked 2026-10-24). The issue asks for
every decoder reachable from `message.New`, so the reference patch is the upstream fix plus the
Relation bounds checks, and the bare upstream fix is graded as an evil (incomplete) solution. It
fails exactly on the hidden Relation tests.

**A note on `tomlkit-crlf-error-position`.** The upstream fix counts each line's own terminator,
which fixes CRLF, but it also changes an error at the very end of an LF document with no trailing
newline from `line 2 col 4` to `line 2 col 0`. The issue asks that LF documents keep their
positions, so the reference is the upstream fix plus the end-of-input case, and the bare upstream
fix (`evil-upstream-only`) fails the five end-of-input hidden cases.

**A note on `path-to-regexp-stringify-empty-text`.** The regression layer runs the project's main
suite (`src/index.spec.ts`), not `src/redos.spec.ts`: that one needs the `recheck` native binary,
which exists for linux-x64 and macOS arm64 only, and gives host-dependent results elsewhere.

## Verification layers

`harness/grade.sh` runs inside the task image with `--network none`. A patch passes only if every
layer passes; the first failing layer is the verdict.

| # | Layer | Why it exists |
|---|---|---|
| 1 | **Patch applies** to the agent's workspace (base commit + visible tests) | Graders must see exactly what the agent produced; nothing is re-based or fixed up. |
| 2 | **Test isolation**: every change the patch makes to test files is discarded and the hidden tests are copied in. Test files include what decides which tests run: `conftest.py`, `pyproject.toml`/pytest config, `vitest.config.*`, `tsconfig*.json`, `package.json` and lockfiles (`TEST_FILE_RE` in `task.env`) | Editing or deleting tests is the cheapest cheat, and a `conftest.py` hook or a vitest `exclude` does the same without touching a test. Discarding the edits (instead of rejecting the patch) means legitimate extra tests cost nothing, and a solution that only works because it rewrote a test fails on the real one. |
| 3 | **Build**: `go build ./...`; Python: byte-compile and import; TypeScript: `tsc --noEmit` | A patch that breaks another package fails here, not as a confusing test error. Vitest strips types without checking them, so for TypeScript this is the only layer that sees a type error. |
| 4 | **Visible tests**. Go: `-race`, `-count=5` | The tests the agent could run. For Go, the race detector and repetition turn "passes once" into "passes reliably"; the Python and TypeScript bugs are deterministic and run once. |
| 5 | **Hidden tests**. Go: `-race`, `-count=5` | Catch hard-coding to the visible fixtures and partial fixes: different inputs, more brokers, every truncation point of every message type, mixed line endings, every Map and Set operation, the failure branches the issue describes but the visible tests do not exercise, and behaviour that must not change. |
| 6 | **Upstream regression suite** | The fix must not break existing behaviour. The tests are the project's own, unchanged. |

Images are pinned by digest (`golang:1.25.14-bookworm@sha256:3b4a…`, `python:3.12-slim-bookworm@sha256:e697…`,
`node:22-bookworm-slim@sha256:4b15…`) and the repository by commit, with its test-data submodules.
Dependencies come from the project's lockfile (`npm ci`, `yarn --frozen-lockfile`); path-to-regexp has
none and is resolved with `npm install --before=<base commit date>`; pytest is pinned. Everything is
downloaded at build time, so grading is offline and repeatable.

## Solutions and what catches them

Expected verdicts come from file names: `reference.patch` and `ok-*.patch` must pass,
`evil-*.patch` must fail. The layer is the one reported by the verifier. Every solution of the Go
tasks was graded 3 times locally, every solution of the Python and TypeScript tasks twice
(`make test`), all with identical verdicts; CI grades each twice more on every push.

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
| `reference` | PASS | the fix: mutexes in `Rate`/`Limit`, a mutex around the `Method` scenario, atomic request counter |
| `ok-atomic-counters-rwmutex` | PASS | lock-free counters (CAS loop that increments and wraps in one step) and an `RWMutex` snapshot of the scenario |
| `evil-partial-counters-only` | FAIL hidden-tests | locks the counters, the `direct` scenario switch still races |
| `evil-hardcode-rate-one` | FAIL hidden-tests | a global toggle for `rate: 1`, the visible test's configuration |
| `evil-atomic-check-then-reset` | FAIL visible-tests | atomic increment and a separate atomic reset: no race report, but counts drift under concurrency |
| `evil-delete-visible-tests` | FAIL visible-tests | deletes the visible test file |

**sqlparse-dotted-real-name**

| Solution | Verdict | What it does |
|---|---|---|
| `reference` | PASS | upstream fix: anchor on the last dot |
| `ok-walk-dots` | PASS | follows `token_next_by` from dot to dot until there is no next one |
| `ok-reverse-scan` | PASS | a `_last_dot_idx` helper scanning the tokens backwards |
| `evil-split-value-string` | FAIL hidden-tests | `value.split(' ')[0].rsplit('.', 1)[-1]`: breaks on quoted names containing dots and on function calls |
| `evil-second-dot-only` | FAIL hidden-tests | uses the second dot when there is one: right for three parts, wrong for four |
| `evil-parent-from-last-dot` | FAIL hidden-tests | moves `get_parent_name` to the last dot too, which the issue says must not change |
| `evil-conftest-skip-tests` | FAIL visible-tests | a `conftest.py` hook that skips the visible tests, no code change |

**tomlkit-crlf-error-position**

| Solution | Verdict | What it does |
|---|---|---|
| `reference` | PASS | upstream fix (`splitlines(keepends=True)`) + the end-of-input position for a last line without a line break |
| `ok-count-line-feeds` | PASS | different approach: line = line feeds before the position + 1, column = distance from the last one |
| `evil-upstream-only` | FAIL hidden-tests | the real upstream fix: moves end-of-input errors in LF documents to column 0 |
| `evil-normalize-line-endings` | FAIL hidden-tests | replaces `\r\n` with `\n` before parsing: right positions, but `as_string()` no longer returns the input |
| `evil-global-line-ending-width` | FAIL hidden-tests | two characters per line break if the file contains any `\r\n`: wrong for mixed endings |
| `evil-drop-crlf-test-case` | FAIL visible-tests | removes the CRLF case from the visible test |

**click-show-envvar-sequence**

| Solution | Verdict | What it does |
|---|---|---|
| `reference` | PASS | upstream fix: the error hint joins each quoted variable of a sequence; help and hint treat an empty `envvar` like `None` |
| `ok-envvar-names-helper` | PASS | different design: one `_envvar_names()` helper that returns a tuple, empty when there are no variables, used by both the hint and the help |
| `evil-list-only` | FAIL hidden-tests | joins the names only for a `list`; a tuple still prints its repr |
| `evil-empty-still-shown-in-error` | FAIL hidden-tests | fixes the sequence and the help, but keeps `is not None` in the hint: `[]` and `""` still print `(env var: …)` |
| `evil-autoenv-only-for-none` | FAIL hidden-tests | hides an empty `envvar` in the help, but falls back to the automatic `PREFIX_NAME` variable only for `None`, unlike the runtime lookup |
| `evil-tests-expect-old-output` | FAIL visible-tests | rewrites the visible tests to expect the current output |

**path-to-regexp-stringify-empty-text**

| Solution | Verdict | What it does |
|---|---|---|
| `reference` | PASS | upstream fix: `stringifyName` skips empty text tokens to find the next character |
| `ok-join-adjacent-text` | PASS | different location: merges adjacent text tokens before stringifying each level |
| `evil-skip-one-empty-token` | FAIL hidden-tests | skips at most one empty token, like the visible example |
| `evil-quote-before-any-empty-text` | FAIL hidden-tests | quotes the name whenever an empty token follows: `/:"foo"/bar` instead of `/:foo/bar` |
| `evil-drop-empty-text-top-level-only` | FAIL hidden-tests | filters empty tokens in `stringify()`, not inside groups |
| `evil-delete-visible-tests` | FAIL visible-tests | deletes the visible test file |

**yaml-flow-key-after-blank-line**

| Solution | Verdict | What it does |
|---|---|---|
| `reference` | PASS | upstream fix: opens a new map item only when the current one is complete or there are new start tokens |
| `ok-reuse-opened-item` | PASS | fills the item the blank line or comment already opened instead of opening another |
| `evil-require-separator` | FAIL hidden-tests | opens a new item only after a `:`; breaks a flow key after an empty explicit key `?` |
| `evil-silence-map-comment-error` | FAIL hidden-tests | removes the composer's error instead: it parses, but comments move to the end of the map |
| `evil-flow-sequence-keys-only` | FAIL hidden-tests | handles `[...]` keys (the visible example), not `{...}` |
| `evil-remove-blank-line-from-test` | FAIL visible-tests | removes the blank line from the visible test's input |

**immer-revoked-mapset-reads**

| Solution | Verdict | What it does |
|---|---|---|
| `reference` | PASS | upstream fix: `assertUnrevoked` in Map `size`/`has`/`forEach`/`keys` and Set `size` |
| `ok-guard-in-latest` | PASS | different location: the shared `latest()` helper throws the revoked error for a revoked Map or Set state |
| `evil-untyped-guard-in-latest` | FAIL build | the same logic reading `state.revoked_` without a cast: every test passes, `tsc` does not |
| `evil-visible-paths-only` | FAIL hidden-tests | guards only Map `has`/`size` and Set `size`, the calls in the visible test |
| `evil-own-error-message` | FAIL hidden-tests | guards the right paths but throws its own error instead of immer's revoked-draft error |
| `evil-delete-visible-tests` | FAIL visible-tests | deletes the visible test file |

## Coding-agent results

Agents get the issue text and a workspace with the base commit and the visible tests
(`./bench workspace`), run headless with full permissions in a directory outside this repository,
and their diff is graded by the same verifier. Transcripts are checked for reads of this
repository (hidden tests, solutions); none were found.

### Claude Code

Claude Code 2.1.220 (`claude -p`, `--permission-mode bypassPermissions`, project settings only, no
MCP servers), three runs per task and model: 54 runs on 2026-11-01, and 6 on 2026-11-03 for
`click-show-envvar-sequence`, added later. Each cell: passing runs, then
the median wall-clock time and median cost of the three.

| Task | Haiku 5.5 | Sonnet 5.5 |
|---|---|---|
| `pgcdc-truncated-pgoutput` | 3/3 · 196 s · $2.45 | 3/3 · 60 s · $0.41 |
| `konsumer-batch-retry-prebatch` | **2/3** · 167 s · $1.26 | **1/3** · 78 s · $0.27 |
| `konsumer-shared-broker-race` | 3/3 · 122 s · $1.13 | 3/3 · 66 s · $0.21 |
| `gaos-scenario-races` | 3/3 · 133 s · $1.52 | 3/3 · 43 s · $0.24 |
| `sqlparse-dotted-real-name` | 3/3 · 62 s · $0.85 | 3/3 · 41 s · $0.30 |
| `tomlkit-crlf-error-position` | 3/3 · 113 s · $1.22 | 3/3 · 46 s · $0.21 |
| `click-show-envvar-sequence` | 3/3 · 55 s · $0.73 | 3/3 · 60 s · $0.27 |
| `path-to-regexp-stringify-empty-text` | 3/3 · 67 s · $0.69 | 3/3 · 29 s · $0.18 |
| `yaml-flow-key-after-blank-line` | 3/3 · 269 s · $2.14 | 3/3 · 68 s · $0.34 |
| `immer-revoked-mapset-reads` | 3/3 · 155 s · $1.20 | 3/3 · 57 s · $0.31 |
| **Total** | **29/30** · 69 min · $39.38 | **28/30** · 29 min · $8.38 |

Generated by `./bench summary` from `results/agents/*/*/result.json`; `make selftest`, and so CI, fails
if this table and the committed runs disagree.

- All three failures are on `konsumer-batch-retry-prebatch` and fail the same hidden test,
  `TestHiddenRetryWithoutDeadLetterProducerKeepsRetrying`: a message with `SendDirectToDeadLetter`
  and no dead letter producer configured. All three patches do return an error, so the message is
  retried, but not the consume error: two return their own error ("dead letter producer is not
  configured…"), one a new error with the same text. The test requires
  `errors.Is(err, consumeErr)`; the issue only says "the callback returns an error and the message
  follows the normal retry flow". This is a gap in the issue text rather than a wrong fix, and is
  left as measured: the verdicts are those of the verifier the runs were graded with.
- `click-show-envvar-sequence` passed 6/6, so its hidden checks (tuples, empty `envvar` in both
  the error and the help, the automatic variable) did not separate the two models. It is the one
  task where Haiku's median time (55 s) is not above Sonnet's (60 s).
- Eight runs also added test cases to existing test files; layer 2 discarded them, and all eight
  passed on the original tests. No run hit the 30-minute cap.
- Cost is the `total_cost_usd` Claude Code reports (an API-price equivalent; the runs used a
  subscription). Haiku costs more than Sonnet here because it takes more turns (per-task median
  12–30 against 7–14), each re-reading the growing context.
- Every run (patch, grader log, `result.json`, sanitized transcript) is in `results/agents/`; one
  row per run in [results/agent-runs.md](results/agent-runs.md).

### Codex CLI

| Task | Agent | Model | Time | Verdict |
|---|---|---|---|---|
| `konsumer-shared-broker-race` | Codex CLI 0.149.0-alpha.4 | gpt-5.6-terra | 6 min 49 s | **PASS** (all layers; it also edited two existing tests, which layer 2 discarded) |
| `pgcdc-truncated-pgoutput` | Codex CLI 0.149.0-alpha.4 | gpt-5.6-terra | 7 min 56 s | **PASS** (all layers; it found the Relation decoder on its own) |
| `konsumer-batch-retry-prebatch` | Codex CLI | — | — | not run: the account hit its Codex usage limit |

Both passing patches were graded three times with the same verdict. Run 2026-10-24. The other
tasks have no Codex runs.

## Layout

```
bench                       driver: check | verify | workspace | agent | report | summary | build | scan | sanitize
Makefile                    make test (lint + scan + selftest + check), make report, make summary
harness/grade.sh            verifier, runs inside the task container
harness/test-sanitize.sh    self-test of the transcript sanitizer (make selftest)
docs/DESIGN.md              why the bench is built this way
tasks/<id>/
  issue.md                  the problem statement the agent gets
  Dockerfile                pinned image: repository at the base commit, modules pre-fetched
  task.env                  upstream, base/fix commits, test-data submodules, test-file pattern;
                            Go: test packages and visible-test selector,
                            other languages: build, visible, hidden and regression commands
  tests/visible/            copied into the agent's workspace
  tests/hidden/             never shown to the agent; named TestHidden* / test_zz_hidden_* / zz-hidden-*
  solutions/*.patch         reference, ok-*, evil-*
results/agents/<task>/<agent>[-<model>]-<time>/
  candidate.patch           the agent's diff
  grade.log, result.json    verifier output; verdict, time, turns, cost
  transcript.jsonl.gz       the agent's transcript, workspace and home paths replaced
```

Other commands:

```bash
./bench verify <task> <patch>        # grade any patch
./bench workspace <task> <dir>       # what an agent gets: upstream at the base commit + visible tests
./bench agent <task> claude|codex    # run an agent headless on the task and grade its diff
./bench summary                      # the Claude Code results table, from results/agents
./bench scan [<git range>]           # leak scan; with a range, also the commits' identities
./bench sanitize <workspace> <in >out   # shareable copy of a raw agent transcript
```

`./bench scan` fails if a tracked file, gzipped transcripts included, contains a local home path,
a token or key, or an account rate-limit event from Claude Code's stream. I run it before every
push with `SCAN_EXTRA` set to my own address, so that pattern stays out of the repository. CI runs
it on every push. On pull requests it also checks that every new commit has me, Midat Faizov, as
both author and committer, with the same name and address as the base branch (`SCAN_IDENTITY`
overrides that), and has no co-author or generator lines. A merge through GitHub's web button
fails this check, because GitHub records itself as the committer, so I merge by fast-forwarding
locally.

## License

The harness and issues are MIT. Patches under `tasks/*/solutions` modify code by others:
go-pq-cdc and kafka-konsumer (MIT), gaos (Apache-2.0) by Trendyol; sqlparse (BSD-3-Clause) by Andi
Albrecht; tomlkit (MIT) by Sébastien Eustace; path-to-regexp (MIT) by Blake Embrey; yaml (ISC) by
Eemeli Aro; immer (MIT) by Michel Weststrate. Tests that extend a project's own suite are under that
project's license. `reference.patch` files are the upstream fixes by their authors (linked above),
extended where noted; the gaos reference is a local fix that is not upstream yet.
