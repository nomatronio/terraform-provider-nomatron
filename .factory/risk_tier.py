#!/usr/bin/env python3
"""Classify a change set against .factory/risk-paths.txt.

Highest matching tier wins. Unmatched files are medium. Changes to the path
list, this checker, or the security-review skill stay high even if the path
list says otherwise.
This script never merges a pull request.
"""

from __future__ import annotations

import os
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DEFAULT_PATHS = ROOT / ".factory" / "risk-paths.txt"
TIERS = ("low", "medium", "high")
RANK = {name: index for index, name in enumerate(TIERS)}

# The path list must not be able to demote a change to itself, the checker,
# or the security-review skill. Entries may be exact paths or globs.
FORCED_HIGH = frozenset(
    {
        ".factory/risk-paths.txt",
        ".factory/risk_tier.py",
        ".factory/risk_tier_test.py",
        ".factory/state-compatibility.md",
        ".github/workflows/risk-tier.yml",
        ".agents/skills/security-review/**",
    }
)

SHA_RE = re.compile(r"^[0-9a-fA-F]{40}$")

SUMMARY_LIMIT = 25


def glob_to_regex(pattern: str) -> re.Pattern[str]:
    """Translate a path glob. `**` crosses directories; `*` does not.

    Matching is case-insensitive: the pattern is lowercased before it is compiled.
    Callers lowercase the path before searching.
    """
    pattern = pattern.lower()
    parts: list[str] = ["^"]
    index = 0
    while index < len(pattern):
        if pattern.startswith("**/", index):
            parts.append("(?:.*/)?")
            index += 3
            continue
        if pattern.startswith("**", index):
            parts.append(".*")
            index += 2
            continue
        char = pattern[index]
        if char == "*":
            parts.append("[^/]*")
        elif char == "?":
            parts.append("[^/]")
        else:
            parts.append(re.escape(char))
        index += 1
    parts.append("$")
    return re.compile("".join(parts))


def is_forced_high(path: str) -> bool:
    """True when checker policy forces this path high, including globs.

    The path and each policy entry are compared case-insensitively.
    """
    lowered = path.lower()
    for entry in FORCED_HIGH:
        folded = entry.lower()
        if "*" in folded or "?" in folded:
            if glob_to_regex(folded).search(lowered):
                return True
        elif folded == lowered:
            return True
    return False


def load_rules(text: str) -> list[tuple[str, re.Pattern[str], str]]:
    rules: list[tuple[str, re.Pattern[str], str]] = []
    for line_number, raw in enumerate(text.splitlines(), start=1):
        line = raw.strip()
        if not line or line.startswith("#"):
            continue
        pieces = line.split(None, 1)
        if len(pieces) != 2:
            raise SystemExit(f"risk-paths:{line_number}: expected '<tier> <glob>'")
        tier, pattern = pieces
        if tier not in RANK:
            raise SystemExit(f"risk-paths:{line_number}: unknown tier {tier!r}")
        rules.append((tier, glob_to_regex(pattern), pattern))
    if not rules:
        raise SystemExit("risk-paths: no rules found")
    return rules


def best_rule(
    path: str, rules: list[tuple[str, re.Pattern[str], str]]
) -> tuple[str, str] | None:
    winner: tuple[int, str, str] | None = None
    lowered = path.lower()
    for tier, pattern, glob in rules:
        if pattern.search(lowered):
            rank = RANK[tier]
            if winner is None or rank > winner[0]:
                winner = (rank, tier, glob)
    if is_forced_high(path) and (winner is None or winner[0] < RANK["high"]):
        return ("high", "checker policy file; tier forced high")
    if winner is None:
        return None
    if is_forced_high(path):
        return ("high", f"matched `{winner[2]}` and checker policy forces high")
    return (winner[1], f"matched `{winner[2]}`")


def classify(
    paths: list[str], rules: list[tuple[str, re.Pattern[str], str]]
) -> tuple[str, list[tuple[str, str, str]]]:
    """Return the overall tier and per-file (path, tier, reason) rows."""
    rows: list[tuple[str, str, str]] = []
    overall = "low"
    for path in paths:
        cleaned = path.strip()
        if cleaned.startswith("./"):
            cleaned = cleaned[2:]
        if not cleaned:
            continue
        match = best_rule(cleaned, rules)
        if match is None:
            tier, reason = "medium", "no rule matched"
        else:
            tier, reason = match
        rows.append((cleaned, tier, reason))
        if RANK[tier] > RANK[overall]:
            overall = tier
    return overall, rows


def parse_name_status(text: str) -> list[str]:
    """Collect every path from `git diff --name-status -z`.

    Records are NUL-separated, so quotes and newlines in a path stay intact.
    A rename or copy lists the old path and the new path; both are returned.
    """
    parts = text.split("\0")
    if parts and parts[-1] == "":
        parts.pop()
    found: list[str] = []
    index = 0
    while index < len(parts):
        status = parts[index]
        if status.startswith(("R", "C")):
            if index + 2 >= len(parts):
                raise SystemExit("git diff name-status: truncated rename")
            found.append(parts[index + 1])
            found.append(parts[index + 2])
            index += 3
            continue
        if index + 1 >= len(parts):
            raise SystemExit("git diff name-status: truncated record")
        found.append(parts[index + 1])
        index += 2
    return found


def changed_files(base: str, head: str) -> list[str]:
    proc = subprocess.run(
        [
            "git",
            "diff",
            "--name-status",
            "-z",
            "--find-renames",
            "--end-of-options",
            base,
            head,
        ],
        cwd=ROOT,
        check=False,
        capture_output=True,
        text=True,
    )
    if proc.returncode != 0:
        detail = (proc.stderr or proc.stdout).strip()
        raise SystemExit(f"git diff failed: {detail}")
    return parse_name_status(proc.stdout)


def write_output(key: str, value: str) -> None:
    destination = os.environ.get("GITHUB_OUTPUT")
    if not destination:
        return
    with open(destination, "a", encoding="utf-8") as handle:
        handle.write(f"{key}={value}\n")


def write_summary(tier: str, rows: list[tuple[str, str, str]]) -> None:
    deciding = [row for row in rows if row[1] == tier]
    lines = [
        f"Risk tier: **{tier}**",
        "",
        f"Changed files: {len(rows)}. Files at this tier: {len(deciding)}.",
        "",
        "Auto-merge is not enabled. See `.factory/merge-policy.md`.",
    ]
    if deciding:
        lines.extend(["", "Why this tier:"])
        for path, _tier, reason in deciding[:SUMMARY_LIMIT]:
            lines.append(f"- `{path}` — {reason}")
        extra = len(deciding) - SUMMARY_LIMIT
        if extra > 0:
            lines.append(f"- {extra} more files at this tier omitted")
    elif not rows:
        lines.extend(["", "No changed files. Tier defaults to low."])
    text = "\n".join(lines) + "\n"
    print(text)
    destination = os.environ.get("GITHUB_STEP_SUMMARY")
    if destination:
        with open(destination, "a", encoding="utf-8") as handle:
            handle.write(text)


def validate_sha(name: str, value: str) -> None:
    """Reject anything other than a full 40-character hex object name."""
    if SHA_RE.fullmatch(value) is None:
        raise SystemExit(f"{name} must be exactly 40 hex characters")


def validate_file_arg(path: str) -> None:
    """Reject absolute paths and any path that climbs with '..'."""
    if os.path.isabs(path) or path.startswith(("/", "\\")):
        raise SystemExit(f"refusing absolute path: {path}")
    parts = re.split(r"[/\\]", path)
    if ".." in parts:
        raise SystemExit(f"refusing path that contains '..': {path}")


def main(argv: list[str]) -> int:
    paths_file = DEFAULT_PATHS
    files: list[str] | None = None
    index = 0
    while index < len(argv):
        arg = argv[index]
        if arg == "--paths-file":
            index += 1
            paths_file = Path(argv[index])
        elif arg == "--files":
            index += 1
            files = argv[index:]
            break
        else:
            raise SystemExit(f"unknown argument: {arg}")
        index += 1

    rules = load_rules(paths_file.read_text(encoding="utf-8"))
    if files is None:
        base = os.environ.get("BASE_SHA", "")
        head = os.environ.get("HEAD_SHA", "")
        validate_sha("BASE_SHA", base)
        validate_sha("HEAD_SHA", head)
        files = changed_files(base, head)
    else:
        for path in files:
            validate_file_arg(path)

    tier, rows = classify(files, rules)
    write_output("tier", tier)
    write_summary(tier, rows)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
