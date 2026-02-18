# ZanGuard docs-updater agent memory

## Project structure

- Docs site: `/Users/thilinashashimalsenarath/Documents/zengard/docs-site/`
- Source docs: `docs-site/docs/` — one folder per topic area
- Sidebar config: `docs-site/sidebars.js` (single `docs` array, categories use `collapsed: false`)
- API source: `pkg/api/` — `types.go`, `server.go`, `management.go`, `authzen.go`, `helpers.go`
- Model types: `pkg/model/` — `tenant.go`, `tuple.go`, `changelog.go`, `attributes.go`
- Entry point: `cmd/server/main.go`

## Documentation conventions

- Frontmatter fields in use: `id`, `title`, `sidebar_position`
- `id` must match the path segment used in `sidebars.js` (e.g. `id: overview` → `'api/overview'`)
- `sidebar_position` is scoped within the category, starting from 1
- H1 heading always follows the frontmatter block
- Section headings use `##` for major sections, `###` for individual endpoints
- Endpoint reference pattern: bold code line `**\`METHOD /path\`**`, then a description paragraph
- Tables used for: endpoint summaries, request fields, query parameters, response fields, status codes
- Code blocks: ` ```bash ` for curl, ` ```json ` for bodies, ` ```yaml ` for YAML
- Cross-links use relative Docusaurus paths: `[text](./path/page)`

## Style

- Present tense, second person ("Returns...", "Sends...", "Use...")
- Descriptions are concise — one sentence per field/parameter where possible
- Error response tables list status code and plain-English condition
- No trailing punctuation on table cells
- No emojis

## API-specific patterns

- Management API: `/api/v1/` — no `X-Tenant-ID` for tenant CRUD (ID in URL), required for all data-plane ops
- AuthZen API: `/access/v1/` — always requires `X-Tenant-ID`
- Error envelope: `{"error": "message"}` for all errors; schema errors add `"details": [...]`
- AuthZen spec: engine errors → `decision:false` + `200 OK` (not an HTTP error)
- Default server addr: `:8080`; override via `ZANGUARD_ADDR` env var
- Server start command: `go run ./cmd/server/main.go`

## Details file

See `patterns.md` for extended notes (none yet — add when patterns accumulate).
