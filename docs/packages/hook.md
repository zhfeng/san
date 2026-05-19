---
package: github.com/genai-io/gen-code/internal/hook
layer: feature
---

# hook

Executes user-defined hooks at well-known application events (tool calls,
session lifecycle, permission requests, file changes, …) and merges their
outcomes back into the calling code path.

## Purpose

Hooks are how users extend Gen Code without writing Go: a shell command, an
LLM prompt, an HTTP endpoint, or an in-memory callback runs at a named
event. The engine resolves which hooks fire (matchers), runs them
synchronously or asynchronously, and reduces their structured outputs into
a single `HookOutcome` for the call site.

This package is intentionally separate from `core` agent lifecycle: hook
events are **application-layer** concerns (`PreToolUse`, `Stop`,
`SessionStart`, …), not part of the agent loop's primitives.

## Contract

The seam consumed by `internal/app` and feature packages that fire hooks:

```go
package hook

// Service is the public contract for the hook module.
type Service interface {
    // execution
    Execute(ctx context.Context, event EventType, input HookInput) HookOutcome
    ExecuteAsync(event EventType, input HookInput)
    FilterToolCalls(ctx context.Context, calls []core.ToolCall, agentID, agentType string) FilterToolCallsResult

    // query
    HasHooks(event EventType) bool
    StopHookActive() *bool
    CurrentStatusMessage() string

    // reconfigure (after session/provider/cwd change)
    SetSettings(settings *setting.Settings)
    SetLLMCompleter(fn LLMCompleter, model string)
    SetTranscriptPath(path string)
    SetCwd(cwd string)
    SetPermissionMode(mode string)
    SetPromptCallback(cb PromptCallback)
    SetAsyncHookCallback(cb AsyncHookCallback)
    SetEnvProvider(fn func() []string)

    // session-scoped hooks
    AddSessionFunctionHook(event EventType, matcher string, hook FunctionHook) string
    AddRuntimeFunctionHook(event EventType, matcher string, hook FunctionHook) string
    ClearSessionHooks()

    // lifecycle
    Wait()

    Engine() *Engine
}

// HookInput / HookOutput / HookOutcome / PermissionUpdate / FunctionHook /
// PromptCallback / AsyncHookCallback ... — see types.go for the full
// payload schema.
```

### Known Violations

The current `Service` is the worst offender in the repository against the
[Contract Rules](TEMPLATE.md#contract-rules). Tracked for PR-3 cleanup; the
contract above is verbatim from today's code.

- **Rule 1 (small).** **16 methods.** It bundles execution, querying,
  reconfiguration, registration, lifecycle, *and* an escape hatch into
  one interface. Suggested split, each <= 3 methods:
  - `HookExecutor` → `Execute`, `ExecuteAsync`, `FilterToolCalls`
  - `HookQuery` → `HasHooks`, `StopHookActive`, `CurrentStatusMessage`
  - `HookRegistrar` → `AddSessionFunctionHook`,
    `AddRuntimeFunctionHook`, `ClearSessionHooks`
  - `HookConfigurator` → the eight `Set*` methods, ideally collapsed
    into one `Reconfigure(Options)` that takes a struct
  - `HookLifecycle` → `Wait`
  - Remove `Engine()` entirely (see next rule)
- **Rule 7 (no wrapper-only interfaces & no escape hatches).** `Engine()
  *Engine` lets every caller bypass the interface and reach into the
  concrete `*Engine`. The interface then no longer constrains anything.
  Either delete `Engine()` (and the implementation
  `func (e *Engine) Engine() *Engine { return e }`), or stop pretending
  there is a seam and expose `*Engine` directly.
- **Rule 5 (constructors return concrete types).** `Default()` returns
  `Service`. `Initialize` constructs `*Engine` internally then up-casts.
  Callers cannot reach engine-only methods (`SetAuditCallback`) without
  `Engine()` — which is why the escape hatch exists in the first place.
- **Singleton via `Default()` and `DefaultIfInit()`.** Two flavors of
  singleton accessor signal that callers are racing initialization.
  Construction should move into the app composition root and be passed in
  explicitly.

## Internals

- `Engine` (`engine.go`) is the only implementation. It owns:
  - `*hookStore` — settings-loaded hooks plus session/runtime function hooks
  - `*statusTracker` — currently-active hook status message for the TUI
  - mutable knobs (`settings`, `cwd`, `transcriptPath`, `permissionMode`,
    `llmCompleter`, `httpClient`, `promptCallback`, `asyncCallback`,
    `auditCallback`, `envProvider`) under one `sync.RWMutex`
  - a `sync.WaitGroup` for fire-and-forget detached goroutines
- Executors live in `executors_command.go` / `executors_http.go` /
  `executors_llm.go`. Each takes a matched hook + `HookInput` and returns a
  `HookOutcome`.
- `matcher.go` resolves which hooks fire for an event by matching against
  patterns (tool name globs, regex, exact match).
- `audit` callback is the single observation seam used by the session
  recorder to write one `hook.fired` transcript record per invocation —
  the hook package does not import `transcript`.

## Lifecycle

- Construction: `Initialize(Options{...})` runs at app startup. Engine is a
  singleton.
- Per-event execution: `Execute` is synchronous; matching hooks run in
  configuration order, outcomes are reduced left-to-right, and the first
  `ShouldContinue=false` short-circuits.
- Async hooks: `ExecuteAsync` plus the `Command.Async` flag on individual
  hooks both spawn detached goroutines tracked by an internal
  `sync.WaitGroup`. `Wait()` blocks until all detached goroutines drain
  (used on app shutdown).
- Concurrency: all `Set*` methods are mutex-guarded reads/writes; hook
  execution reads under RLock.

## Tests

```
internal/hook/hooks_test.go         — large table of execution
                                       scenarios (sync, async, matchers,
                                       outcome merging, permission paths).
internal/hook/hooks_test.go         — also covers types/registry roundtrips.
```

The 49 KB test file is the canonical reference for how outcomes merge and
how Claude-Code-compatible hook configs are interpreted.

## See Also

- Code: `internal/hook/`
- Concepts: [`concepts/permission-model.md`](../concepts/permission-model.md)
- Concepts: [`concepts/extension-model.md`](../concepts/extension-model.md)
- Related: [`packages/tool.md`](tool.md) (tool permission gate), [`packages/setting.md`](setting.md) (where hooks are loaded from)
- Layer: `feature` (see [`reference/dependency-rules.md`](../reference/dependency-rules.md))
