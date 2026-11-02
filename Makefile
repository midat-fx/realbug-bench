# make test: lint the harness, then grade every solution of every task twice (needs Docker).
# make test TASKS="sqlparse-dotted-real-name immer-revoked-mapset-reads" grades a subset.
# make selftest: tests of the harness itself (no Docker).
# make scan: leak scan of every tracked file (SCAN_EXTRA adds private patterns from the environment).
TASKS ?=

.PHONY: test lint scan selftest report

test: lint scan selftest
	RUNS=2 ./bench check $(TASKS)

lint:
	bash -n bench harness/*.sh
	shellcheck -S warning bench harness/*.sh

scan:
	./bench scan

selftest:
	harness/test-sanitize.sh

report:
	./bench report
