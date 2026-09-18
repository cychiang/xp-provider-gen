#!/bin/bash
# Repo-wide consistency gate: every check below was once a manual audit finding.
# Each prints one PASS/FAIL line plus its violations; any FAIL exits 1.
# Run from anywhere via `make check-consistency`. To add a check, write a
# `report Cn "<title>" "<violations>"` block here and mention it in
# docs/development.md.
set -euo pipefail

cd "$(dirname "$0")/.."

FAILED=0

# report ID TITLE VIOLATIONS — PASS when VIOLATIONS is empty, else FAIL and list them.
report() {
    if [ -z "$3" ]; then
        echo "PASS $1 $2"
    else
        echo "FAIL $1 $2"
        printf '%s\n' "$3" | sed 's/^/     /'
        FAILED=1
    fi
}

# hits PATTERN [PATHSPEC...] — matching lines in tracked and untracked (not
# ignored) files. docs/plans/ is a historical record that is deliberately left
# as written, and this script necessarily spells out the patterns it bans.
hits() {
    local pattern=$1
    shift
    git grep --untracked -nEi "$pattern" -- "${@:-.}" ':!docs/plans' ':!hack/check-consistency.sh' || true
}

# Names are stable only if nothing still points at the old ones: a stale
# reference in a doc or workflow reads as valid but names a script that no
# longer exists.
report C1 "no legacy script names" "$(
    hits 'upgrade-sim|e2e-test|upgrade[- ]sim|(^|[^[:alnum:]_])sim($|[^[:alnum:]_])' | grep -v SIMULATED-V2-CHANGE || true
)"

# A target missing from .PHONY silently stops running once a file with its name
# appears; a leftover .PHONY entry is what a rename that forgot it looks like.
targets=$(sed -nE 's/^([A-Za-z0-9_-]+):([^=]|$).*/\1/p' Makefile | sort -u)
phony=$(sed -nE 's/^\.PHONY:[[:space:]]*//p' Makefile | tr ' ' '\n' | sed '/^$/d' | sort -u)
report C2 "Makefile .PHONY matches the defined targets" "$(
    comm -23 <(echo "$targets") <(echo "$phony") | sed 's/^/defined but not in .PHONY: /'
    comm -13 <(echo "$targets") <(echo "$phony") | sed 's/^/in .PHONY but not defined: /'
)"

# `make help` is hand-grouped, so a new target with a `## ` description is easy
# to leave out of it — and an undiscoverable target is one nobody runs.
documented=$(sed -nE 's/^([A-Za-z0-9_-]+):.*## .*/\1/p' Makefile | sort -u)
listed=$(make -s help | awk '/^  [A-Za-z]/ {print $1}' | sort -u)
report C3 "make help lists exactly the documented targets" "$(
    comm -23 <(echo "$documented") <(echo "$listed") | sed 's/^/documented but not in make help: /'
    comm -13 <(echo "$documented") <(echo "$listed") | sed 's/^/in make help but not documented: /'
)"

# The three e2e scripts are read and edited as one family; a skeleton that
# differs in one of them makes every cross-script edit a guessing game.
# facet FILE NAME prints one line describing how FILE handles NAME.
facet() {
    case $2 in
        "set line")         grep -m1 '^set -' "$1" || echo "(none)" ;;
        "sources lib.sh")   grep -q 'source .*lib\.sh' "$1" && echo yes || echo no ;;
        "line-2 comment")   sed -n 2p "$1" | grep -q '^# ' && echo yes || echo no ;;
        "uses step_header") grep -q 'step_header' "$1" && echo yes || echo no ;;
    esac
}
skeleton=""
for name in "set line" "sources lib.sh" "line-2 comment" "uses step_header"; do
    values=""
    for f in scripts/e2e-*.sh; do
        values="$values$(printf '%s: %s' "$f" "$(facet "$f" "$name")")"$'\n'
    done
    if [ "$(printf '%s' "$values" | sed 's/^[^:]*: //' | sort -u | wc -l)" -gt 1 ]; then
        skeleton="$skeleton$(printf '%s differs:\n%s' "$name" "$values" | sed '2,$s/^/  /')"$'\n'
    fi
done
report C4 "e2e scripts share one skeleton" "${skeleton%$'\n'}"

# Each e2e run deletes and recreates its working directories; one shared prefix
# keeps that cleanup away from anything else in /tmp and makes leftovers easy
# to recognise.
report C5 "e2e scratch paths use the /tmp/xpg-e2e- prefix" "$(
    grep -noE '/tmp/[^[:space:]"'"'"')]*' scripts/e2e-*.sh | grep -v ':/tmp/xpg-e2e-' || true
)"

# One concept, one word. These are the retired spellings from the terminology
# cleanup; if they reappear, two names for the same thing are back in circulation.
report C6 "retired terminology is gone" "$(
    hits 'stage [0-9]|ownership header|framework bump|tool[ -]file|user[ -]file'
)"

# An error string that starts with a bare verb ("build ...") reads as an
# instruction, and once wrapped it stutters ("build: failed to ..."); the
# repo's shape is a gerund ("building ..."). A denylist of the known bare verbs
# is the cheapest check that catches this. It scans the generator's own Go
# (fmt.Errorf/errors.New, not t.Errorf test messages) on a single line.
report C7 "error strings start with a gerund, not a bare verb" "$(
    hits '(fmt\.Errorf|errors\.New)\("(failed to|error|build|configure|set|load|get|scaffold|parse|init)([^[:alnum:]_]|$)' '*.go'
)"

# A workflow whose paths: filter misses a file it depends on silently skips the
# e2e run on the PRs that most need it. The logic lives in the Python script.
if out=$(python3 hack/check-workflow-paths.py 2>&1); then
    report C8 "workflow paths filters match the files each e2e workflow uses" ""
else
    report C8 "workflow paths filters match the files each e2e workflow uses" "${out:-hack/check-workflow-paths.py failed}"
fi

exit "$FAILED"
