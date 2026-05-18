# Harness Channels

Gen Code delivers context to the model through **three distinct channels**,
each with different cache, lifecycle, and stability properties:

| Channel | What lives there | Cache-friendly? | Mutable mid-session? |
|---|---|---|---|
| **System prompt** | Identity, policy, tool schemas, active-skill list, slot-sectioned blocks. | Yes — invariant per session unless a section mutates. | Yes (Use/Drop), but expensive (cache miss). |
| **`<system-reminder>` blocks** | Session-level / project-level dynamic content: enabled-skills directory, GEN.md/CLAUDE.md memory, ad-hoc notices. | Yes — once attached, the user message is immutable. | No (re-emitted as new attachments, never mutated). |
| **User messages** | The actual prompt the user typed. | Yes — already cached. | No. |

The harness chooses which channel to use based on **how often the content
changes** and **whether the LLM's prompt cache should survive the
change**.

## Why Three Channels?

The LLM's prompt cache works on **exact prefix match**. Anything in the
system prompt that mutates invalidates the cache prefix from that point
onward — so frequent system-prompt edits are expensive.

The harness optimizes:

- **System prompt** = "things true for every turn of this session".
  Identity, policy, output style, base tool definitions. Mutates rarely.
- **`<system-reminder>` blocks** = "things true now, but may change". Each
  reminder is attached to a *user message* (not the system prompt) and
  re-emitted on session start and after every PostCompact. Because user
  messages are immutable once attached, the cache from prior turns stays
  valid; only the new user message + reminder is freshly evaluated.
- **User messages** = actual user input.

## System Prompt: Slot Sections

The system prompt is composed of **Sections**, each owning a numbered
**Slot**. Slots define ordering. Sections within the same slot use
insertion order. Mutations to a section trigger `Refresh` (lazy
re-render).

```
slot 0   identity         (built-in or custom persona)
slot 1   environment      (cwd, git status, plan-mode notice)
slot 2   policy           (built-in coding policy)
slot 3   tools            (rendered tool schemas)
slot 4   active-skills    (skills with State=active)
slot 5   harness          (current mode, output style)
...
```

See `internal/core/system/builder.go` and the `Section` and `System`
types in [`packages/core.md`](../packages/core.md).

## Reminders

Reminders carry "session-level" or "project-level" mutable content. The
harness has standard providers:

| Provider ID | Source | Re-emit triggers |
|---|---|---|
| `skills-directory` | enabled skills list | session start, PostCompact, skill enable/disable |
| `memory-user` | `~/.gen/GEN.md` and `~/.claude/CLAUDE.md` | session start, PostCompact, file change |
| `memory-project` | `<project>/GEN.md` and `<project>/CLAUDE.md` | session start, PostCompact, file change, cwd change |

Each provider has a stable ID; re-emitting from the same ID **drops the
previous queued entry**, so toggling a skill three times in a row
produces one final reminder, not three.

Reminders wrap their body in:

```xml
<system-reminder source="skills-directory">
  Enabled skills:
  - github:create-pr — ...
  - jira:link-ticket — ...
</system-reminder>
```

The LLM is instructed (in the system prompt) to treat the
`<system-reminder>` tag as a system instruction even though it appears
inside a user message.

Implementation: [`packages/reminder.md`](../packages/reminder.md).

## Memory: GEN.md / CLAUDE.md

Two memory tiers:

- **User memory**: `~/.gen/GEN.md` (Gen Code) and `~/.claude/CLAUDE.md`
  (Claude Code compat). Loaded once per session, attached as
  `memory-user` reminder.
- **Project memory**: `<project>/GEN.md`, `<project>/CLAUDE.md`, plus
  recursively-loaded `<dir>/GEN.md` upwards from the start path.
  Attached as `memory-project` reminder.

Memory is **never** in the system prompt — that would invalidate the
prompt cache every time the user edited their memory file.

## Compaction

When the context window approaches its limit, the harness compacts:

1. Pick the prefix of messages to summarize (everything except the most
   recent N turns).
2. Call the LLM with a "summarize the following conversation" prompt to
   produce a `CompactInfo` summary.
3. Replace the prefix with a single synthetic message containing the
   summary.
4. Re-emit all reminders (`EnqueueAllProviders`) so the post-compact
   conversation has fresh skill/memory context.

Compaction is **not** a channel by itself — it's a mutation of the
user-message channel. The reminder re-emission step is what makes
compaction safe across the reminder channel.

Implementation: `internal/app/conv/compact.go`. The agent emits
`OnCompact` events for observers.

## See Also

- [`concepts/extension-model.md`](extension-model.md) — skills (one
  reminder source) and how plugins contribute to it.
- [`packages/core.md`](../packages/core.md) — `System`, `Section`, slot
  layout.
- [`packages/reminder.md`](../packages/reminder.md) — runtime API.
- [`packages/session.md`](../packages/session.md) — how compaction
  records flow into the transcript.
