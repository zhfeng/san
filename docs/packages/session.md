---
package: github.com/genai-io/gen-code/internal/session
layer: feature
---

# session

Persists conversation transcripts to disk, generates session IDs, and
exposes save/load/list/fork operations. The deeper transcript event model
lives in the `transcript` subpackage.

## Purpose

Each gen-code run is a session. Sessions are auto-saved as JSONL transcripts
under `~/.gen/projects/<workdir-hash>/`. `--continue` and `--resume` use the
list/load APIs here to restore a previous session's messages; `--fork`
duplicates one mid-stream.

## Contract

```go
package session

// Service is the public contract for the session module.
type Service interface {
    // identity
    ID() string
    SetID(id string)
    TranscriptPath() string

    // store access
    GetStore() *Store
    SetStore(s *Store)
    EnsureStore(cwd string) error

    // persistence (delegates to store)
    Save(snap *Snapshot) error
    Load(id string) (*Snapshot, error)
    LoadLatest() (*Snapshot, error)
    List() ([]*SessionMetadata, error)
    Fork(id string) (*Snapshot, error)

    // tracing
    NewRecorder(agentID, provider, model string, maxTokens int) *Recorder
    Recorder() *Recorder
}
```

### Known Violations

- **Rule 1 (small).** 11 methods spanning identity, store access,
  persistence, and tracing. Suggested split:
  - `SessionIdentity` → `ID`, `SetID`, `TranscriptPath`
  - `SessionStoreAccess` → `GetStore`, `SetStore`, `EnsureStore`
  - `SessionPersistence` → `Save`, `Load`, `LoadLatest`, `List`, `Fork`
  - `SessionRecording` → `NewRecorder`, `Recorder`
- **Rule 7 (no escape hatch).** `GetStore() *Store` and `SetStore(*Store)`
  let callers reach into the concrete store. If callers need store
  methods, expose them on `Service` directly or have them depend on
  `*Store`.
- **Rule 5.** `Default()` returns `Service`. Reasoning same as elsewhere.

## Internals

- `Setup` (`setup.go`) — concrete implementation, holds `SessionID`,
  `Store`, and the current `Recorder` under a mutex.
- `Store` (`store.go`) — filesystem-backed JSON store under
  `~/.gen/projects/<hash>/`; provides Save / Load / List / Fork.
- `Recorder` (`recorder.go`) — writes the event-sourced transcript
  (one record per inference / tool call / hook / system mutation) into
  the `transcript` subpackage's filesystem store.
- `transcript/` (subpackage) — record types, JSONL store, projector that
  reconstructs `Snapshot` from event log.
- `convert.go`, `message_convert.go` — translate between in-memory
  `core.Message` and persisted forms.

## Lifecycle

- Construction: `Initialize(Options{CWD})` creates the store and a fresh
  session ID. Singleton thereafter.
- Per-run: agent emits events → `Recorder` writes records → `Snapshot`
  reconstructible at any point.
- Forks copy the underlying transcript file and assign a new session ID.

## Tests

```
internal/session/recorder_test.go         — recorder writes events correctly.
internal/session/recorder_order_test.go   — record ordering invariants.
internal/session/message_convert_test.go  — message ↔ record roundtrips.
internal/session/transcript/projector_test.go — replay correctness.
```

## See Also

- Code: `internal/session/`, `internal/session/transcript/`
- Replay UI: [`packages/inspector.md`](inspector.md)
- Layer: `feature`
