---
package: github.com/genai-io/gen-code/internal/task
layer: feature
---

# task

Background task manager. Long-running shell commands and subagent runs
are tracked here so the user can observe progress, kill them, and read
their output asynchronously.

## Purpose

When the agent invokes `Bash` with `run_in_background: true` or spawns a
subagent via `Agent`, the work is registered as a `BackgroundTask` here.
The TUI's task panel reads from this registry; the `TaskOutput` and
`TaskList` tools let the agent itself observe its background work.

## Contract

```go
package task

// Service is the public contract for the task module.
type Service interface {
    // lifecycle
    RegisterTask(t BackgroundTask)
    CreateBashTask(cmd *exec.Cmd, command, description string,
                   ctx context.Context, cancel context.CancelFunc) *BashTask
    Get(id string) (BackgroundTask, bool)
    List() []BackgroundTask
    ListRunning() []BackgroundTask
    Kill(id string) error
    Remove(id string)

    // output
    SetOutputDir(dir string) error
}
```

### Known Violations

- **Rule 1 (small).** 8 methods. Borderline. `Get` / `List` /
  `ListRunning` could collapse to `Get(id)` + `List(filter)`.
- **`CreateBashTask` knows too much about Bash.** A generic
  `RegisterRunningProcess(p Process)` factory would be more reusable.
  The Bash-specific signature is convenient for the one caller but
  prevents future task types (HTTP, async hook) from using the same
  factory.
- **Rule 5.** `Default()` returns `Service`.
- **Singleton via `Default()`.**

## Internals

- `Manager` (`manager.go`) — concrete implementation. Tracks active and
  completed tasks under a mutex.
- `BackgroundTask` (`types.go`) — interface implemented by `BashTask` and
  `AgentTask`.
- `BashTask` (`bash_task.go`) — wraps `*exec.Cmd`, streams stdout/stderr
  to disk, exposes `Tail`/`Read`.
- `AgentTask` (`agent_task.go`) — wraps a subagent invocation.
- `output_store.go` — filesystem-backed per-task output files under
  `<output-dir>/<task-id>.log`.
- `tracker/` (subpackage) — task state machine the session recorder
  serializes into transcripts; surfaced by the `TaskCreate` /
  `TaskUpdate` tools.

## Lifecycle

- Construction: `Initialize(Options{OutputDir})` at app start.
- Per-task: `CreateBashTask(...)` returns a `*BashTask` already
  registered. `Kill(id)` cancels the context; `Remove(id)` evicts the
  record after completion.
- Concurrency: registry is mutex-protected; per-task streams use their
  own buffers.

## Tests

```
internal/task/manager_test.go        — register / get / list / kill.
internal/task/bash_task_test.go      — output streaming and cancel.
internal/task/hooks_test.go          — lifecycle hook emission.
```

## See Also

- Code: `internal/task/`, `internal/task/tracker/`
- Spawning surface: [`packages/tool.md`](tool.md) (Bash/Agent tools), [`packages/subagent.md`](subagent.md)
- Layer: `feature`
