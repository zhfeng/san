---
package: github.com/genai-io/gen-code/internal/plugin
layer: feature
---

# plugin

Plugin loader, installer, marketplace, and aggregator. A plugin is a
single package (directory) that may contribute skills, subagent
definitions, slash commands, MCP servers, hooks, and env vars — this
package discovers them, enables/disables them, and exposes their
contributions to the consuming feature packages.

## Purpose

Gen Code's "everything else" extension surface. Plugins are how a user
installs a bundle of skills + agents + commands + MCP servers + hooks in
one shot. This package handles install/uninstall, marketplace lookup,
load order, and the cross-cutting callbacks that let `skill`, `subagent`,
`command`, `mcp`, `hook`, and `setting` see what each enabled plugin
contributes without importing `plugin` directly.

## Contract

```go
package plugin

// Service is the public contract for the plugin module.
type Service interface {
    // loading
    Load(ctx context.Context, cwd string) error
    LoadClaudePlugins(ctx context.Context) error
    LoadFromPath(ctx context.Context, path string) error

    // query
    List() []*Plugin
    Get(name string) (*Plugin, bool)
    GetEnabled() []*Plugin
    Count() int
    EnabledCount() int

    // mutation
    Enable(name string, scope Scope) error
    Disable(name string, scope Scope) error

    // installer
    NewInstaller(cwd string) *Installer

    // access
    Registry() *Registry

    // plugin root management
    SetActivePluginRoot(path string)
    ClearActivePluginRoot()
    FindPluginRootForPath(path string) string

    // cross-domain (consumed by other services at init)
    AgentPaths() []PluginPath
    SkillPaths() []PluginPath
    CommandPaths() []PluginPath
    MCPServers() []PluginMCPServer
    PluginHooks() map[string][]setting.Hook
    PluginEnv() []string
}
```

### Known Violations

- **Rule 1 (small).** 21 methods — second worst after `hook`. Concerns
  span loading, querying, mutating, installing, registry-access, plugin
  root management, and six cross-domain accessor methods. Suggested
  split:
  - `PluginLoader` → `Load`, `LoadClaudePlugins`, `LoadFromPath`
  - `PluginRegistry` → `List`, `Get`, `GetEnabled`, `Count`,
    `EnabledCount`
  - `PluginStateStore` → `Enable`, `Disable`
  - `PluginInstallerFactory` → `NewInstaller`
  - `PluginContributions` → the six `AgentPaths`/`SkillPaths`/… methods
- **Rule 7 (escape hatches).** `Registry() *Registry` and
  `SetActivePluginRoot` / `ClearActivePluginRoot` /
  `FindPluginRootForPath` (package-level state) both leak. The latter
  three are package-level globals dressed up as methods; move into the
  registry value.
- **Rule 5.** `Default()` returns `Service`.
- **Cross-domain coupling.** Each `*Paths()` method exists so that
  `command`/`skill`/`subagent`/`mcp` can consume plugin contributions
  without importing `plugin`. The right pattern is reverse-injection:
  on `Initialize`, `plugin` *pushes* its contributions into each
  consumer rather than each consumer *pulling* from `plugin`. Today's
  init order (callbacks passed in `Options{PluginAgentPaths: ...}`) is
  already half of this; finish it.

## Internals

- `Registry` (`registry.go`) — `Plugin` map keyed by name, enable state
  per scope (user / project).
- `loader.go` — discovers `.gen/plugins/`, `~/.gen/plugins/`,
  `.claude/plugins/`, `~/.claude/plugins/`.
- `installer.go` — install/uninstall logic, dependency check, version
  pin (~13 KB).
- `marketplace.go` — registry-of-registries lookup (where to find
  plugin sources).
- `resolver.go` — name → install spec resolution.
- `integration.go` — the cross-domain callback wiring.

## Lifecycle

- Construction: `Initialize(ctx, Options{CWD})` is one of the first
  `Initialize` calls because every feature package's `Initialize`
  pulls `plugin.*Paths()`.
- Reload: enabling/disabling a plugin triggers a reload of the
  affected feature packages (commands/skills/subagents/MCP).

## Tests

```
internal/plugin/plugin_test.go      — large suite covering load /
                                       install / enable / contributions.
```

## See Also

- Code: `internal/plugin/`
- Consumers: [`packages/skill.md`](skill.md), [`packages/subagent.md`](subagent.md), [`packages/command.md`](command.md), [`packages/mcp.md`](mcp.md), [`packages/hook.md`](hook.md), [`packages/setting.md`](setting.md)
- Concepts: [`concepts/extension-model.md`](../concepts/extension-model.md)
- Layer: `feature`
