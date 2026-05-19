---
package: github.com/genai-io/gen-code/internal/setting
layer: feature
---

# setting

Settings loader, merger, and the central permission decision gate.
Reads `~/.gen/settings.json` and `<project>/.gen/settings.json`, merges
project-over-user with documented precedence, and decides allow / deny /
ask for every tool call.

## Purpose

Two concerns live here:

1. **Configuration**: load and merge two-tier settings (user + project),
   plus hooks, disabled tools, search provider, permission rules, env
   vars, work directory, and Claude Code-compatible `.claude/` shims.
2. **Permission decisions**: `HasPermissionToUseTool` is the
   authoritative gate every tool call passes through. Decision sources
   include explicit rules, suggestions from hooks, session-scoped
   permissions, and bypass-mode policy.

## Contract

```go
package setting

// Service is the public contract for the setting module.
type Service interface {
    // Snapshot returns the current merged settings.
    Snapshot() *Settings
    AllowBypass() bool
    IsGitRepo(cwd string) bool
    Reload(cwd string) error
    DisabledTools() map[string]bool
    SearchProvider() string
    SetSearchProvider(provider string)
    Hooks() map[string][]Hook

    // permission gate
    CheckPermission(toolName string, args map[string]any, session *SessionPermissions) PermissionBehavior
    HasPermissionToUseTool(toolName string, args map[string]any, session *SessionPermissions) PermissionDecision
    ResolveHookAllow(toolName string, args map[string]any, session *SessionPermissions) bool

    // per-level disabled tools
    GetDisabledToolsAt(userLevel bool) map[string]bool
    UpdateDisabledToolsAt(disabledTools map[string]bool, userLevel bool) error
}
```

### Known Violations

- **Rule 1 (small) — 14 methods.** Two concerns (config + permission)
  fused. Suggested split into two packages or two interfaces:
  - `Settings` → snapshot/reload/disabled-tools/search-provider/hooks
  - `PermissionGate` → CheckPermission / HasPermissionToUseTool /
    ResolveHookAllow / AllowBypass
  This split matches the actual code surface and would let
  `internal/tool` depend on `PermissionGate` alone.
- **Rule 7 (no escape hatch).** `Snapshot() *Settings` returns a clone of
  the concrete struct, which exposes every field. Most callers only need
  a few — narrow with focused accessors.
- **Rule 5.** `Default()` returns `Service`.
- **Singleton via `Default()` + `DefaultIfInit()`.**
- **Permission API surface is wide.** `CheckPermission`,
  `HasPermissionToUseTool`, and `ResolveHookAllow` overlap in concern.
  Consolidating into a single `Decide(req) PermissionDecision` would
  simplify both callers and tests.

## Internals

- `Settings` (`settings.go`) — value type holding all merged config.
- `loader.go` + `merger.go` — read the two tiers and combine them with
  documented precedence (project overrides user, except in a few flagged
  fields).
- `permission.go` — the rule engine. Big file (19k); deserves to move out
  to a `service/permission/` package per the split above.
- `bash_ast.go` — bash command parsing for the granular Bash permission
  rules (read-only matchers like `git status` allowed but `git push`
  asked).
- `workdir.go` — cwd resolution and git-root detection.
- `security.go` — env var sanitization for hook/MCP subprocess execution.

## Lifecycle

- Construction: `Initialize(Options{CWD})` runs once at startup and after
  cwd changes.
- Reload: `Reload(cwd)` rebuilds settings under lock; the singleton swaps
  atomically.
- Per-call: permission checks are mutex-protected reads against the
  current snapshot.

## Tests

```
internal/setting/permission_test.go      — large table of permission
                                            scenarios.
internal/setting/config_extra_test.go    — config merge semantics.
internal/setting/bash_ast_test.go        — bash command parsing for
                                            permission patterns.
internal/setting/workdir_test.go         — cwd resolution.
```

## See Also

- Code: `internal/setting/`
- Reference: [`reference/configuration.md`](../reference/configuration.md)
- Concepts: [`concepts/permission-model.md`](../concepts/permission-model.md)
- Permission consumers: [`packages/tool.md`](tool.md), [`packages/hook.md`](hook.md)
- Layer: `feature`
