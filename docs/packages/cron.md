---
package: github.com/genai-io/gen-code/internal/cron
layer: feature
---

# cron

Cron expression scheduler. Maintains a list of `Job`s with cron exprs and
prompts; the TUI's main loop calls `Tick()` periodically; due jobs are
returned for the loop to dispatch as agent messages.

## Purpose

The user-visible `/loop` and `/schedule` slash commands persist their
recurring or one-shot jobs here. Durable jobs survive process restart;
non-durable jobs are in-memory only.

## Contract

```go
package cron

// Service is the public contract for the cron module.
type Service interface {
    // CRUD
    Add(job Job) error
    Remove(id string) bool
    Create(cronExpr, prompt string, recurring, durable bool) (*Job, error)
    Delete(id string) error
    List() []*Job

    // runtime
    Tick() []FiredJob

    // query
    Empty() bool

    // lifecycle
    Reset()
    SetStoragePath(path string)
    LoadDurable() error
}
```

### Known Violations

- **Rule 1 (small).** 10 methods spanning CRUD, runtime tick, and
  lifecycle. Could split into `CronJobs` (CRUD/Query) and `CronRuntime`
  (Tick/Lifecycle).
- **`Add` and `Create` overlap.** Both produce a `Job` from input. `Add`
  takes a built `Job`; `Create` takes raw fields. The two-API surface
  invites confusion — pick one.
- **`Remove` returns bool, `Delete` returns error.** Different idioms
  for the same operation. Consolidate.
- **Rule 5.** `Default()` returns `Service`.
- **Singleton via `Default()`.**

## Internals

- `Store` (`store.go`) — concrete implementation. Holds the job map +
  optional `storagePath` for durable persistence.
- `cron.go` — `Job` struct, cron expression parsing, next-fire-time
  calculation.
- `loop.go` — internal loop used by tests; production code uses the
  TUI's tick instead.

## Lifecycle

- Construction: `Initialize(Options{StoragePath})` loads durable jobs
  from disk.
- Per-tick: the TUI calls `Tick()` once per UI tick; returned `FiredJob`s
  are converted to user messages and routed through the agent.
- Persistence: durable jobs are written to `<storagePath>` on Add/Delete.

## Tests

```
internal/cron/cron_test.go    — expression parsing, next-fire, durability.
internal/cron/loop_test.go    — tick semantics.
```

## See Also

- Code: `internal/cron/`
- Reference: [`reference/slash-commands.md`](../reference/slash-commands.md) (`/loop`, `/schedule`)
- Layer: `feature`
