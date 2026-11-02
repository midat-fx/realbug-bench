# make test: lint the harness, then grade every solution of every task twice (needs Docker).
# make test TASKS="sqlparse-dotted-real-name immer-revoked-mapset-reads" grades a subset.
# make selftest: tests of the harness itself, and the README results table against ./bench summary (no Docker).
# make scan: leak scan of every tracked file (SCAN_EXTRA adds private patterns from the environment).
SHELL := bash
TASKS ?=

.PHONY: test lint scan selftest report summary

test: lint scan selftest
	RUNS=2 ./bench check $(TASKS)

lint:
	bash -n bench harness/*.sh
	shellcheck -S warning bench harness/*.sh

scan:
	./bench scan

selftest:
	harness/test-sanitize.sh
	diff <(./bench summary) <(sed -n '/^| Task | Haiku/,/^| \*\*Total/p' README.md)

report:
	./bench report

summary:
	./bench summary
