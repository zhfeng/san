---
package: github.com/genai-io/gen-code/internal/llm
layer: feature
---

# llm

Provider registry, model store, and client factory for every LLM backend
(Anthropic, OpenAI, Google, Moonshot, Alibaba, MiniMax, Z.ai/GLM, DeepSeek,
plus the generic openai-compat shim). Provider implementations live in
`internal/llm/<name>/` subpackages.

## Purpose

The agent loop talks to LLMs through `core.LLM` (see
[`packages/core.md`](core.md)). This package owns the *machinery around*
that contract — discovering providers, persisting the user's chosen
provider/model, switching between them at runtime, and tracking cost and
streaming details for each call.

## Contract

```go
package llm

// Service is the public contract for the llm module.
type Service interface {
    // connection
    Provider() Provider              // current active provider
    SetProvider(p Provider)          // switch provider
    ModelID() string                 // current model identifier
    CurrentModel() *CurrentModelInfo // full model metadata
    SetCurrentModel(info *CurrentModelInfo)

    // factory
    NewClient(model string, maxTokens int) *Client

    // store
    Store() *Store // underlying provider persistence store

    // registry
    ListProviders() map[Name][]Info // all registered providers with status
}
```

`Client` implements `core.LLM`. `Provider` is the per-backend handle
(API key, auth method, list-models endpoint).

### Known Violations

- **Rule 1 (small).** 8 methods. Mixes connection state, factory, store
  access, and registry listing. Suggested split: `LLMConnection`
  (Provider/SetProvider/Model methods), `LLMClientFactory` (NewClient),
  `LLMProviderRegistry` (ListProviders).
- **Rule 7 (no escape hatch).** `Store()` exposes the concrete `*Store`.
  Callers should depend on a narrower read-only view.
- **Rule 5.** `Default()` returns `Service` not `*service`.
- **Singleton via `Default()`.**

## Internals

- `service` (`service.go`) — singleton implementation wrapping a `Setup`
  struct (mutex + current Provider/Model + Store).
- `Provider` registry (`registry.go`) — discovery, dynamic model list
  fetching (per memory: prefer `/models` over hardcoded catalogs).
- `Client` (consolidated `Infer` path) — adapts a `Provider` + model into
  `core.LLM`, tracks per-call token counts, streams `core.Chunk`, applies
  retry/cost logic via `logging.go` and `money.go`.
- `Store` (`store.go`) — persists user's provider connections under
  `~/.gen/providers.json`; tracks current model.
- `stream/` — provider-side helpers for SSE parsing.
- Provider subpackages: `anthropic/`, `openai/`, `google/`, `moonshot/`,
  `alibaba/`, `bigmodel/`, `minmax/`, `deepseek/`, `openaicompat/`.

## Lifecycle

- Construction: `Initialize(Options{})` loads `~/.gen/providers.json`,
  picks the last-used provider (or the first connectable one), and stores
  it.
- Switching: `/model` slash command calls `SetCurrentModel` + reload.
- Per-call: `NewClient(model, maxTokens)` produces a `*Client` for one
  inference; the client wraps `Provider.Infer`.

## Tests

```
internal/llm/llm_test.go        — Client.Infer plumbing.
internal/llm/store_test.go      — provider config persistence.
internal/llm/fake_llm.go        — test double consumed by other packages.
```

## See Also

- Code: `internal/llm/`
- Primitive: [`packages/core.md`](core.md) (`LLM` interface)
- Cost tracking surfaced via [`packages/session.md`](session.md) recorder.
- Layer: `feature`
