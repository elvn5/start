#!/usr/bin/env bash
# PostToolUse hook для Claude Code: форматирует только что изменённый файл.
# Получает JSON события на stdin, путь файла — в .tool_input.file_path.
set -euo pipefail

file=$(python3 -c 'import json,sys; print(json.load(sys.stdin).get("tool_input",{}).get("file_path",""))' 2>/dev/null || true)
[[ -z "$file" || ! -f "$file" ]] && exit 0

root="${CLAUDE_PROJECT_DIR:-$(git rev-parse --show-toplevel)}"

case "$file" in
  */gen/*|*/.nuxt/*) exit 0 ;;
  *.go)    gofmt -w "$file" ;;
  *.proto) (cd "$root/backend/proto" && go tool -modfile=../tools/go.mod buf format -w "$file") ;;
  "$root"/apps/web/*.ts|"$root"/apps/web/*.vue|"$root"/apps/web/*.mjs)
           (cd "$root/apps/web" && pnpm exec eslint --fix "$file" >/dev/null 2>&1) || true ;;
esac
exit 0
