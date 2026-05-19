---
package: github.com/genai-io/gen-code/internal/skill
layer: feature
---

# skill

Loads markdown-defined skills from user / project / plugin scopes, tracks
their enable state, and renders the section injected into the system prompt
for active skills.

## Purpose

A skill is a markdown file (with YAML frontmatter) that the model can be
made aware of (`active`), made invocable via slash command (`enabled`), or
hidden (`disabled`). This package:

1. Discovers skills across six scopes
   (`~/.claude/skills/`, `~/.gen/plugins/*/skills/`, `~/.gen/skills/`,
   `.claude/skills/`, `.gen/plugins/*/skills/`, `.gen/skills/`) with
   project overriding user overriding Claude-compat.
2. Persists per-skill state in user / project state stores.
3. Renders the active-skills section consumed by `core.System`.

For the broader extension model see
[`concepts/extension-model.md`](../concepts/extension-model.md). A
how-to-author-a-skill guide is tracked in `notes/tech-debt.md`.

## Contract

The seam consumed by `internal/app`, `internal/command`, and
`internal/core/system`:

```go
package skill

// Service is the public contract for the skill module.
type Service interface {
    // query
    List() []*Skill                       // all loaded skills
    Get(name string) (*Skill, bool)       // lookup by name
    IsEnabled(name string) bool           // check if enabled
    FindByPartialName(name string) *Skill // partial/suffix match
    GetEnabled() []*Skill                 // all enabled or active skills
    GetActive() []*Skill                  // all active skills (model-aware)
    Count() int                           // total number of loaded skills

    // mutation
    SetEnabled(name string, enabled bool, userLevel bool) error
    GetDisabledAt(userLevel bool) map[string]bool

    // system prompt
    PromptSection() string                       // rendered section for system prompt
    GetSkillInvocationPrompt(name string) string // full skill content for injection

    // plugin
    AddPluginSkills(paths []struct {
        Path      string
        Namespace string
        IsProject bool
    })

    // concrete access
    Registry() *Registry
}

// Skill, SkillState, SkillScope — see types.go for value types.
```

### Known Violations

Tracked for PR-3. The contract above is verbatim from today's code.

- **Rule 1 (small).** **12 methods** across four concerns. Suggested split:
  - `SkillQuery` → `List`, `Get`, `Count` (or pick three of the seven
    query methods; consolidate `IsEnabled` / `GetEnabled` / `GetActive`
    behind a `Filter(state SkillState)` if downstream usage permits)
  - `SkillStateStore` → `SetEnabled`, `GetDisabledAt`
  - `SkillPrompt` → `PromptSection`, `GetSkillInvocationPrompt`
  - `SkillSourceRegistrar` → `AddPluginSkills`
  - Remove `Registry()` (see Rule 7)
- **Rule 7 (no escape hatch).** `Registry() *Registry` lets every caller
  reach the concrete type. Drop it; if a caller needs methods that aren't
  on `Service`, add them to the appropriate split interface or have the
  caller depend on `*Registry` directly.
- **Rule 4 (named types over anonymous structs).** `AddPluginSkills` takes
  `[]struct { Path string; Namespace string; IsProject bool }`. Define
  `type PluginSkillSource struct { ... }` and take `[]PluginSkillSource`.
  Anonymous structs in exported signatures cannot be referenced by name
  and break go-doc readability.
- **Rule 5 (constructors return concrete types).** `Default()` returns
  `Service` (interface). Should return `*Registry` if callers are
  collaborators in the same module.
- **Singleton via `Default()` and `DefaultIfInit()`.** Same issue as
  `hook` and `agent`: two-flavor accessors paper over racy init. Move
  construction into the app composition root.

## Internals

- `Registry` (`registry.go`) is the only implementation, holding:
  - `skills []*Skill` — loaded by `loader`
  - `userStore`, `projectStore` — JSON-backed persistence of per-skill
    `SkillState`
  - `cwd` — for project-scope resolution
- `loader.go` walks the six scopes in priority order, parsing
  `SKILL.md` frontmatter and bundled resource directories
  (`scripts/` / `references/` / `assets/`).
- State (`disable` → `enable` → `active` → `disable`) cycles via
  `SkillState.NextState()`; the TUI's `/skill` flow uses this.
- Active skills render into the system prompt through `PromptSection()`;
  enable-only skills surface as slash commands but stay out of the system
  prompt.

## Lifecycle

- Construction: `Initialize(Options{CWD})` at app startup, before
  `internal/command` builds its slash-command list. Singleton thereafter.
- Mutation: `SetEnabled` writes through to user or project store
  immediately; in-memory `skills` slice is updated in place.
- Plugin sources are added post-init by `internal/plugin` via
  `AddPluginSkills`.
- Concurrency: registry mutations are mutex-guarded; reads are
  RWMutex-locked.

## Tests

```
internal/skill/skill_test.go            — loader, state cycling,
                                            scope priority, prompt rendering.
internal/skill/lazy_loading_test.go     — verifies content stays on disk
                                            until GetInstructions().
```

## See Also

- Code: `internal/skill/`
- Concepts: [`concepts/extension-model.md`](../concepts/extension-model.md), [`concepts/harness-channels.md`](../concepts/harness-channels.md)
- Related: [`packages/command.md`](command.md) (slash-command surface), [`packages/plugin.md`](plugin.md) (plugin-scoped skills), [`packages/core.md`](core.md) (system prompt assembly)
- Layer: `feature` (see [`reference/dependency-rules.md`](../reference/dependency-rules.md))
