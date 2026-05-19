---
package: github.com/genai-io/gen-code/internal/subagent
layer: feature
---

# subagent

Registry of custom agent **types** (markdown-defined personas with their
own system prompt, tool subset, and permission mode) plus the **Executor**
that spawns them as background `core.Agent` instances from within the
main agent's tool loop.

## Purpose

Where [`packages/agent.md`](agent.md) owns the *foreground* agent session,
this package owns the *background* agents the foreground spawns via the
Agent tool. Each background agent runs in its own goroutine with its own
inbox/outbox, isolated work tree (optional), and isolated tool/permission
set; its final result flows back into the foreground conversation as a
tool result.

## Contract

```go
package subagent

// Service is the public contract for the subagent module.
type Service interface {
    // query
    ListConfigs() []*AgentConfig
    Get(name string) (*AgentConfig, bool)
    IsEnabled(name string) bool
    GetDisabledAt(userLevel bool) map[string]bool

    // mutation
    SetEnabled(name string, enabled bool, userLevel bool) error
    Register(config *AgentConfig)

    // factory
    NewExecutor(provider llm.Provider, cwd string, parentModelID string, hookEngine *hook.Engine) *Executor

    // system prompt
    PromptSection() string

    // concrete access
    Registry() *Registry
}
```

### Known Violations

- **Rule 1 (small).** 9 methods. Suggested split: `SubagentQuery`,
  `SubagentStateStore`, `SubagentExecutorFactory`, `SubagentPrompt`.
- **Rule 7 (no escape hatch).** `Registry() *Registry` defeats the
  interface. Drop it.
- **Rule 5.** `Default()` returns `Service`.
- **`NewExecutor` takes a `*hook.Engine` directly.** Per
  [`packages/hook.md`](hook.md) the hook package shouldn't be reached
  through its concrete `*Engine` — accept `hook.Service` or a narrow
  `HookExecutor` interface instead.

## Internals

- `Registry` (`registry.go`) — `AgentConfig` map keyed by name, plus
  enable state stores (user + project).
- `Executor` (`executor.go`) — spawns a `core.Agent` for one subagent
  invocation, manages its lifecycle, drains its outbox, and returns the
  aggregated result.
- `executor_prompt.go` / `executor_run.go` / `executor_session.go` —
  split executor concerns (prompt assembly, run loop, session attribution).
- `loader.go` — reads markdown agent definitions from
  `~/.gen/agents/`, `<project>/.gen/agents/`, plus plugin paths.
- `match.go` — name matching for the Agent tool's `agent_type` parameter.
- `progress_tools.go` — pseudo-tools the subagent emits to surface
  progress to the parent agent.

## Lifecycle

- Construction: `Initialize(Options{CWD, PluginAgentPaths})` loads
  definitions, initializes state stores.
- Per-invocation: `NewExecutor(provider, cwd, model, hookEngine)` →
  `Executor.Run(ctx, params)` spawns a `core.Agent`, blocks until end of
  turn, returns aggregated `Result`.
- Concurrency: multiple executors may run in parallel; the registry is
  RWMutex-protected.

## Tests

```
internal/subagent/executor_test.go      — end-to-end run scenarios.
internal/subagent/lazy_loading_test.go  — config files read on demand.
internal/subagent/scenarios_test.go     — common invocation shapes.
```

## See Also

- Code: `internal/subagent/`
- Parent agent: [`packages/agent.md`](agent.md)
- Spawning tool: [`packages/tool.md`](tool.md) (the Agent tool)
- Worktree isolation: [`packages/worktree.md`](worktree.md)
- Layer: `feature`
