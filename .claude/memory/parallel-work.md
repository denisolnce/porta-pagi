---
name: parallel-work
description: Parallel agents go in git worktrees, i18n dictionaries merge three-way by key; Playwright on the owner's Mac is pinned to 1.52
metadata:
  type: project
---

Parallel agents work best in git worktrees of this repo, one branch each,
merged back to `main` by the session that started them. The i18n
dictionaries (`backend/internal/httpapi/i18n/*.json`) conflict on every such
merge, because each agent appends keys at the end: merge them three-way by
key (keep additions from both sides, honour a deletion on either), never by
line. Watch for two agents writing the same test helper or struct field —
vet finds the first, review the second. Symlink `frontend/node_modules`
into a worktree rather than installing; `.gitignore` ignores the link.

A second Claude session may also work in this same checkout at the same
time (it happened 2026-10-07). Commit a section before another commit points
at it, and read only your own commits when reviewing.

Playwright on the owner's Mac (macOS 13) needs `playwright@1.52`: newer
releases ship no browser build for that macOS.

**Why:** learnt by hand, not derivable from the code (recorded 2026-10-07,
from the handoff note that sessions started here replaced).

**How to apply:** before fanning out agents, before merging their branches,
and before installing a browser for screenshots.
