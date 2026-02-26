#!/usr/bin/env bash
#
# run.sh — Performance comparison of diffyml vs alternative YAML diff tools.
#
# Uses hyperfine for precise timing and /usr/bin/time -l for peak RSS memory.
#
# Usage:
#   bash bench/compare/run.sh                        # full run
#   bash bench/compare/run.sh --skip-install          # skip tool installation
#   bash bench/compare/run.sh --sizes small,medium    # only specific sizes
#   bash bench/compare/run.sh --runs 5                # fewer runs
set -euo pipefail

# ──────────────────────────────────────────────────────────────────────────────
# Configuration
# ──────────────────────────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
RESULTS_DIR="$SCRIPT_DIR/results"
TOOL_DIR="/tmp/bench-tools"
DATA_DIR="/tmp/bench-data"
MEM_DIR="/tmp/bench-mem"
ALL_SIZES="small medium large"
SIZES="$ALL_SIZES"
RUNS=20
WARMUP=5
MEM_RUNS=5
SKIP_INSTALL=false

# ──────────────────────────────────────────────────────────────────────────────
# Parse flags
# ──────────────────────────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case $1 in
    --skip-install)  SKIP_INSTALL=true; shift ;;
    --sizes)         SIZES="${2//,/ }"; shift 2 ;;
    --runs)          RUNS="$2"; shift 2 ;;
    *)               echo "Unknown flag: $1"; exit 1 ;;
  esac
done

# ──────────────────────────────────────────────────────────────────────────────
# Helpers
# ──────────────────────────────────────────────────────────────────────────────
info()  { printf "\033[1;34m▸ %s\033[0m\n" "$*"; }
ok()    { printf "\033[1;32m✓ %s\033[0m\n" "$*"; }
warn()  { printf "\033[1;33m⚠ %s\033[0m\n" "$*"; }

check_tool() {
  command -v "$1" &>/dev/null
}

# capitalize first letter (bash 3 compatible)
capitalize() {
  local first rest
  first="$(echo "$1" | cut -c1 | tr '[:lower:]' '[:upper:]')"
  rest="$(echo "$1" | cut -c2-)"
  echo "${first}${rest}"
}

# median of a list of numbers (one per line)
median() {
  sort -n | awk '{a[NR]=$1} END{if(NR%2==1)print a[(NR+1)/2]; else print (a[NR/2]+a[NR/2+1])/2}'
}

# Store/retrieve memory results via files (bash 3 compatible, no associative arrays)
store_mem() {
  # $1=tool_key $2=size $3=value
  echo "$3" > "$MEM_DIR/${1}__${2}"
}

get_mem() {
  # $1=tool_key $2=size
  local f="$MEM_DIR/${1}__${2}"
  if [ -f "$f" ]; then
    cat "$f"
  else
    echo "N/A"
  fi
}

# Format KB value for display
format_mem() {
  local rss="$1"
  if [ "$rss" = "N/A" ] || [ "$rss" = "0" ]; then
    echo "N/A"
    return
  fi
  if [ "$rss" -ge 1024 ] 2>/dev/null; then
    awk "BEGIN{printf \"%.1f MB\", $rss/1024}"
  else
    echo "${rss} KB"
  fi
}

# Size description lookup (bash 3 compatible)
size_desc() {
  case "$1" in
    small)  echo "~50 lines, ~1 KB" ;;
    medium) echo "~500 lines, ~9 KB" ;;
    large)  echo "~5,000 lines, ~90 KB" ;;
    xlarge) echo "~50,000 lines, ~900 KB" ;;
    *)      echo "$1" ;;
  esac
}

should_skip() {
  local key="$1" size="$2"
  if [ "$size" = "xlarge" ]; then
    case "$key" in
      yaml-diff-sters|yamldiff-sahilm) return 0 ;;  # sters/yaml-diff, sahilm/yamldiff
    esac
  fi
  return 1
}

# ──────────────────────────────────────────────────────────────────────────────
# Step 0: Prerequisites
# ──────────────────────────────────────────────────────────────────────────────
info "Checking prerequisites..."

if ! check_tool hyperfine; then
  info "Installing hyperfine via Homebrew..."
  brew install hyperfine
fi
ok "hyperfine $(hyperfine --version)"

if ! check_tool go; then
  echo "ERROR: Go is required but not installed." >&2
  exit 1
fi
ok "go $(go version | awk '{print $3}')"

mkdir -p "$TOOL_DIR" "$DATA_DIR" "$RESULTS_DIR" "$MEM_DIR"
rm -f "$MEM_DIR"/*

# ──────────────────────────────────────────────────────────────────────────────
# Step 1: Install tools
# ──────────────────────────────────────────────────────────────────────────────
if [ "$SKIP_INSTALL" = false ]; then
  info "Building diffyml from source..."
  (cd "$PROJECT_ROOT" && go build -o "$TOOL_DIR/diffyml" .)
  ok "diffyml built"

  info "Installing dyff..."
  GOBIN="$TOOL_DIR" go install github.com/homeport/dyff/cmd/dyff@v1.10.5
  ok "dyff installed"

  info "Installing semihbkgr/yamldiff..."
  GOBIN="$TOOL_DIR" go install github.com/semihbkgr/yamldiff@v0.3.1
  mv "$TOOL_DIR/yamldiff" "$TOOL_DIR/yamldiff-semihbkgr"
  ok "yamldiff-semihbkgr installed"

  info "Installing sters/yaml-diff..."
  GOBIN="$TOOL_DIR" go install github.com/sters/yaml-diff/cmd/yaml-diff@v1.4.1
  ok "yaml-diff installed"

  info "Installing sahilm/yamldiff..."
  GOBIN="$TOOL_DIR" go install github.com/sahilm/yamldiff@584d5771767b262cf171d9c1f890d6daeb82492c
  mv "$TOOL_DIR/yamldiff" "$TOOL_DIR/yamldiff-sahilm"
  ok "yamldiff-sahilm installed"

  ok "All tools installed in $TOOL_DIR"
else
  warn "Skipping installation (--skip-install)"
fi

# Verify all tools are available
TOOLS=(
  "diffyml:$TOOL_DIR/diffyml"
  "dyff:$TOOL_DIR/dyff"
  "yamldiff-semihbkgr:$TOOL_DIR/yamldiff-semihbkgr"
  "yaml-diff:$TOOL_DIR/yaml-diff"
  "yamldiff-sahilm:$TOOL_DIR/yamldiff-sahilm"
  "diff:/usr/bin/diff"
)

for entry in "${TOOLS[@]}"; do
  name="${entry%%:*}"
  path="${entry##*:}"
  if [ ! -x "$path" ]; then
    echo "ERROR: $name not found at $path" >&2
    exit 1
  fi
done
ok "All 6 tools verified"

# ──────────────────────────────────────────────────────────────────────────────
# Step 2: Generate test data
# ──────────────────────────────────────────────────────────────────────────────
info "Generating test data..."
for sz in $SIZES; do
  go run "$SCRIPT_DIR/generate_testdata.go" --size "$sz" --output-dir "$DATA_DIR/$sz"
done
ok "Test data generated"

# Tool keys (filesystem-safe), display names, and commands
TOOL_KEYS=(  "diffyml"     "dyff"               "yamldiff-semihbkgr"    "yaml-diff-sters"      "yamldiff-sahilm"      "diff-unix")
TOOL_DISPLAY=("diffyml"    "dyff"               "semihbkgr/yamldiff"    "sters/yaml-diff"      "sahilm/yamldiff"      "diff (unix)")
TOOL_CMDS=(  "$TOOL_DIR/diffyml" "$TOOL_DIR/dyff between" "$TOOL_DIR/yamldiff-semihbkgr" "$TOOL_DIR/yaml-diff" "$TOOL_DIR/yamldiff-sahilm" "/usr/bin/diff")

# ──────────────────────────────────────────────────────────────────────────────
# Step 3: Run hyperfine timing benchmarks
# ──────────────────────────────────────────────────────────────────────────────
info "Running timing benchmarks (hyperfine)..."
for sz in $SIZES; do
  FROM="$DATA_DIR/$sz/from.yaml"
  TO="$DATA_DIR/$sz/to.yaml"
  info "  Size: $sz ($(wc -l < "$FROM") lines)"

  hyperfine_args=(
    --warmup "$WARMUP" --min-runs "$RUNS"
    --export-markdown "$RESULTS_DIR/timing-$sz.md"
    --export-json "$RESULTS_DIR/timing-$sz.json"
  )
  i=0
  while [ $i -lt ${#TOOL_KEYS[@]} ]; do
    if ! should_skip "${TOOL_KEYS[$i]}" "$sz"; then
      hyperfine_args+=(--command-name "${TOOL_DISPLAY[$i]}" "${TOOL_CMDS[$i]} $FROM $TO > /dev/null 2>&1 || true")
    fi
    i=$((i + 1))
  done
  hyperfine "${hyperfine_args[@]}"

  ok "  $sz timing complete"
done

# ──────────────────────────────────────────────────────────────────────────────
# Step 4: Memory measurement (peak RSS via /usr/bin/time -l)
# ──────────────────────────────────────────────────────────────────────────────
info "Measuring peak memory usage..."

for sz in $SIZES; do
  FROM="$DATA_DIR/$sz/from.yaml"
  TO="$DATA_DIR/$sz/to.yaml"
  info "  Memory: $sz"

  i=0
  while [ $i -lt ${#TOOL_KEYS[@]} ]; do
    key="${TOOL_KEYS[$i]}"
    if should_skip "$key" "$sz"; then
      i=$((i + 1))
      continue
    fi
    cmd="${TOOL_CMDS[$i]} $FROM $TO"
    rss_values=""
    rss_count=0

    j=0
    while [ $j -lt $MEM_RUNS ]; do
      # macOS /usr/bin/time -l reports "maximum resident set size" in bytes
      output=$(/usr/bin/time -l bash -c "$cmd > /dev/null 2>&1 || true" 2>&1) || true
      rss=$(echo "$output" | grep "maximum resident set size" | awk '{print $1}')
      if [ -n "$rss" ]; then
        rss_values="${rss_values}${rss}
"
        rss_count=$((rss_count + 1))
      fi
      j=$((j + 1))
    done

    if [ $rss_count -gt 0 ]; then
      median_kb=$(echo "$rss_values" | grep -v '^$' | median | awk '{printf "%.0f", $1/1024}')
      store_mem "$key" "$sz" "$median_kb"
    else
      store_mem "$key" "$sz" "0"
    fi

    i=$((i + 1))
  done
  ok "  $sz memory complete"
done

# ──────────────────────────────────────────────────────────────────────────────
# Step 5: Assemble REPORT.md
# ──────────────────────────────────────────────────────────────────────────────
info "Assembling report..."

REPORT="$RESULTS_DIR/REPORT.md"

{
  echo "# diffyml Performance Comparison"
  echo ""
  echo "**Date:** $(date '+%Y-%m-%d %H:%M')"
  echo "**System:** $(uname -s) $(uname -m) | $(sysctl -n machdep.cpu.brand_string 2>/dev/null || uname -p)"
  echo "**Go:** $(go version | awk '{print $3}')"
  echo "**Runs:** $RUNS (warmup: $WARMUP)"
  echo ""

  # Tool versions
  echo "## Tools"
  echo ""
  echo "| Tool | Source | Language |"
  echo "|------|--------|----------|"
  echo "| diffyml | built from source | Go |"
  echo "| dyff | homeport/dyff | Go |"
  echo "| semihbkgr/yamldiff | semihbkgr/yamldiff | Go |"
  echo "| sters/yaml-diff | sters/yaml-diff | Go |"
  echo "| sahilm/yamldiff | sahilm/yamldiff | Go |"
  echo "| diff (unix) | system | C |"
  echo ""

  # Timing results
  echo "## Execution Time"
  echo ""
  for sz in $SIZES; do
    echo "### $(capitalize "$sz") ($(size_desc "$sz"))"
    echo ""
    if [ -f "$RESULTS_DIR/timing-$sz.md" ]; then
      cat "$RESULTS_DIR/timing-$sz.md"
    else
      echo "_No data_"
    fi
    echo ""
  done

  # Memory results
  echo "## Peak Memory Usage (RSS)"
  echo ""
  header_line="| Tool |"
  sep_line="|------|"
  for sz in $SIZES; do
    header_line="$header_line $(capitalize "$sz") |"
    sep_line="${sep_line}------|"
  done
  echo "$header_line"
  echo "$sep_line"

  i=0
  while [ $i -lt ${#TOOL_KEYS[@]} ]; do
    key="${TOOL_KEYS[$i]}"
    display="${TOOL_DISPLAY[$i]}"
    row="| $display |"
    for sz in $SIZES; do
      rss=$(get_mem "$key" "$sz")
      row="$row $(format_mem "$rss") |"
    done
    echo "$row"
    i=$((i + 1))
  done
  echo ""

  # Summary
  echo "## Summary"
  echo ""
  echo "### How to read these results"
  echo ""
  echo "- **Execution Time**: Lower is better. The \"Relative\" column shows how many times slower each tool is compared to the fastest."
  echo "- **Peak Memory**: Lower is better. Shows the maximum resident set size during execution."
  echo "- **diff (unix)** is included as a baseline — it performs plain text diff without YAML awareness."
  echo ""
  echo "### Notes"
  echo ""
  echo "- All tools were benchmarked with stdout redirected to /dev/null to measure processing time only."
  echo "- Memory was measured using macOS \`/usr/bin/time -l\` (median of $MEM_RUNS runs)."
  echo "- Test data contains realistic YAML service configurations with ~20% modifications between files."
  echo ""
  echo "---"
  echo ""
  echo "Generated by \`make bench-compare\` on $(date '+%Y-%m-%d')."
} > "$REPORT"

ok "Report written to $REPORT"

# Print a quick summary to terminal
echo ""
echo "================================================================"
echo "  Benchmark complete! Results: $REPORT"
echo "================================================================"
