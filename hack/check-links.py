#!/usr/bin/env python3
"""C9: markdown links and anchors resolve. Run from repo root. Exit 1 on any miss.

Scans every tracked .md and .tmpl file (docs/plans/ excluded, like the rest of this
gate) for `](...)` links. A relative path must exist; a `#anchor` on a .md target
must match one of that file's heading slugs (GitHub's slug rules). An absolute
`.../blob/main/<path>` URL onto this repo is converted to `<path>` and checked the
same way; any other absolute URL (including one onto a different repo) is untouched
- checking it would mean a network call, which this gate does not make.

.tmpl files render into a *different* repo (a scaffolded provider), so a relative
link in one resolves there, not here: only the converted blob/main URLs in a .tmpl
file are checked; its relative links are skipped entirely.
"""
import os
import re
import subprocess
import sys

REPO_URL = "https://github.com/cychiang/xp-provider-gen/blob/main/"

files = subprocess.check_output(
    ["git", "ls-files", "*.md", "*.tmpl", ":!docs/plans"], text=True
).split()
files = [f for f in files if f != "CLAUDE.md"]  # symlink to AGENTS.md, not a second copy


def slug(heading):
    # GitHub's heading-to-anchor rule: strip markdown emphasis/links, lowercase,
    # drop anything but word chars/space/hyphen, spaces become hyphens.
    h = heading.strip().lower()
    h = re.sub(r"[`*_~]", "", h)
    h = re.sub(r"\[([^\]]*)\]\([^)]*\)", r"\1", h)
    h = re.sub(r"[^\w\- ]", "", h)
    return h.replace(" ", "-")


_anchor_cache = {}


def anchors(path):
    if path in _anchor_cache:
        return _anchor_cache[path]
    seen, counts, in_fence = set(), {}, False
    try:
        lines = open(path, encoding="utf-8").readlines()
    except OSError:
        lines = []
    for line in lines:
        if line.startswith("```"):
            in_fence = not in_fence
            continue
        if in_fence:
            continue
        m = re.match(r"^(#{1,6})\s+(.*)", line)
        if m:
            s = slug(m.group(2))
            n = counts.get(s, 0)
            counts[s] = n + 1
            seen.add(s if n == 0 else f"{s}-{n}")
    _anchor_cache[path] = seen
    return seen


def check_target(src, target, base_dir):
    """base_dir is where a relative `path` resolves against - the source file's own
    directory for an ordinary link, or the repo root for a converted blob/main URL.
    Returns None if OK, else a BROKEN reason string."""
    path, _, anchor = target.partition("#")
    resolved = src if path == "" else os.path.normpath(os.path.join(base_dir, path))
    if not os.path.exists(resolved):
        return f"path does not exist: {resolved}"
    if anchor and resolved.endswith(".md") and anchor not in anchors(resolved):
        return f"no such anchor #{anchor} in {resolved}"
    return None


bad = 0
for f in files:
    is_tmpl = f.endswith(".tmpl")
    for i, line in enumerate(open(f, encoding="utf-8"), 1):
        for m in re.finditer(r"\]\(([^)\s]+)\)", line):
            target = m.group(1)
            if target.startswith(REPO_URL):
                reason = check_target(f, target[len(REPO_URL):], base_dir=".")
            elif target.startswith(("http://", "https://", "mailto:")):
                continue  # external URL (including a different repo) - not fetched
            elif is_tmpl:
                continue  # a relative link in a .tmpl resolves in the generated project
            else:
                reason = check_target(f, target, base_dir=os.path.dirname(f))
            if reason:
                print(f"BROKEN {f}:{i}: {target} ({reason})")
                bad = 1

sys.exit(bad)
