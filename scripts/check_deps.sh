#!/usr/bin/env bash
set -o pipefail

# Strict layered dependency check for the docx public layer.
#
# Rules:
#   1. observability/tracing/*.go MUST NOT import mqx subpackages (kafkax,
#      redisx, observability/exporter/*). It may use go-redis + confluent-
#      kafka-go directly to honour the "self-build" requirement.
#   2. observability/tracing/*.go MUST NOT import couchbase or mongo (the
#      sub-modules are downstream consumers; reverse coupling is a bug).
#   3. The two sub-modules (couchbase, mongo) may import docx, but they
#      MUST NOT pull in each other or docx's internal observability/tracing
#      tests (verified separately by the sub-module build).
#
# Exits non-zero on any violation.

TRACING_DIR="observability/tracing"
FORBIDDEN_TRACING=(
  'github\.com/gospacex/mqx/kafkax'
  'github\.com/gospacex/mqx/redisx'
  'github\.com/gospacex/mqx/observability/exporter'
  'github\.com/gospacex/hubx/cache/couchbase'
  'github\.com/gospacex/hubx/cache/mongo'
)

violations=0

while IFS= read -r -d '' f; do
  for pat in "${FORBIDDEN_TRACING[@]}"; do
    if grep -qn "$pat" "$f"; then
      echo "FORBIDDEN IMPORT in $f: $(grep -h "$pat" "$f")"
      ((violations++))
    fi
  done
done < <(find "$TRACING_DIR" -name '*.go' -type f -print0)

if (( violations > 0 )); then
  echo "FAILED: $violations forbidden import(s) found in tracing package"
  exit 1
fi

echo "✓ tracing package: no forbidden imports"
exit 0
