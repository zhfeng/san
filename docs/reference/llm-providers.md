# LLM Providers

Provider overview currently lives in:

- `README.md`
- `docs/packages/llm.md`
- `minmax-provider.md`

Provider implementation lives under `internal/llm` and provider subpackages.
Product code should depend on the `llm` service and registry, not concrete
provider packages.
