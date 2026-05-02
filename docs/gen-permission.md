# gen Code Permissions

A unified design covering both the main loop and subagents. Same vocabulary, same rule syntax, same evaluation pipeline. Differences between the two are confined to one bit — whether the runtime is allowed to prompt the user — and one knob — which mode is in effect.

## Design Principles

1. **One mode vocabulary.** Main loop and subagents share the same mode names. `default`, `acceptEdits`, `dontAsk`, `bypassPermissions`, and `auto` apply to both. `explore` is the only subagent-only label, because an interactive user has no reason to "lock themselves out" of mutations. No `edit`/`acceptEdits` synonyms.
2. **One rule syntax.** Every allow / ask / deny rule is a `Tool(pattern)` string. Glob wildcards (`*`, `**`, `?`). No regex.
3. **One evaluation pipeline.** A single `HasPermissionToUseTool(tool, args, mode, rules) → Allow | Ask | Deny` function. The main loop turns `Ask` into a TUI dialog; subagents turn `Ask` into `Deny`. That is the only behavioral difference.
4. **Tools beat mode.** Per-target deny / allow rules always win over the mode default. Mode decides what to do when no rule matches.

---

## Modes

**Main-loop modes** (selectable via `Shift+Tab`, `--permission-mode`, or `defaultMode` in settings):

| Mode | Reads | Edits / Writes | Bash / Exec / Agent | Use case | Status |
|------|-------|----------------|---------------------|----------|--------|
| `default` | Allow | Ask | Ask | Sensitive work; review every mutation | implemented |
| `acceptEdits` | Allow | Allow | Ask | Iterating on code | implemented |
| `dontAsk` | Allow | Silent Deny | Silent Deny | Non-interactive runs (`gen -p ...`, scripts, CI); only pre-allowed tools execute | implemented |
| `bypassPermissions` | Allow | Allow | Allow | Containers / CI; trust everything except bypass-immune checks | implemented |
| `auto` | Allow | Allow | Allow w/ safety classifier | Long-running unattended sessions that should make progress without asking, with a learned classifier escalating risky bash to Deny | **TODO** (shared with subagents) — currently aliased to `acceptEdits` until the classifier ships |

**Subagent-only modes** (declared in agent frontmatter via `mode:`):

| Mode | Reads | Edits / Writes | Bash / Exec / Agent | Use case | Status |
|------|-------|----------------|---------------------|----------|--------|
| `explore` | Allow | **Deny** | **Deny** | Read-only investigation pass; the agent is told it cannot mutate the workspace | implemented |

`explore` does not appear as a main-loop mode because an interactive user does not need it: in the main loop a user wanting read-only just declines the prompts (or runs in `dontAsk`). It exists as a subagent label so the agent's system prompt can communicate "you may only read" up front, instead of having the agent attempt mutations that would be silently denied.

The previous `edit` mode is now `acceptEdits`.

### Subagent collapsing rule

The runtime cannot show prompts inside a subagent. Any `Ask` decision becomes `Deny` automatically. So:

- A subagent in `default` behaves like a read-only assistant (mutations Ask → Deny).
- A subagent in `acceptEdits` can read and edit but cannot run Bash / spawn agents.
- A subagent in `explore` is the same effective behavior as `default`-on-subagent, but the agent is *told* it is read-only via the system prompt, so it does not waste turns attempting mutations.
- A subagent in `bypassPermissions` runs unattended.
- `auto` is the future "long-running, less prompting" mode — TODO.

---

## Rule Syntax

A rule is a string: `Tool(pattern)`. Used identically in:

- `permissions.allow` / `permissions.ask` / `permissions.deny` in `settings.json`
- `allow_tools` / `deny_tools` in agent definitions (AGENT.md frontmatter)

### Pattern Forms

**Bash**
```
Bash(git status)        # exact
Bash(git:*)             # any git subcommand
Bash(npm run *)         # npm run + any args
Bash(* --version)       # any command ending in --version
```

`Bash(...)` is the only pattern that splits compound commands. The command is parsed via shell AST and split on `&&`, `||`, `;`, `|`, `&`, newlines. **Each subcommand must independently match an allow rule** for the whole command to be allowed. Bypass-immune destructive checks (`rm -rf /`, `git push --force`, etc.) apply to every subcommand regardless.

Process wrappers (`timeout`, `time`, `nice`, `nohup`, `stdbuf`) are stripped before matching. `watch`, `find -exec`, `xargs`, `ionice` are not stripped.

**File paths** (`Read`, `Edit`, `Write`)
```
Read(./src/**)              # cwd-relative, recursive
Edit(/docs/**)              # project-root-relative, recursive
Read(~/.ssh/id_rsa)         # home-relative
Read(//etc/passwd)          # filesystem-absolute (note double slash)
```

`*` matches one path segment. `**` matches any number of segments.

**Other tools**
```
WebFetch(domain:github.com)
Skill(git:*)
Agent(Explore)              # restrict which subagent types may be spawned
MCP(mcp__puppeteer__*)
```

Tools without a pattern (`Bash`, `Read`, etc. with no parentheses) match any invocation.

---

## Evaluation Pipeline

Run for every tool call. First match wins within each step. The pipeline is identical in the main loop and in subagents — only the headless coercion at step 8 differs.

```
1. deny rules   (settings.deny + agent.deny_tools)        → Deny
2. bypass-immune checks                                    → Ask
   - sensitive paths (.git/, .ssh/, .env, shell rc files, …)
   - destructive bash (rm -rf, git push --force, …)
   - bash with injection / control-char obfuscation
3. mode == bypassPermissions                               → Allow
4. session runtime permissions                             → Allow
   (allowAllEdits, allowAllBash, /allow-pattern, …)
5. ask rules    (settings.ask)                             → Ask
6. allow rules  (settings.allow + agent.allow_tools)       → Allow
7. mode default
   - default       : safe tools Allow, else Ask
   - acceptEdits   : safe + edit tools Allow, else Ask
   - explore       : safe tools Allow, else Deny
   - dontAsk       : safe tools Allow, else Deny (silent)
   - auto          : safe + edit + bash Allow (subject to TODO above)
   - bypassPermissions handled in step 3
8. headless coercion
   - if subagent OR headless runtime: Ask → Deny
```

### Priority (the rule the user has to remember)

```
deny_tools  >  allow_tools  >  mode
```

Both `settings.{allow,deny}` and `agent.{allow_tools,deny_tools}` feed the same pipeline, so this priority is automatic — there is no per-source override. If a subagent's `allow_tools` whitelists `Bash(git diff*)` and the agent is in `explore` mode, the explicit allow rule wins over the mode-default Deny. If the same agent puts `Bash(git diff*)` in `deny_tools`, the deny wins.

### `Allow` / `Deny` are sticky

`Allow` returned at any step ends evaluation. `Deny` returned by step 1 cannot be overridden anywhere downstream. `Ask` returned by step 2 cannot be downgraded to `Allow` by an allow rule — sensitive paths and destructive commands always prompt the user, even when a rule whitelists them.

### Safe Tools

`Read`, `Glob`, `Grep`, `WebFetch`, `WebSearch`, `LSP`, `TaskCreate`, `TaskGet`, `TaskList`, `TaskUpdate`, `AskUserQuestion`, `CronList`.

These are allowed by the mode default in every mode. They do **not** bypass deny / ask rules. For example, `Read(./.env)` in `permissions.deny` still blocks `Read`, and `Read(./secrets/**)` in `permissions.ask` still prompts in the main loop.

### Edit tools

`Edit`, `Write`, `NotebookEdit`. Treated as a single class for `acceptEdits` / `auto`.

---

## Settings File Locations

Highest to lowest precedence. Permission arrays merge across scopes; **deny at any level cannot be overridden by allow at a lower level**.

| Scope | Location | Shared |
|-------|----------|--------|
| Managed | OS-specific (plist / registry / `/etc/gen/managed-settings.json`) | Enterprise-deployed, immutable |
| Local project | `.gen/settings.local.json` | gitignored |
| Shared project | `.gen/settings.json` | committed |
| User | `~/.gen/settings.json` | personal, all projects |

### Example

```json
{
  "permissions": {
    "defaultMode": "acceptEdits",
    "allow": [
      "Bash(npm run *)",
      "Bash(git:*)",
      "Read(./src/**)",
      "WebFetch(domain:github.com)"
    ],
    "ask": [
      "Bash(git push *)"
    ],
    "deny": [
      "Bash(rm -rf *)",
      "Read(./.env)",
      "Read(./.env.*)",
      "Read(./secrets/**)"
    ],
    "additionalDirectories": ["../shared-docs/"]
  }
}
```

---

## Subagent Permissions

Subagents do **not** inherit the main session's allow / ask / deny rules. Each subagent declares its own policy in its frontmatter:

```yaml
---
name: code-reviewer
mode: explore
allow_tools:
  - Read
  - Glob
  - Grep
  - Bash(git diff*)
  - Bash(git log:*)
  - WebFetch
deny_tools:
  - Bash(rm:*)
---
```

Field semantics:

| Field | Maps to (in pipeline) | Behavior |
|-------|----------------------|----------|
| `mode` | step 7 | Default policy when no rule matches. One of `default` / `acceptEdits` / `explore` / `dontAsk` / `bypassPermissions` / `auto`. Defaults to `default`. |
| `allow_tools` | step 6 | Allow rules. Same `Tool(pattern)` syntax as `settings.permissions.allow`. When non-empty, tools not in the list are also removed from the LLM-visible schema set (whitelist filter — see below). |
| `deny_tools` | step 1 | Deny rules. Same syntax as `settings.permissions.deny`. |

### `allow_tools` is also a schema filter

When `allow_tools` is non-empty, tools not in the list are removed from the LLM-visible schema set entirely. This is a UX convenience — the LLM never sees the tool, so it never tries to call it and never wastes turns on rejected calls. The permission gate is still authoritative; schema filtering is just a hint to the model.

A bare tool name (`Read`) makes that tool visible and unconstrained. A pattern (`Bash(git diff*)`) makes the tool visible but constrains the parameter at the gate.

### `auto` mode (TODO)

`auto` is reserved for unattended sessions that need to make progress without prompting — both long-running subagents and headless main-loop runs. Future direction: a learned classifier that auto-approves benign bash and escalates suspicious commands to `Deny`. Until the classifier ships, `auto` is aliased to `acceptEdits` everywhere it's selected.

---

## Ask Approval (main loop only)

When the pipeline returns `Ask`, the TUI shows a 4-option dialog:

| Option | Behavior | Persistence |
|--------|----------|-------------|
| Yes | Allow this single call | None |
| Yes, allow all this session | Allow this tool class for the rest of the session | Until session ends |
| Always allow | Persist as allow rule in settings.json | Across sessions |
| No | Deny this single call | None |

"Always allow" on a Bash compound command writes one rule per subcommand (up to 5).

---

## additionalDirectories

```json
"additionalDirectories": ["../shared-docs/"]
```

Extends file-access scope beyond cwd:

- Read tools always allowed for files inside.
- Write tools follow the active mode (no special exception).
- `.gen/` configuration in additional directories is **not** loaded — only `.gen/skills/`. This is a security boundary: a checked-out repo cannot inject hooks or settings via being added as an additional directory.

Set via `--add-dir <path>` on the CLI, in-session via `/add-dir`, or persisted in `additionalDirectories`.

---

## Bypass-Immune Checks

These prompt the user even when an allow rule or `acceptEdits`/`bypassPermissions` would allow them. They cannot be silenced.

**Sensitive paths** (Edit / Write):
- Directories: `.git/`, `.gen/`, `.claude/`, `.vscode/`, `.idea/`, `.ssh/`, `.aws/`, `.gnupg/`, `.kube/`
- Files: `.bashrc`, `.zshrc`, `.profile`, `.gitconfig`, `.npmrc`, `.netrc`, `.docker/config.json`, …

**Destructive bash**:
- `rm -rf`, `rm -fr`, `rm -r`
- `git reset --hard`, `git clean -fd`, `git push --force`, `git checkout --`, `git branch -D`
- `chmod 777`, `:(){ :|:& };:` (fork bomb), `> /dev/sd*`, `dd if=`, `mkfs`, `fdisk`

**Suspicious bash**:
- Nested command substitution
- Backslash-obfuscated flags
- Control characters / zero-width unicode in commands
- IFS injection
- Zsh privileged builtins (`zmodload`, `zsocket`, `zf_rm`, …)
- `/proc/.../environ` access
- Output redirection to `/etc/`, `/dev/sd*`, `~/.ssh/`, shell rc files

In subagents, `Ask` → `Deny`, so these effectively block the call.

---

## Configuring Permissions

1. **Edit settings files directly** (locations above).
2. **`/permissions` in the TUI** — interactive rule management.
3. **CLI flags** (session-only):
   ```bash
   gen --permission-mode acceptEdits
   gen --add-dir /tmp/data
   gen --allowedTools 'Bash(npm test)'
   gen --disallowedTools 'Bash(rm:*)'
   ```

Managed (enterprise) settings can lock the config:
- `disableBypassPermissionsMode: true` — users cannot enter `bypassPermissions`
- `allowManagedPermissionRulesOnly: true` — users cannot add their own allow rules
- Register `PreToolUse` hooks for custom audit logic

---

## Pitfalls

- **Deny is absolute.** A deny rule at any scope blocks the call regardless of allow rules at any scope.
- **`*` does not match empty.** `Bash(git *)` matches `git status` but not bare `git`. Use `Bash(git*)` (no space) if you need both.
- **`*` vs `**` in paths.** `Read(./src/*)` matches one segment; `Read(./src/**)` matches all subdirectories.
- **Spacing in Bash patterns.** `Bash(ls *)` matches `ls -la` but not `lsof`. `Bash(ls*)` matches both.
- **Compound Bash + allow rules.** Every subcommand must independently be allowed. A single allow rule covering one subcommand is not enough to allow the whole compound.
- **MCP tool naming.** MCP tools use `mcp__<server>__<tool>`. Deny `MCP(mcp__puppeteer__*)` blocks the entire server.
- **Symlinks.** Allow rules check both link and target; deny rules block if either matches.

---

## Implementation

The same 8-step pipeline (above) runs in main loop and subagents. Code map:

```
internal/tool/perm/
├── decision.go      Decision enum, Checker interface, mode-aligned built-in
│                    checkers (Default / AcceptEdits / ReadOnly / PermitAll /
│                    DenyAll), IsSafeTool, IsReadOnlyTool, IsEditTool,
│                    PermissionFunc, AsPermissionFunc
├── types.go         PermissionRequest, DiffMetadata, BashMetadata, ...
└── diff.go          GenerateDiff, GeneratePreview

internal/tool/permission.go   WithPermission decorator (wraps core.Tools with
                              a PermissionFunc; safe tools bypass the check)

internal/setting/permission.go
├── HasPermissionToUseTool    main-loop gate — runs all 8 steps
├── MatchAllowList            per-subcommand allow check for Bash; shared with
│                             subagent
├── MatchesToolPattern        any-subcommand match for deny / ask
├── BypassImmuneReason        sensitive paths + destructive bash; shared
└── checkHardBlocks           internal helper composing deny + bypass-immune
                              + working-dir + ask

internal/subagent/match.go
├── ToolList.Matches          deny / ask (any-subcommand)
└── ToolList.Allows           allow (every-subcommand, delegates to MatchAllowList)

internal/subagent/executor.go
├── modeChecker               PermissionMode → perm.Checker
└── subagentPermissionFunc    runs steps 1-4 + Ask→Deny coercion; the result
                              becomes the PermissionFunc passed into
                              tool.WithPermission for the agent.
```

### Wiring

Main loop:

```
Settings + SessionPermissions
    ▼
settings.HasPermissionToUseTool         8-step pipeline
    ▼
agent.PermissionBridge.PermissionFunc    Permit/Reject → return; Prompt →
    ▼                                    cross goroutine to TUI dialog
tool.WithPermission(tools, fn)
    ▼
core.Agent (Tools)
```

Subagent:

```
AgentConfig (mode, allow_tools, deny_tools)
    ▼
subagent.subagentPermissionFunc          steps 1-4 + Ask→Deny
    ▼
tool.WithPermission(tools, fn)
    ▼
core.Agent (Tools)
```

Both paths produce a `tool.PermissionFunc` that the same `WithPermission`
decorator wraps around `core.Tools`. The agent itself has no knowledge of
permission — the wrap is transparent.

### Hook integration

Hooks sit around the gate at the **app layer** (see [hook.md](hook.md)):

```
tool call arrives
│
│  ① PreToolUse           (sync)
│     hook may return permissionDecision: allow / deny / ask
│     hook may rewrite tool input via updatedInput
│
│  ② PermissionFunc       (the gate above)
│     Permit  → execute
│     Reject  → return error
│     Prompt  → continue to ③
│
│  ③ PermissionRequest    (sync, only on Prompt)
│     hook may decide on user's behalf, or update rules in-flight
│     no hook decides → show TUI dialog
│
│  ④ PermissionDenied     (async, only on final Deny)
│     hook may set retry: true to resume the assistant turn
```

`PreToolUse` runs **before** the gate — it can short-circuit the pipeline.
`PermissionRequest` runs **only after** the gate returns Prompt — it can
auto-approve before the user dialog. `PreToolUse` cannot return
`updatedPermissions`; that is exclusive to `PermissionRequest`.

---

## Testing

### Unit + integration

```bash
go test ./internal/setting/...      -v -run TestPermission
go test ./internal/tool/perm/...    -v
go test ./internal/subagent/...     -v -run 'Mode|Permission|Allow|Deny'
go test ./tests/integration/permission/... -v
```

Key cases:

```
# Unified pipeline
TestMatchRule                        rule pattern matching
TestBuildRule                        rule string construction
TestCheckPermission                  deny / allow / ask / mode interactions
TestBypassPermissionsMode            bypass + bypass-immune still enforced
TestDontAskMode                      Ask coerced to Deny
TestDenyRulesPriorityOverSession     deny absolute
TestSafeToolAllowlist                safe tools auto-allow
TestIsDestructiveCommand             destructive pattern catch
TestIsSensitivePath                  sensitive path catch
TestCheckBashSecurity                injection / obfuscation
TestBashSecurityBypassImmune         security checks always fire

# Subagent gate (steps 1-4 + Ask→Deny coercion)
TestExploreModeAllowsOnlyGitDiffBash         allow_tools per-subcommand
TestDefaultModeRestrictsConfiguredBash       allow_tools whitelist
TestDenyToolRulesMatchPatterns               deny_tools any-subcommand
TestExploreModeFiltersMutatingToolSchemas    schema filter
TestAcceptEditsModeFiltersApprovalOnlyToolSchemas
TestBypassModeAllowsEverything
TestNormalizePermissionModeDefaultsEmpty

# Tool classification
TestIsReadOnlyToolMatchesConfig
TestIsSafeToolMatchesConfig
```

### Manual interactive (subagent, headless)

`gen agent run --type <name> --prompt "..."` runs an AGENT.md fixture through
the full subagent pipeline. Use it to verify allow / deny / mode end-to-end
without a TUI.

```bash
mkdir -p .gen/agents
cat > .gen/agents/test-perm.md <<'EOF'
---
name: test-perm
description: Permission gate fixture
mode: explore
allow_tools:
  - Read
  - Bash(git diff*)
  - Bash(git log*)
deny_tools:
  - Bash(git stash*)
---
You are a test fixture. Run exactly what the user asks. After each call output:
  RESULT: <tool>(<short-args>) -> <ALLOWED|DENIED: reason>
EOF

# Allow path (matches Bash(git diff*))
./bin/gen agent run --type test-perm --prompt 'Run bash: git diff --stat'
#   → RESULT: Bash(git diff --stat) -> ALLOWED

# Per-subcommand allow rejects when any part doesn't match
./bin/gen agent run --type test-perm --prompt 'Run bash: git diff && git status'
#   → DENIED: tool Bash call is outside the allow_tools constraint

# Deny wins over allow
./bin/gen agent run --type test-perm --prompt 'Run bash: git stash list'
#   → DENIED: tool Bash is blocked by deny_tools

# Bypass-immune wins over everything below
./bin/gen agent run --type test-perm --prompt 'Run bash: git diff && rm -rf /tmp/dummy'
#   → DENIED: destructive command
```

### Manual interactive (main loop, TUI)

```bash
mkdir -p /tmp/perm_test/.gen
cat > /tmp/perm_test/.gen/settings.local.json <<'EOF'
{"permissions": {"allow": ["Bash(echo *)", "Bash(ls *)"]}}
EOF

tmux new-session -d -s t_perm -x 220 -y 60
tmux send-keys -t t_perm 'cd /tmp/perm_test && gen' Enter
sleep 2

# Compound where every subcommand matches → no dialog
tmux send-keys -t t_perm 'Run bash: echo hi && ls /tmp' Enter
sleep 5
tmux capture-pane -t t_perm -p

# Compound where one subcommand has no allow rule → approval dialog
tmux send-keys -t t_perm 'Run bash: echo hi && cat /etc/hosts' Enter
sleep 5
tmux capture-pane -t t_perm -p

tmux kill-session -t t_perm
rm -rf /tmp/perm_test
```
