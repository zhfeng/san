---
package: github.com/genai-io/gen-code/internal/tool
layer: feature
---

# tool

Registry of built-in tools the agent can call, with their JSON schemas,
permission gate, side-effect plumbing, and per-call dispatch.

## Purpose

Every built-in tool (Bash, Read, Edit, Write, Grep, Glob, WebFetch, …)
registers into this package's singleton at init time. The agent loop calls
`Execute(name, params, cwd)` to dispatch; the registry resolves the tool,
runs the permission check (`internal/setting`), invokes the tool, and
returns a `toolresult.ToolResult` with stdout, error, side-effect handle,
and audit metadata.

## Contract

```go
package tool

// Service is the public contract for the tool module.
type Service interface {
    // registration
    Register(t Tool)
    RegisterAlias(alias string, t Tool)
    Get(name string) (Tool, bool)
    List() []string

    // execution
    Execute(ctx context.Context, name string, params map[string]any, cwd string) toolresult.ToolResult

    // side effects
    PopSideEffect(toolCallID string) any
}
```

`Tool` itself is defined in [`packages/core.md`](core.md). The built-in
tool implementations live under `internal/tool/{fs,web,agent,task,perm,
mode,skill,tasktools,toolresult}/`.

### Known Violations

- **Rule 1 (small).** 6 methods on `Service` spanning registration,
  execution, and a side-effect escape hatch. Acceptable; the seam is
  cohesive. Possible future split: `ToolRegistry` (Register/Get/List) vs
  `ToolDispatcher` (Execute/PopSideEffect).
- **Rule 5 (constructors return concrete types).** `Default()` returns
  `Service`; `defaultRegistry` is the only implementation. Returning
  `*Registry` would let tests substitute behavior without `SetDefault`.
- **`PopSideEffect` is a leaky abstraction.** Side effects (e.g. file
  edits queued for confirmation) flow through a parallel channel rather
  than the `ToolResult` value. Callers must remember to pop or leak data.
  Consider folding side effects into `ToolResult`.
- **Singleton via `Default()`.** Same as the rest of the codebase. Move
  to composition root.

## Internals

- `Registry` (`registry.go`) — name → `Tool` map plus alias map. The
  package-level `defaultRegistry` is the singleton; subpackages call
  `tool.Register(...)` from `init()`.
- Schemas (`schema_base.go`, `schema_agent.go`, `schema_task.go`) — JSON
  schema fragments shared by the built-in tools.
- Permission gate (`perm/`) — wraps tool execution with
  `setting.HasPermissionToUseTool`. Returns deny / ask / allow.
- Side-effect store — `Execute` may return a token; the consumer calls
  `PopSideEffect(token)` to retrieve a queued action (e.g. a pending
  file write).
- Subpackages own their tools: `fs/` for filesystem, `web/` for fetch,
  `agent/` for the Agent tool, `task/` for task tools, `skill/` for skill
  invocation, etc.

## Lifecycle

- Registration: tools register from `init()` in their subpackages.
- `Initialize(Options{})` flips the singleton to `defaultRegistry` —
  before that, `Default()` already returns `defaultRegistry`, so
  `init()`-time registrations are not lost.
- Per-call: `Execute` is goroutine-safe; the registry uses an RWMutex.

## Tests

```
internal/tool/execute_test.go         — dispatch and not-found behavior.
internal/tool/schema_agent_test.go    — schema generation for the Agent tool.
internal/tool/taskoutput_disabled_test.go — TaskOutput tool gating.
```

## See Also

- Code: `internal/tool/`
- Primitive: [`packages/core.md`](core.md) (`Tool` and `Tools` interfaces)
- Permission gate: [`packages/setting.md`](setting.md), [`concepts/permission-model.md`](../concepts/permission-model.md)
- MCP-registered tools: [`packages/mcp.md`](mcp.md)
- Layer: `feature`
