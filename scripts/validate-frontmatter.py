#!/usr/bin/env python3
"""
Custom frontmatter + filing-standard validator.

Validates markdown documents in 000-docs/ against:
  1. Document Filing System v4.3 (/doc-filing skill) — filename pattern
     NNN-CC-ABCD-short-description.ext (NNN 001-999) or
     000-CC-ABCD-short-description.ext (canonical cross-repo standards)
  2. If YAML frontmatter is present (delimited by --- lines), it must parse.

Allowlisted (skipped): README.md, CLAUDE.md, CONTRIBUTING.md, CHANGELOG.md,
LICENSE.md and anything outside 000-docs/. Legacy 6767-prefixed files emit a
warning but do not fail the gate (pre-v4.3 cruft, scheduled for renaming).

Exit code 0 = pass; non-zero = at least one error. Warnings never fail the gate.
"""

from __future__ import annotations

import argparse
import pathlib
import re
import sys
from typing import Iterable

# NNN-CC-ABCD-description.ext where NNN is 001-999 (with optional letter suffix
# like 005a) OR exactly 000 for canonical cross-repo standards. CC is two
# uppercase letters; ABCD is four uppercase letters; description is kebab-case
# lowercase alphanumerics plus hyphens (1-4 words enforced loosely as <= 60
# chars). Sub-doc numeric suffix variant (006-1) is allowed.
FILENAME_RE = re.compile(
    r"^(?:000|[0-9]{3}[a-z]?)(?:-[0-9]+)?-[A-Z]{2}-[A-Z]{4,6}-[a-z0-9][a-z0-9-]{0,59}\.[a-z]{2,5}$"
)

# v4.2/legacy pattern — files prefixed `6767-` were the canonical cross-repo
# series before v4.3; they now live under the 000- prefix. Warn rather than
# fail so existing repos can migrate at their own pace.
LEGACY_6767_RE = re.compile(r"^6767-[a-z]?-?[A-Z]{2}-[A-Z]{4}-")

# Pre-v4 layout commonly used NNN-XXX-description.ext where XXX was a 2-4 letter
# doc-type abbreviation (PRD, ADR, API, RFC, etc.) without a separate CC
# category code. Warn so existing repos can migrate; do not fail the gate.
LEGACY_3LETTER_RE = re.compile(r"^[0-9]{3}[a-z]?-[A-Z]{2,4}-[a-z0-9-]+\.[a-z]{2,5}$")

# Files allowlisted regardless of pattern. The docstring promises these are
# skipped; we keep the implementation aligned defensively in case the validator
# is ever pointed at a parent dir that contains repo-root meta files.
ALLOWLIST_NAMES = frozenset({
    "README.md",
    "INDEX.md",
    "CLAUDE.md",
    "CONTRIBUTING.md",
    "CHANGELOG.md",
    "LICENSE.md",
    "AGENTS.md",
    "SECURITY.md",
    "CODE_OF_CONDUCT.md",
})


def walk_docs(roots: Iterable[pathlib.Path]) -> Iterable[pathlib.Path]:
    for root in roots:
        if not root.exists():
            continue
        if root.is_file():
            yield root
            continue
        for p in sorted(root.rglob("*")):
            if p.is_file() and p.suffix.lower() in {
                ".md",
                ".pdf",
                ".doc",
                ".docx",
                ".txt",
                ".xlsx",
                ".xls",
                ".csv",
                ".ppt",
                ".pptx",
            }:
                yield p


def validate_frontmatter(path: pathlib.Path) -> list[str]:
    """Return list of error strings for this file. Empty = OK."""
    errors: list[str] = []
    if path.suffix.lower() != ".md":
        return errors
    try:
        text = path.read_text(encoding="utf-8")
    except UnicodeDecodeError as exc:
        return [f"{path}: not valid UTF-8 ({exc})"]
    if not text.startswith("---\n"):
        return errors  # no frontmatter; that is fine
    # Match the closing fence either as `\n---\n` (frontmatter followed by body)
    # OR `\n---` at end-of-string (file ends exactly at the closing fence with
    # no trailing newline). Without the EOS alternative, files like a stand-alone
    # `---\nkey: value\n---` get rejected as missing-delimiter.
    match = re.search(r"\n---\n|\n---\Z", text[4:])
    if not match:
        return [f"{path}: opening --- but no closing --- delimiter"]
    body = text[4 : 4 + match.start()]
    # Minimal YAML parse — avoid PyYAML dep. Just confirm keys look like KEY: value
    # or list/object indent. Reject lines that contain only a stray colon-less token.
    for lineno, raw in enumerate(body.splitlines(), start=2):
        line = raw.rstrip()
        if not line or line.startswith("#"):
            continue
        if line.startswith(" ") or line.startswith("\t") or line.startswith("-"):
            continue  # nested key or list item
        if ":" not in line:
            errors.append(
                f"{path}:{lineno}: frontmatter line lacks ':' separator: {line!r}"
            )
    return errors


def validate_filename(path: pathlib.Path) -> tuple[list[str], list[str]]:
    """Return (errors, warnings) for this file's name."""
    errors: list[str] = []
    warnings: list[str] = []
    name = path.name
    if name in ALLOWLIST_NAMES:
        return errors, warnings
    if LEGACY_6767_RE.match(name):
        warnings.append(
            f"{path}: legacy 6767- prefix (Document Filing Standard v4.2); "
            f"v4.3 canonical prefix is 000-. Rename when convenient."
        )
        return errors, warnings
    if LEGACY_3LETTER_RE.match(name):
        warnings.append(
            f"{path}: legacy NNN-XXX- pre-v4 pattern (single doc-type code, no "
            f"separate CC category). v4.3 wants NNN-CC-ABCD-description. "
            f"Rename when convenient."
        )
        return errors, warnings
    if not FILENAME_RE.match(name):
        errors.append(
            f"{path}: filename does not match Document Filing Standard v4.3 pattern "
            f"NNN-CC-ABCD-description.ext (NNN=001-999 or 000 for cross-repo standards; "
            f"CC=two uppercase letters; ABCD=four uppercase letters; "
            f"description=kebab-case lowercase)"
        )
    return errors, warnings


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "paths",
        nargs="*",
        default=["000-docs"],
        help="Files or directories to validate (default: 000-docs)",
    )
    parser.add_argument(
        "--warnings-as-errors",
        action="store_true",
        help="Fail the gate on legacy-prefix warnings (default: warn only)",
    )
    args = parser.parse_args()
    roots = [pathlib.Path(p) for p in args.paths]
    total_errors = 0
    total_warnings = 0
    for path in walk_docs(roots):
        name_errors, name_warnings = validate_filename(path)
        fm_errors = validate_frontmatter(path)
        for e in name_errors + fm_errors:
            print(f"ERROR: {e}", file=sys.stderr)
            total_errors += 1
        for w in name_warnings:
            print(f"WARN:  {w}", file=sys.stderr)
            total_warnings += 1
    print(
        f"validate-frontmatter: {total_errors} error(s), {total_warnings} warning(s)",
        file=sys.stderr,
    )
    if total_errors:
        return 1
    if total_warnings and args.warnings_as_errors:
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
