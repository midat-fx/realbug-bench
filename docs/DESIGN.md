# Design notes

Why the bench is built the way it is. The README says what it does; this file is the reasoning
behind it, including the options that were tried and dropped.

## Tasks come from real, already-fixed bugs

A task is an upstream bug with a known fix: the base commit is the parent of the fix commit, the
issue text describes the symptom (written for the task, not copied from the upstream issue), and
the upstream patch is the reference. This keeps two things honest:

- **The bug is real.** Somebody hit it and somebody fixed it; the task is not shaped around what an
  agent happens to be good at.
- **A correct answer exists and was reviewed.** The reference has passed upstream review and the
  project's own suite.

The Go tasks are from libraries I use (go-pq-cdc, kafka-konsumer, gaos). The Python and TypeScript
tasks were picked by scanning the history of well-known projects for fixes that are:

1. **Recent** (July to October 2026). Older fixes are more likely to be in a model's training data
   together with their diff.
2. **Small in source, specified by public behaviour.** One function or one class, observable
   through the public API, so the issue can state the expected behaviour without pointing at the
   line to change.
3. **Wide enough for hidden tests to separate a fix from a patch for the example.** Several paths,
   several inputs, or a behaviour that has to stay unchanged.
4. **Cheap to grade.** Light dependencies, a suite that runs in seconds, no services.

Candidates that were looked at and dropped:

| Candidate | Why not |
|---|---|
| hono `eb78eca` (encoded duplicate query keys) | Good bug, but the dev install is pnpm 12, vite-plus and wrangler: gigabytes per image and per agent workspace on a shared, disk-constrained machine. |
| packaging `55cbf1b` (unbounded range bounds) | The fix is in a private module (`packaging._ranges`); the issue could not be stated through public behaviour. |
| marshmallow `252090c` (nested partial with `data_key`) | A one-word fix (`field_name` -> `attr_name`); not enough room for hidden tests to separate solutions. |

## When the reference is not the bare upstream patch

The issue describes the behaviour the code should have, and the hidden tests check that
behaviour. When the upstream fix is incomplete, the reference is upstream plus the missing part,
and the bare upstream patch becomes an evil solution. That happened twice:

- **pgcdc-truncated-pgoutput**: the upstream fix misses the Relation decoder.
- **tomlkit-crlf-error-position**: the upstream fix (`splitlines(keepends=True)`) fixes CRLF, but
  moves an error at the very end of an LF document without a trailing newline from
  `line 2 col 4` to `line 2 col 0`, a regression for documents that were reported correctly
  before. The issue asks that LF positions do not change, so the reference keeps the end-of-input
  position, and `evil-upstream-only` fails exactly the five end-of-input hidden cases.

In both cases the README says so, so nobody mistakes the reference for the upstream commit.

## The verifier

### Test edits are discarded, not rejected

Layer 2 throws away every change a patch makes to test files and copies the hidden tests in.
Rejecting such patches would punish agents for adding their own tests, which is good practice.
Discarding them means extra tests cost nothing, and a solution that only passes because it edited
a test fails on the real one.

"Test files" includes what controls which tests run. Per language, `TEST_FILE_RE` covers:

- Python: `tests/`, every `conftest.py`, `pyproject.toml`, `setup.cfg`, `tox.ini`, `pytest.ini`.
  A `conftest.py` hook can mark tests as skipped, and `addopts` can deselect them.
- TypeScript: `*.test.ts`, `*.spec.ts`, test directories, `vite(st).config.*`, `tsconfig*.json`,
  `package.json` and the lockfile. `vitest run <file>` still applies the config's
  `include`/`exclude`, so the config is as much a part of the tests as the test files are.

`evil-conftest-skip-tests` (sqlparse) is the example: its `conftest.py` skips the visible tests,
the verifier discards it, and the unfixed code fails them.

### The build layer for TypeScript is `tsc --noEmit`

Vitest strips types without checking them, so a patch with a type error passes every test.
`evil-untyped-guard-in-latest` (immer) is that case: the logic is right (the same logic with a cast
is `ok-guard-in-latest`), it reads a field the type does not have, and only the build layer
catches it. The project's own CI type-checks, so a patch that does not compile would not be
merged.

### Repetition only where it buys something

Go tasks run visible and hidden tests with `-race -count=5`: three of them are about data races,
and a race that shows up one run in five has to fail the verdict. The Python and TypeScript bugs are
deterministic and single-threaded; their tests run once. Flakiness in the verifier itself is still
checked: `./bench check` grades every solution twice and fails on a verdict that changes.

### Regression suite: the project's own tests, minus what cannot run

Layer 6 runs the project's suite unchanged. The one exclusion is `src/redos.spec.ts` in
path-to-regexp: it runs the `recheck` ReDoS analyser, whose native binary is published only for
linux-x64 and macOS arm64. Under colima on Apple silicon (linux-arm64) it falls back to a mode that
reports different results and times out, which would make verdicts depend on the host. The
stringify fix does not touch regex generation, and the main suite (`src/index.spec.ts`, every
parse/compile/match/stringify case) still runs.

## Reproducible images

- Base images are pinned by digest (`golang`, `python:3.12-slim-bookworm`, `node:22-bookworm-slim`).
- Repositories are pinned by commit; test-data submodules (`tests/toml-test`, `tests/yaml-test-suite`,
  `tests/json-test-suite`) are checked out at the commit's gitlinks, in the image and in the agent's
  workspace (`SUBMODULES` in `task.env`).
- Dependencies come from the repository's lockfile where there is one (`npm ci` for yaml, `yarn
  install --frozen-lockfile` for immer). path-to-regexp has no lockfile, so the image resolves
  dependencies with `npm install --before=<base commit date>`: the same versions on every build.
  Python tasks pin pytest (and PyYAML for tomlkit's suite); the libraries have no runtime
  dependencies.
- Grading runs with `--network none`, so a build that passed once grades the same way offline.

## Evil solutions

Each evil solution is a shortcut an agent plausibly takes, aimed at one hidden check:

- **Overfitting to the visible test**: only the paths the visible test calls (immer), only flow
  sequences because the visible example is `[3]: 4` (yaml), only three-part names (sqlparse), only one
  empty token (path-to-regexp).
- **A symptom fix in the wrong place**: silencing the composer error instead of fixing the parser
  state (yaml). It parses, but comments move to the end of the map.
- **A fix with a side effect**: normalising line endings before parsing (tomlkit). Positions are
  right, but the document no longer round-trips byte for byte, which is tomlkit's whole point.
- **Over-correction**: quoting a name before every empty token (path-to-regexp), changing
  `get_parent_name` along with `get_real_name` (sqlparse).
- **Wrong error**: throwing a custom error instead of the library's revoked-draft error (immer).
- **Editing or removing the tests.**

At least two valid solutions per task (the reference and an `ok-*` with a different design) make sure
the hidden tests check behaviour, not the reference's implementation. When an evil passed every test
during development, the hidden tests were extended rather than the evil dropped: yaml's
"silence the error" variant led to the test that checks which key a comment is attached to.

## Agent runs

- **Same input as a human contributor**: the workspace is a fresh clone at the base commit plus the
  visible tests, outside this repository; the prompt is the issue text. The agent may install
  dependencies and run anything (network on, `bypassPermissions`).
- **Isolation from the host's configuration**: `--setting-sources project`, an empty MCP config,
  slash commands off, no session persistence. The agent gets neither the user's global
  instructions nor hooks nor tools beyond Claude Code's built-ins.
- **Contamination check**: the transcript is searched for paths into this repository (hidden tests,
  solutions). A sanitized copy (workspace and home paths replaced) is kept in `results/` for audit.
- **A wall-clock cap** (30 minutes) so a stuck run is recorded as such instead of hanging the sweep.
- **Three runs per task and model.** One run says little about an agent that samples; three show
  whether a pass is repeatable and stay within the budget of 60 runs (9 tasks x 2 models x 3 = 54).
- **Cost** is the `total_cost_usd` Claude Code reports in its final event. Runs used a subscription,
  so it is an API-price equivalent, not money spent.
- **Verdicts are not revised after the fact.** The first sweep showed that the
  `konsumer-batch-retry-prebatch` issue under-specifies one case (which error the callback returns
  when there is no dead letter producer), and the hidden test is stricter than the text. Changing the
  test or the issue and re-grading would tune the bench to the results it had already seen. The
  numbers stay as graded; a clarified issue would be a new version of the task, with its own runs
  reported separately.
