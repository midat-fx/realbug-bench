#!/usr/bin/env bash
# ./bench sanitize: every spelling of the workspace path and $HOME is replaced, rate-limit events are dropped,
# and Claude Code's dashed project-directory names (~/.claude/projects, scratchpads) are replaced too.
# The workspace is built from a TMPDIR with a trailing slash, as on macOS, so it contains "//".
set -euo pipefail
cd "$(dirname "$0")/.."

ws=/var/folders/q1/x9_y/T//realbug-claude-run-task
rate_limit='"type":"rate_limit_''event"' # split so that ./bench scan does not flag this file
projects=.claude/'projects' users=-'Users'-ann home=-'home'-bob # Claude Code's dashed spellings, split likewise
got=$(HOME=/srv/u ./bench sanitize "$ws" <<EOF
{"type":"system","cwd":"/private/var/folders/q1/x9_y/T/realbug-claude-run-task"}
{$rate_limit,"rate_limit_info":{"status":"allowed"}}
{"type":"assistant","text":"edit /var/folders/q1/x9_y/T/realbug-claude-run-task/src/a.py"}
{"type":"user","text":"cd /var/folders/q1/x9_y/T//realbug-claude-run-task && cat /srv/u/.gitconfig"}
{"type":"user","text":"a different path stays: /var/folders/q1/x9_yT/realbug-claude-run-task"}
{"memory_paths":{"auto":"/srv/u/$projects/-private-var-folders-q1-x9-y-T--realbug-claude-run-task/memory/"}}
{"stdout":"$users-projects-x\n$home--cache\n/private/tmp/claude-501/$users-Desktop-y/tasks"}
EOF
)
want='{"type":"system","cwd":"$WORKSPACE"}
{"type":"assistant","text":"edit $WORKSPACE/src/a.py"}
{"type":"user","text":"cd $WORKSPACE && cat $HOME/.gitconfig"}
{"type":"user","text":"a different path stays: /var/folders/q1/x9_yT/realbug-claude-run-task"}
{"memory_paths":{"auto":"$HOME/'"$projects"'/$PROJECT/memory/"}}
{"stdout":"-$HOME-projects-x\n-$HOME--cache\n/private/tmp/claude-501/-$HOME-Desktop-y/tasks"}'

if [ "$got" != "$want" ]; then
	diff <(echo "$want") <(echo "$got") || true
	echo "FAIL sanitize" >&2
	exit 1
fi
echo "ok sanitize"
