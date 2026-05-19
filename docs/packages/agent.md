---
package: github.com/genai-io/gen-code/internal/agent
layer: feature
---

# agent

Owns the **main agent session lifecycle** — construction, start, stop, and
TUI-facing send/permission/outbox plumbing for the single foreground agent.

## Purpose

`internal/app` runs exactly one foreground agent session at a time. This
package is the seam between that TUI shell and the underlying agent loop in
[`packages/core.md`](core.md). The shell starts a session, hands user input
to it, observes its outbox, and routes permission requests back to the user.

Subagents (parallel background agents) are owned separately by
[`packages/subagent.md`](subagent.md); cron and async triggers feed into the
same `Send` path used by user input.

## Contract

The seam consumed by `internal/app`:

```go
package agent

// Service manages the main agent session lifecycle.
type Service interface {
    // Start builds a core.Agent from params, starts its goroutine.
    // If messages is non-empty, they are loaded as conversation history.
    Start(params BuildParams, messages []core.Message) error

    // Stop cancels the agent goroutine and cleans up.
    Stop()

    // Active reports whether an agent session is running.
    Active() bool

    // Send pushes a user message to the agent's inbox. No-op if not active.
    Send(content string, images []core.Image)

    // Outbox returns the agent's event channel. Nil if not active.
    Outbox() <-chan core.Event

    // PermissionBridge returns the current session's permission bridge.
    PermissionBridge() *PermissionBridge

    // PendingPermission gets the pending permission request.
    PendingPermission() *PermBridgeRequest

    // SetPendingPermission tracks a pending permission request for TUI approval.
    SetPendingPermission(req *PermBridgeRequest)

    // System returns the running agent's system prompt for hot-patching
    // (e.g. swapping identity mid-session). Returns nil if no agent is
    // active. Mutations are visible on the next inference call.
    System() core.System
}
```

The underlying primitive — `core.Agent` with its Inbox/Outbox channels —
lives in [`packages/core.md`](core.md). This package wraps a single
`core.Agent` per running session.

### Known Violations

These break the [Contract Rules](TEMPLATE.md#contract-rules) in the
template. They are tracked here for a PR-3 cleanup; the present doc
faithfully reflects today's code.

- **Rule 1 (small).** `Service` has **11 methods** covering four concerns:
  lifecycle (`Start`/`Stop`/`Active`), I/O (`Send`/`Outbox`), permission
  bridge (`PermissionBridge`/`PendingPermission`/`SetPendingPermission`),
  hot-patching (`System`). Suggested split:
  - `AgentLifecycle` → `Start`, `Stop`, `Active`
  - `AgentIO` → `Send`, `Outbox`
  - `PermissionGate` → the three bridge methods
  - `SystemPatcher` → `System`
  Each call site narrows to the interface it actually uses; the concrete
  `*service` keeps implementing all of them.
- **Rule 5 (constructors return concrete types).** `Default()` returns
  `Service` rather than the concrete `*service`. Callers can never reach
  unexported fields and tests can't substitute behavior except through
  `SetDefault`. Should return `*service` once exported, or split as above.
- **Singleton via `Default()`.** Hidden global state; not strictly an
  interface rule but it blocks the consumer-side interface pattern.
  Construction should move into `internal/app/services.go`.

## Internals

- `service` (`session.go`) is the only implementation. It tracks one
  `*core.Agent` plus its cancellation context, a `PermissionBridge`, and a
  pending `PermBridgeRequest` (TUI approval handshake).
- `build.go` translates `BuildParams` (model, identity, skills, tools,
  permission mode, cwd, ...) into a `core.Config` for `core.NewAgent`.
- `permission.go` owns the bridge: a thread-safe channel pair that turns
  asynchronous permission asks into synchronous TUI approval modals.
- No persistence here — session/transcript state lives in
  [`packages/session.md`](session.md).

## Lifecycle

- Construction: `Initialize(Options{})` runs at app startup, registering the
  singleton.
- Per-session: `Start(params, messages)` builds a `core.Agent` and launches
  its `Run` goroutine. The agent's outbox is the only return channel.
- Termination: `Stop()` cancels the run context. Outbox closes via
  `core.Agent` shutdown. `Active()` flips to false.
- Reentrancy: methods are guarded by a mutex; concurrent `Send` from the
  user-input goroutine and the cron/trigger goroutines is the design
  intent.

## Tests

```
internal/agent/                — no package-level test file today.
                                  Coverage is exercised end-to-end via
                                  internal/app and integration tests.
```

A unit test for `BuildParams → core.Config` translation is missing and
worth adding (logged in `notes/tech-debt.md`).

## See Also

- Code: `internal/agent/`
- Underlying primitive: [`packages/core.md`](core.md) (the `Agent`
  interface and the inbox/outbox event model)
- Background agents: [`packages/subagent.md`](subagent.md)
- Permission model: [`concepts/permission-model.md`](../concepts/permission-model.md)
- Layer: `feature` (see [`reference/dependency-rules.md`](../reference/dependency-rules.md))
