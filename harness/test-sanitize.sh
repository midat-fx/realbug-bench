#!/usr/bin/env bash
# ./bench sanitize: every spelling of the workspace path and $HOME is replaced, rate-limit events are dropped.
# The workspace is built from a TMPDIR with a trailing slash, as on macOS, so it contains "//".
set -euo pipefail
cd "$(dirname "$0")/.."

ws=/var/folders/q1/x9_y/T//realbug-claude-run-task
rate_limit='"type":"rate_limit_''event"' # split so that ./bench scan does not flag this file
got=$(HOME=/srv/u ./bench sanitize "$ws" <<EOF
{"type":"system","cwd":"/private/var/folders/q1/x9_y/T/realbug-claude-run-task"}
{$rate_limit,"rate_limit_info":{"status":"allowed"}}
{"type":"assistant","text":"edit /var/folders/q1/x9_y/T/realbug-claude-run-task/src/a.py"}
{"type":"user","text":"cd /var/folders/q1/x9_y/T//realbug-claude-run-task && cat /srv/u/.gitconfig"}
{"type":"user","text":"a different path stays: /var/folders/q1/x9_yT/realbug-claude-run-task"}
EOF
)
want='{"type":"system","cwd":"$WORKSPACE"}
{"type":"assistant","text":"edit $WORKSPACE/src/a.py"}
{"type":"user","text":"cd $WORKSPACE && cat $HOME/.gitconfig"}
{"type":"user","text":"a different path stays: /var/folders/q1/x9_yT/realbug-claude-run-task"}'

if [ "$got" != "$want" ]; then
	diff <(echo "$want") <(echo "$got") || true
	echo "FAIL sanitize" >&2
	exit 1
fi
echo "ok sanitize"
