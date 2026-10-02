#!/usr/bin/env bash
# Инициализация проекта, созданного из шаблона: task init NAME=myshop [MODULE=github.com/me/myshop/backend]
#   NAME   — имя проекта: compose-проекты (<NAME>, <NAME>-prod), Docker-тома и образы, package.json
#   MODULE — путь Go-модуля backend (по умолчанию остаётся app/backend — это валидно для приложения)
# Запускать один раз, на чистом дереве. После — task gen && task check.
set -euo pipefail

NAME="${1:-}"
MODULE="${2:-}"
OLD_MODULE="app/backend"

if [[ ! "$NAME" =~ ^[a-z][a-z0-9-]*$ ]]; then
  echo "usage: $0 <name> [go-module-path]   # name: [a-z][a-z0-9-]*" >&2
  exit 1
fi

root="$(git rev-parse --show-toplevel)"
cd "$root"

if ! grep -q '^name: app$' compose.yaml; then
  echo "Похоже, проект уже инициализирован (compose.yaml: name != app)." >&2
  exit 1
fi

# sed -i, переносимый между GNU и BSD (macOS)
sedi() { sed -i.bak "$@" && rm -f "${@: -1}.bak"; }

# ---------- имя проекта ----------
sedi "s/^name: app$/name: ${NAME}/" compose.yaml
sedi "s/^name: app-prod$/name: ${NAME}-prod/" compose.prod.yaml
sedi "s/app_pgdata/${NAME}_pgdata/; s/-t app-/-t ${NAME}-/g" Taskfile.yml
sedi "s/\"name\": \"app\"/\"name\": \"${NAME}\"/" package.json

# ---------- путь Go-модуля ----------
if [[ -n "$MODULE" && "$MODULE" != "$OLD_MODULE" ]]; then
  # Все отслеживаемые и новые файлы, где встречается старый путь (включая сгенерированный код —
  # его всё равно перегенерирует task gen, но так проект компилируется и без него).
  git ls-files --cached --others --exclude-standard -z \
    | while IFS= read -r -d '' f; do
        [[ "$f" == docs/* || ! -f "$f" ]] && continue
        grep -qF "$OLD_MODULE" "$f" && sedi "s#${OLD_MODULE}#${MODULE}#g" "$f"
      done
  (cd backend && go mod tidy)
fi

echo "Готово: проект '${NAME}'${MODULE:+, модуль ${MODULE}}."
echo "Дальше: task setup && task check"
