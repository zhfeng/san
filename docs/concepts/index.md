# Concepts

Cross-cutting concepts that don't belong to a single package. Each page
explains how multiple packages collaborate around one idea; package-level
detail lives in [`packages/`](../packages/).

## Available Concepts

- [extension-model.md](extension-model.md) — the four extension primitives
  (skill / subagent / slash command / hook) plus plugin as a packaging
  source.
- [harness-channels.md](harness-channels.md) — three channels for
  delivering context to the model (system prompt, `<system-reminder>`,
  user messages), why they're separate, and how compaction interacts.
- [permission-model.md](permission-model.md) — tool-call permission
  pipeline, mode policy, foreground vs subagent differences, plan mode.

## When to Write a Concept Page

Most things should be documented in [`packages/<pkg>.md`](../packages/) — one
package, one page, one Contract. A concept page is warranted only when:

- Three or more packages collaborate around the idea.
- The reader needs to understand the collaboration before any single
  package's docs make sense.
- The content would distort one package's page if pinned there.

Don't promote single-package internals to a concept page. If a topic
fits in one package's "Internals" or "Lifecycle" section, keep it there.
