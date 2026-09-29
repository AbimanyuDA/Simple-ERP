#!/bin/sh
set -eu

root_dir=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
mapping="$root_dir/docs/PROCESS-MAPPING.md"

for file in "$root_dir"/docs/bpmn/*.bpmn; do
  xmllint --noout "$file"
done

task_ids=$(rg -o '<bpmn:(userTask|serviceTask) id="[A-Z0-9-]+"' "$root_dir"/docs/bpmn/l2-*.bpmn \
  | sed 's/.*id="//; s/"$//' \
  | sort -u)
task_count=$(printf '%s\n' "$task_ids" | sed '/^$/d' | wc -l | tr -d ' ')

if [ "$task_count" -ne 39 ]; then
  echo "expected 39 stable BPMN task IDs, found $task_count" >&2
  exit 1
fi

for task_id in $task_ids; do
  count=$(grep -c "^| $task_id |" "$mapping" || true)
  if [ "$count" -ne 1 ]; then
    echo "$task_id must have exactly one mapping row, found $count" >&2
    exit 1
  fi
done

action_count=$(grep -c '^| ACT-' "$mapping" || true)
if [ "$action_count" -ne 22 ]; then
  echo "expected 22 supporting action mappings, found $action_count" >&2
  exit 1
fi

echo "traceability structure verified: 39 BPMN tasks and 22 supporting actions have exactly one mapping row"
echo "note: this structural check does not prove source, UI, API, audit, test, or runtime implementation"
