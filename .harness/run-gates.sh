#!/usr/bin/env bash
set -u

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HARNESS_DIR="$ROOT_DIR/.harness"
CONFIG_FILE="$HARNESS_DIR/config.yaml"

if [[ ! -f "$CONFIG_FILE" ]]; then
  echo "Missing harness config: $CONFIG_FILE"
  exit 2
fi

run_gate() {
  local gate_key="$1"
  local script_name="$2"
  local gate_name="$3"
  local script_path="$HARNESS_DIR/$script_name"

  local enabled
  enabled=$(python3 - "$CONFIG_FILE" "$gate_key" <<'PY'
import sys
from pathlib import Path

cfg_path = Path(sys.argv[1])
gate_key = sys.argv[2]

try:
    import yaml
except Exception:
    print("true")
    raise SystemExit(0)

data = yaml.safe_load(cfg_path.read_text(encoding="utf-8")) or {}
enabled = data.get("gates", {}).get(gate_key, {}).get("enabled", True)
print("true" if enabled else "false")
PY
)

  if [[ "$enabled" != "true" ]]; then
    echo "[SKIP] $gate_name (disabled in config)"
    return 0
  fi

  if [[ ! -x "$script_path" ]]; then
    echo "[FAIL] $gate_name ($script_name is missing or not executable)"
    return 1
  fi

  echo "[RUN ] $gate_name"
  "$script_path" "$CONFIG_FILE"
}

declare -a requested=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --gates)
      shift
      IFS=',' read -r -a requested <<< "$1"
      ;;
    *)
      ;;
  esac
  shift || true
done

declare -a keys=("gate_a" "gate_b" "gate_c" "gate_d" "gate_e" "gate_f")
declare -a scripts=("gate-a.sh" "gate-b.sh" "gate-c.sh" "gate-d.sh" "gate-e.sh" "gate-f.sh")
declare -a names=(
  "Gate A: Formatting + Vet"
  "Gate B: Import Boundaries"
  "Gate C: Structural Ratchets"
  "Gate D: Snapshot Testing"
  "Gate E: Golden Outputs"
  "Gate F: Numerical Equivalence"
)

should_run() {
  local key="$1"
  local short="${key#gate_}"
  if [[ ${#requested[@]} -eq 0 ]]; then
    return 0
  fi
  for item in "${requested[@]}"; do
    local lower
    lower="$(printf '%s' "$item" | tr '[:upper:]' '[:lower:]')"
    if [[ "$lower" == "$short" || "$lower" == "$key" ]]; then
      return 0
    fi
  done
  return 1
}

failures=0
for idx in "${!keys[@]}"; do
  key="${keys[$idx]}"
  script="${scripts[$idx]}"
  name="${names[$idx]}"

  if should_run "$key"; then
    if ! run_gate "$key" "$script" "$name"; then
      failures=$((failures + 1))
    fi
  fi
done

if [[ $failures -gt 0 ]]; then
  echo "Harness failed: $failures gate(s) failed"
  exit 1
fi

echo "Harness passed"
exit 0
