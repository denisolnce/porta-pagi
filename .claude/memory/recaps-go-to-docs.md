---
name: recaps-go-to-docs
description: Every turn ends with its state, decisions, how-tos and owner to-dos in the repo, technical and ideological, never only in chat (2026-09-07, widened 2026-09-28 and 2026-10-07)
metadata:
  type: feedback
---

The rule itself is `CLAUDE.md` § Docs stay current and rule 5 at its top.
This file keeps the owner's words behind it, which no doc records.

- 2026-09-07, after a recap of what is built and what could come next:
  "make sure it's recorded in this repo". Conversations, not only code
  changes, are held to the docs-currency rule.
- 2026-09-28, after the steps for wiring feedback email were given only in
  chat: "record it in repo for cold starts to know whats up. always record
  stuff like this at the end of turns".
- 2026-10-07, when this checkout became the desk for all four repos: every
  agent working here updates the docs on technical and ideological things
  after each turn, so a cold start is fast and finds everything current.

**Why:** the owner works in evening sessions and restarts cold; a fact left
in a chat is lost to the next session.

**How to apply:** before ending a turn, walk what it built, decided,
deferred or told the owner, and put each in the doc that owns it, in the
repo that owns it, sibling repos included. The reviewer checks it; the stop
hook catches only turns that wrote a file.
