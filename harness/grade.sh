#!/usr/bin/env bash
# Grades one candidate patch inside a task container. Runs with --network none.
#
# Mounts (read-only):
#   /in/candidate.patch   diff against the agent workspace (base commit + visible tests)
#   /task/task.env        Go: TEST_PKGS, VISIBLE_RUN, REGRESSION_PKGS, REPEAT (hidden tests are named TestHidden*)
#                         other languages: BUILD_CMD, VISIBLE_CMD, HIDDEN_CMD, REGRESSION_CMD
#   /task/tests/visible   tests the agent saw
#   /task/tests/hidden    tests the agent never saw
#
# Prints one line "VERDICT PASS|FAIL <layer>" and exits 0 on PASS, 1 on FAIL, 2 on harness error.
set -uo pipefail

# shellcheck disable=SC1091
source /task/task.env
REPEAT="${REPEAT:-5}"
TEST_FILE_RE="${TEST_FILE_RE:-(_test\.go$|/testdata/)}"

verdict() {
	echo "VERDICT $1 $2"
	[ "$1" = PASS ] && exit 0
	exit 1
}

cd /work/repo || { echo "VERDICT ERROR no-repo"; exit 2; }
git config user.email grader@bench.local
git config user.name grader

# Workspace the agent got: base commit + visible tests.
cp -R /task/tests/visible/. .
git add -A && git commit -qm "visible tests" || { echo "VERDICT ERROR visible-commit"; exit 2; }
WORKSPACE=$(git rev-parse HEAD)

echo "== layer 1: patch applies"
[ -s /in/candidate.patch ] || verdict FAIL empty-patch
git apply --whitespace=nowarn /in/candidate.patch || verdict FAIL patch-does-not-apply

echo "== layer 2: test isolation (candidate changes to tests are discarded)"
touched=$(
	{ git diff --name-only "$WORKSPACE"; git ls-files --others --exclude-standard; } | sort -u | grep -E "$TEST_FILE_RE" || true
)
if [ -n "$touched" ]; then
	echo "discarding candidate changes to test files:"
	echo "$touched" | sed 's/^/  /'
	while read -r f; do
		if git cat-file -e "$WORKSPACE:$f" 2>/dev/null; then
			git checkout -q "$WORKSPACE" -- "$f"
		else
			rm -f "$f"
		fi
	done <<<"$touched"
fi
cp -R /task/tests/hidden/. .

if [ "${LANGUAGE:-go}" = go ]; then
	BUILD_CMD="go build ./..."
	VISIBLE_CMD="go test -race -count=$REPEAT -timeout 600s -run '$VISIBLE_RUN' $TEST_PKGS"
	HIDDEN_CMD="go test -race -count=$REPEAT -timeout 600s -run '^TestHidden' $TEST_PKGS"
	REGRESSION_CMD="go test -count=1 -timeout 900s $REGRESSION_PKGS"
fi

echo "== layer 3: build"
eval "$BUILD_CMD" || verdict FAIL build

echo "== layer 4: visible tests"
eval "$VISIBLE_CMD" || verdict FAIL visible-tests

echo "== layer 5: hidden tests"
eval "$HIDDEN_CMD" || verdict FAIL hidden-tests

echo "== layer 6: upstream regression suite"
eval "$REGRESSION_CMD" || verdict FAIL regression

verdict PASS all-layers
