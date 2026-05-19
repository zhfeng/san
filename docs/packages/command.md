---
package: github.com/genai-io/gen-code/internal/command
layer: feature
---

# command

Registry for slash commands: built-in handlers (`/help`, `/model`,
`/identity`, …), dynamic provider-supplied entries, and user-defined
markdown commands under `commands/` directories. Resolves names, fuzzy
prefixes, and plugin-scoped command paths.

## Purpose

Slash commands are the user-input side of the TUI's command palette. This
package owns the unified lookup surface: `Get("/help")`, `List()`,
fuzzy-prefix matching for the autocompleter, and the registry of custom
commands loaded from disk.

## Contract

```go
package command

// Service is the public contract for the command module.
type Service interface {
    Get(name string) (Info, bool)
    List() []Info
    ListCustom() []CustomCommand
    GetMatching(prefix string) []Info
    IsCustomCommand(cmd string) (*CustomCommand, bool)
    BuiltinNames() map[string]Info
    GetCustomCommands() []Info
}

type PluginCommandPath struct {
    Path      string
    Namespace string
    IsProject bool
}

type Options struct {
    CWD                string
    DynamicProviders   []func() []Info
    PluginCommandPaths func() []PluginCommandPath
}
```

### Known Violations

- **Rule 1 (small).** 7 methods on `Service` is at the upper edge.
  `List` / `BuiltinNames` / `GetCustomCommands` overlap conceptually;
  could consolidate. Acceptable; the methods all serve the autocompleter.
- **Rule 5.** `Default()` returns `Service`.
- **Singleton via `Default()`.**

Otherwise this is one of the cleaner package contracts — the surface is
narrow and consumer-oriented.

## Internals

- `service` (`service.go`) — concrete implementation, holds cwd, dynamic
  provider functions, and a plugin-command-path callback.
- `registry.go` — combines three sources at query time:
  - **Built-ins** registered from `builtin/` subpackage at init.
  - **Dynamic** — `DynamicProviders` callbacks returning `Info` slices
    (used by skill/agent slash-command surfaces).
  - **Custom** — markdown files under `~/.gen/commands/` and
    `<project>/.gen/commands/`, plus plugin-scoped paths returned by
    `PluginCommandPaths`.
- `Info` carries name, description, namespace, source path.

## Lifecycle

- Construction: `Initialize(Options{CWD, DynamicProviders, PluginCommandPaths})`
  at app startup, after `skill` and `subagent` are initialized so their
  dynamic providers are wired in.
- Reload on plugin reload: callers re-call `Initialize` with refreshed
  provider closures.

## Tests

```
internal/command/registry_test.go    — name lookup, fuzzy matching,
                                        custom + built-in precedence.
```

## See Also

- Code: `internal/command/`
- Related: [`packages/skill.md`](skill.md), [`packages/subagent.md`](subagent.md) (dynamic providers), [`packages/plugin.md`](plugin.md)
- Reference: [`reference/slash-commands.md`](../reference/slash-commands.md)
- Layer: `feature`
