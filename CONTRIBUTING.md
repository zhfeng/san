# Contributing to GenCode

Thanks for your interest in contributing! This guide will help you get started.

## Quick Start

```bash
git clone https://github.com/genai-io/gen-code.git
cd gen-code
go build -o gen ./cmd/gen
./gen
```

## Development

### Prerequisites

- Go 1.21+
- An LLM API key (Anthropic, OpenAI, or Google)

### Project Structure

```
cmd/               # Binary entrypoints (cmd/gen)
docs/              # Documentation (architecture.md + packages/ + concepts/ + reference/ + guides/ + operations/)
internal/          # All Go code; see docs/reference/package-map.md for the full table
tools/             # Developer tooling (layercheck, …)
tests/integration/ # Cross-package behavioral tests
notes/             # Work-in-progress plans (not durable docs)
```

See [`docs/architecture.md`](docs/architecture.md) for primitives and
the runtime model. See [`docs/reference/package-map.md`](docs/reference/package-map.md)
and [`docs/reference/dependency-rules.md`](docs/reference/dependency-rules.md)
for the full package list, layer assignment, and import rules. New
package contributions also need a [`docs/packages/<name>.md`](docs/packages/)
following [`docs/packages/TEMPLATE.md`](docs/packages/TEMPLATE.md).

### Run Tests

```bash
GOCACHE=/private/tmp/gencode-go-build-cache go test ./...
```

Transcript/session focused suites:

```bash
GOCACHE=/private/tmp/gencode-go-build-cache go test \
  ./internal/session/... ./tests/integration/session/... ./tests/integration/cli/...
```

Transcript storage layout, recording rules, and the event model are
documented in [`docs/packages/session.md`](docs/packages/session.md).

### Debug Mode

```bash
GEN_DEBUG=1 ./gen
# Logs written to ~/.gen/debug.log
```

## How to Contribute

### Report Bugs

Open an issue with:
- Steps to reproduce
- Expected vs actual behavior
- OS, Go version, and provider used

### Suggest Features

Open an issue describing:
- The problem you're solving
- Your proposed solution
- Alternative approaches considered

### Submit Code

1. Fork the repo
2. Create a branch: `git checkout -b feature/your-feature`
3. Make changes and test
4. Commit with sign-off: `git commit -s -m "feat: add feature"`
5. Push and open a PR

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add new feature
fix: resolve bug
docs: update documentation
refactor: restructure code
test: add tests
chore: maintenance tasks
```

## Areas for Contribution

| Area | Description |
|------|-------------|
| **Providers** | Add new LLM providers (Ollama, Mistral, etc.) |
| **Tools** | Create new built-in tools |
| **MCP** | Improve MCP server support |
| **TUI** | Enhance terminal UI/UX |
| **Docs** | Improve documentation |
| **Tests** | Increase test coverage |

## Code of Conduct

Be respectful and constructive. We welcome contributors of all backgrounds and experience levels.

## Questions?

Open an issue or start a discussion. We're happy to help!
