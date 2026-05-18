# Design Principles

## Documentation

- Keep short navigation in `AGENTS.md`.
- Keep durable explanations in `docs/`.
- Prefer a single obvious index for each documentation area.
- Document package ownership and dependency direction before large refactors.

## Code Structure

- Favor cohesive capability packages over abstract layer names.
- Keep `internal/app` as composition, not as a place for feature behavior.
- Keep `internal/core` stable and dependency-light.
- Move shared contracts downward only when multiple packages need them.
- Avoid large rewrites when a file split or small interface will solve the
  coupling problem.
