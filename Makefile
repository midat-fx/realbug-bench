# make test: lint the harness, then grade every solution of every task twice (needs Docker).
# make test TASKS="sqlparse-dotted-real-name immer-revoked-mapset-reads" grades a subset.
TASKS ?=

.PHONY: test lint report

test: lint
	RUNS=2 ./bench check $(TASKS)

lint:
	bash -n bench harness/grade.sh
	shellcheck -S warning bench harness/grade.sh

report:
	./bench report
