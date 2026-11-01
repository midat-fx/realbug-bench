# make test: lint the harness, then grade every solution of every task twice (needs Docker).
# make test TASKS="sqlparse-dotted-real-name immer-revoked-mapset-reads" grades a subset.
# make scan: leak scan of every tracked file (SCAN_EXTRA adds private patterns from the environment).
TASKS ?=

.PHONY: test lint scan report

test: lint scan
	RUNS=2 ./bench check $(TASKS)

lint:
	bash -n bench harness/grade.sh
	shellcheck -S warning bench harness/grade.sh

scan:
	./bench scan

report:
	./bench report
