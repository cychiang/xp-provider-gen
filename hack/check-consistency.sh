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
# A search that cannot run (say, a malformed PATTERN) is reported as a hit, so
# a broken check turns red instead of passing.
hits() {
    local pattern=$1 out rc=0
    shift
    out=$(git grep --untracked -nEi "$pattern" -- "${@:-.}" ':!docs/plans' ':!hack/check-consistency.sh') || rc=$?
    # git grep: 0 = hits, 1 = none, anything else = it could not search.
    [ "$rc" -le 1 ] || out="git grep failed (rc=$rc) for pattern: $pattern"
    printf '%s' "$out"
}

# Names are stable only if nothing still points at the old ones: a stale
# reference in a doc or workflow reads as valid but names a script that no
# longer exists — and a file that still carries an old name is the same drift
# in the other direction. The separator may be `-`, `_` or nothing (hits is
# case-insensitive, so upgradeSim and UPGRADE_SIM are covered too); "upgrade sim"
# with a space is a legacy name as well, but "e2e test" is ordinary English.
# Known trade-off: this also blocks file names like `_e2e_test.go` and `sim.go`;
# narrow it when one is really needed.
LEGACY='upgrade[-_ ]?sim|e2e[-_]?test|(^|[^[:alnum:]_])sim($|[^[:alnum:]_])'
legacy_files=$(git ls-files --cached --others --exclude-standard -- . ':!docs/plans' ':!hack/check-consistency.sh' |
    { grep -Ei "$LEGACY" || true; } | sed 's/$/: file name carries a legacy name/') || legacy_files="listing file names failed"
report C1 "no legacy script names" "$(
    hits "$LEGACY" | grep -v SIMULATED-V2-CHANGE || true
    printf '%s' "$legacy_files"
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

# One concept, one word. These are the retired spellings from the terminology
# cleanup; if they reappear, two names for the same thing are back in circulation.
report C6 "retired terminology is gone" "$(
    hits 'stages? [0-9]|ownership[ -]header|framework (bump|change)|tool[ -]file|user[ -]file'
)"

# An error string that starts with a bare verb ("build ...") reads as an
# instruction, and once wrapped it stutters ("build: failed to ..."); the
# repo's shape is a gerund ("building ..."). A denylist of bare verbs is the
# cheapest check that catches this: it is the bare form of every gerund the repo
# already opens an error string with, plus the "failed to"/"unable to" phrasings.
# Extend it when a new verb shows up. It scans the generator's own Go
# (fmt.Errorf/errors.New, not t.Errorf test messages, not *.go.tmpl), whether the
# string opens on the call's line or — as gofmt leaves a long message — on the next.
BARE_VERBS='failed to|unable to|could not|error|adopt|build|check|configure|create|decode|derive|discover|encode|enumerate|fetch|get|init|load|open|parse|read|reconcile|record|refuse|render|run|scaffold|set|stamp|start|work|write'
# awk keeps one line of memory: `open` is true when the previous line ended at the
# call's opening parenthesis, so this line starts the string. The /dev/null
# argument keeps awk from reading stdin when xargs has no files to pass (GNU xargs
# still runs it once). A scan that cannot run is reported as a violation, like
# `hits` does.
bare_verb_errors=$(git ls-files --cached --others --exclude-standard -z -- '*.go' | xargs -0 awk -v verbs="$BARE_VERBS" '
    FNR == 1 { open = 0 }
    {
        line = tolower($0)
        call = "(fmt\\.errorf|errors\\.new)\\("
        opens = "(" verbs ")([^[:alnum:]_]|$)"
        re = open ? "^[[:space:]]*\"" opens : call "\"" opens
        if (line ~ re) print FILENAME ":" FNR ":" $0
        open = (line ~ (call "[[:space:]]*$"))
    }' /dev/null) || bare_verb_errors="scanning *.go for error strings failed"
report C7 "error strings start with a gerund, not a bare verb" "$bare_verb_errors"

# A workflow whose paths: filter misses a file it depends on silently skips the
# e2e run on the PRs that most need it. The logic lives in the Python script.
if out=$(python3 hack/check-workflow-paths.py 2>&1); then
    report C8 "workflow paths filters match the files each e2e workflow uses" ""
else
    report C8 "workflow paths filters match the files each e2e workflow uses" "${out:-hack/check-workflow-paths.py failed}"
fi

# A relative markdown link or #anchor that no longer resolves reads as valid but
# sends a reader nowhere. The logic lives in the Python script.
if out=$(python3 hack/check-links.py 2>&1); then
    report C9 "markdown links and anchors resolve" ""
else
    report C9 "markdown links and anchors resolve" "${out:-hack/check-links.py failed}"
fi

exit "$FAILED"
