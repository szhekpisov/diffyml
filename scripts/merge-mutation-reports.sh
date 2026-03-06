#!/usr/bin/env bash
# merge-mutation-reports.sh — merge per-package gremlins JSON reports into one.
#
# Usage: bash scripts/merge-mutation-reports.sh report-a.json report-b.json ... > merged.json
#
# Each input is a standard gremlins JSON report. The script:
#   1. Collects file entries that have at least one mutation whose status != "NOT COVERED"
#   2. Deduplicates by file_name (keeps the first occurrence)
#   3. Recalculates aggregate stats from the merged mutation list
set -euo pipefail

if [ $# -eq 0 ]; then
  echo "usage: $0 report1.json report2.json ..." >&2
  exit 1
fi

jq -s '
  # Grab go_module from the first report
  (.[0].go_module) as $module |

  # Collect all file entries that have at least one non-NOT-COVERED mutation
  [.[] | .files[] | select(.mutations | any(.status != "NOT COVERED"))] |

  # Deduplicate by file_name (keep first occurrence)
  group_by(.file_name) | map(.[0]) |

  # Build aggregate counts from all mutations across merged files
  # gremlins counts: total = killed + lived (excludes timed_out, not_viable, not_covered)
  . as $files |
  [.[] | .mutations[]] as $all |
  ([$all[] | select(.status == "KILLED")]      | length) as $killed |
  ([$all[] | select(.status == "LIVED")]       | length) as $lived |
  ([$all[] | select(.status == "TIMED OUT")]   | length) as $timed_out |
  ([$all[] | select(.status == "NOT VIABLE")]  | length) as $not_viable |
  ([$all[] | select(.status == "NOT COVERED")] | length) as $not_covered |
  ($killed + $lived) as $total |

  # Efficacy = killed / total * 100
  (if $total == 0 then 100 else ($killed / $total * 100) end) as $efficacy |

  # Coverage = total / (total + not_covered) * 100
  (($total + $not_covered) as $denom |
   if $denom == 0 then 100 else ($total / $denom * 100) end) as $coverage |

  # Build mutator_statistics by aggregating mutation types
  ([$all[] | .type] | group_by(.) | map({(.[0]): length}) | add // {}) as $mutator_stats |

  # Assemble final report
  {
    go_module: $module,
    files: $files,
    test_efficacy: $efficacy,
    mutations_coverage: $coverage,
    mutants_total: $total,
    mutants_killed: $killed,
    mutants_lived: $lived,
    mutants_not_viable: $not_viable,
    mutants_not_covered: $not_covered,
    mutator_statistics: $mutator_stats
  }
' "$@"
