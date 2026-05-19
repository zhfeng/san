---
package: github.com/genai-io/gen-code/internal/mcp
layer: feature
---

# mcp

Model Context Protocol (MCP) client — connects to external MCP servers,
lists their tools, and exposes a `Caller` that the tool registry uses to
invoke them. Transport implementations live in `internal/mcp/transport/`.

## Purpose

MCP is how Gen Code talks to **external** tool providers (file system
servers, GitHub servers, custom user servers, …). Each configured MCP
server is a separate subprocess (`stdio`) or HTTP endpoint that publishes
a list of tools; this package maintains the connection, surfaces those
tools as `core.ToolSchema` to the agent loop, and dispatches calls
through `Caller`.

Config lives at `<project>/.gen/mcp.json` (Gen Code) or
`<project>/.claude/mcp_servers.json` (Claude-compat).

## Contract

```go
package mcp

// Service is the public contract for the mcp module.
type Service interface {
    // connection
    ListServers() []Server
    Connect(ctx context.Context, name string) error
    ConnectAll(ctx context.Context) []error
    Disconnect(name string) error
    Reconnect(ctx context.Context, name string) error

    // tools
    ListTools() []core.ToolSchema
    NewCaller() *Caller

    // config
    EditConfig(name string) (*EditInfo, error)
    SaveConfig(info *EditInfo) error

    // registry access (backward compat)
    Registry() *Registry
}
```

### Known Violations

- **Rule 1 (small).** 10 methods. Suggested split: `MCPConnections`,
  `MCPTools`, `MCPConfigEditor`.
- **Rule 7 (escape hatch).** `Registry() *Registry` is even labeled
  "backward compat" — remove it.
- **Rule 5.** `Default()` returns `Service`.
- **Singleton via `Default()` + `DefaultIfInit()`.**

## Internals

- `Registry` (`registry.go`) — server name → `Client` map; merges
  user/project/plugin config.
- `Client` (`client.go`) — one per server; owns the transport
  connection, request/response framing, tool list cache.
- `Caller` (`caller.go`) — invokes a tool on the right server based on
  tool name (MCP tool names are namespaced).
- `transport/` — stdio and HTTP transports implementing the MCP
  protocol.
- `config.go` + `edit.go` — load/edit the JSON config files.
- `core_adapter.go` — wraps MCP `tool/call` into `core.Tool` interface.
- `hooks.go` — fires lifecycle hooks (server connect/disconnect).

## Lifecycle

- Construction: `Initialize(Options{CWD, PluginServers})` reads config,
  merges plugin-supplied servers, but does **not** connect.
- Per-connect: `ConnectAll(ctx)` opens transports in parallel; failures
  surface in the returned `[]error`.
- Reload: `/mcp` slash command edits config and reconnects.
- Concurrency: registry is mutex-protected; per-client message routing
  is goroutine-safe.

## Tests

```
internal/mcp/config_test.go               — config parse/merge.
internal/mcp/registry_connecting_test.go  — connect/disconnect cycles.
internal/mcp/hooks_test.go                — lifecycle hook emission.
```

## See Also

- Code: `internal/mcp/`, `internal/mcp/transport/`
- Tools: [`packages/tool.md`](tool.md) (MCP tools register here via `core_adapter`)
- Plugins: [`packages/plugin.md`](plugin.md) (plugins can ship MCP servers)
- Reference: [`reference/configuration.md`](../reference/configuration.md)
- Concepts: [`concepts/extension-model.md`](../concepts/extension-model.md)
- Layer: `feature`
