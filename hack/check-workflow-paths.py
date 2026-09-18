#!/usr/bin/env python3
"""A5: workflow paths: <-> files actually used. Run from repo root. Exit 1 on any miss."""
import re, subprocess, sys, yaml
from fnmatch import fnmatch

ENTRY = {   # workflow -> the scripts it runs, directly or via make
    ".github/workflows/e2e-native-full.yml": ["scripts/e2e-native.sh", "scripts/e2e-upgrade.sh"],
    ".github/workflows/e2e-upjet.yml": ["scripts/e2e-upjet.sh"],
}
COMMON = ["pkg/templates/generators", "pkg/templates/loader.go",
          "pkg/plugins/crossplane/v2", "pkg/versions", "Makefile"]
INPUTS = {  # per-flavor render inputs -- not derivable from the scripts, so listed explicitly
    ".github/workflows/e2e-native-full.yml": ["pkg/templates/files"] + COMMON,
    ".github/workflows/e2e-upjet.yml": ["pkg/templates/upjet"] + COMMON,
}
# Known limitation: adding a new template root without adding it to INPUTS
# above will not be caught by this check.
tracked = subprocess.check_output(["git", "ls-files"], text=True).split("\n")
match = lambda p, f: f == p or fnmatch(f, p.replace("/**", "/*"))   # Python's * crosses /
bad = 0
for wf, entries in ENTRY.items():
    doc = yaml.safe_load(open(wf))
    pats = (doc.get("on") or doc.get(True))["pull_request"]["paths"]  # PyYAML reads `on:` as True
    for p in pats:                                   # forward: every pattern hits at least one file
        if not any(match(p, f) for f in tracked):
            print(f"DEAD   {wf}: {p}"); bad = 1
    need = set(entries) | {wf}
    for s in entries:                                # reverse 1: scripts/, hack/ referenced by the script
        for m in re.findall(r"(?:scripts|hack)/[A-Za-z0-9_./-]+", open(s).read()):
            m = m.rstrip("./")
            need |= {f for f in tracked if f == m or f.startswith(m + "/")}
    for root in INPUTS[wf]:                          # reverse 2: render inputs
        need |= {f for f in tracked if f == root or f.startswith(root + "/")}
    for f in sorted(need):
        if not any(match(p, f) for p in pats):
            print(f"MISSED {wf}: {f}"); bad = 1
sys.exit(bad)
