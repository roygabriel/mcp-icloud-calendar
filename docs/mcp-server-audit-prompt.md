# Go MCP Server Audit & Improvement Prompt

Use this document as a prompt file when auditing Go MCP servers for production readiness. Read the entire codebase first, then investigate each category below. Produce a prioritized improvement plan with dependency ordering.

---

## 1. Critical Bugs & Race Conditions (P0)

Investigate these first — they cause data corruption or crashes in production.

- **UID/ID generation**: Are unique identifiers generated with `time.Now().UnixNano()`, incrementing counters, or other collision-prone methods? Replace with `uuid.New().String()` or crypto/rand.
- **Shared state without synchronization**: Look for package-level or struct-level caches (connection pools, discovery results, token caches) accessed from multiple goroutines without `sync.Mutex` or `sync.Once`. The MCP stdio server handles requests concurrently.
- **Goroutine leaks**: Are goroutines launched without cancellation paths? Check for missing `context.Context` propagation, unbuffered channels with no readers, or `time.After` in select without context deadline.
- **Nil pointer dereferences**: Are optional fields from `req.GetArguments()` type-asserted without nil checks? JSON numbers arrive as `float64`, booleans as `bool` — verify all type assertions handle the zero/missing case.

## 2. No Test Coverage (P0)

- **Zero or near-zero tests**: Check for `*_test.go` files. If absent, this is the single highest-leverage improvement.
- **Testability blockers**: Are CalDAV/API clients passed as concrete types (`*Client`) instead of interfaces? Extract a service interface so handlers can be tested with mocks.
- **Mock client**: Create a mock implementing the service interface with per-method error injection fields and call-tracking fields (LastPath, CallCount, etc.).
- **Table-driven tests**: Each tool handler needs at minimum: happy path, missing required fields, invalid input formats, backend error propagation, and input validation (path traversal).
- **Race detector**: Tests must pass under `go test -race ./...`.

## 3. Structured Logging (P0)

- **fmt.Printf / log.Printf to stderr**: Replace with `log/slog` using `slog.NewJSONHandler(os.Stderr, ...)`. MCP servers communicate over stdin/stdout — any stray `fmt.Println` will corrupt the JSON-RPC stream.
- **Configurable log level**: Add a `LOG_LEVEL` env var (DEBUG, INFO, WARN, ERROR) parsed at startup.
- **Contextual fields**: Include request IDs, tool names, account names, and operation durations in log entries. Never log PII (event titles, descriptions, user data) in production logs.

## 4. Graceful Shutdown & Signal Handling (P0)

- **No signal handling**: Check if the server just calls `server.ServeStdio()` with no way to clean up. Use `server.NewStdioServer(s).Listen(ctx, os.Stdin, os.Stdout)` with a cancellable context.
- **SIGTERM/SIGINT handling**: Register signal handlers that cancel the root context, drain in-flight requests, close health servers, and flush metrics.
- **Resource cleanup**: Database connections, HTTP clients, file handles — ensure they're closed on shutdown.

## 5. Input Validation & Sanitization (P1)

- **Path traversal**: Any parameter used to construct file paths, URLs, or CalDAV/API paths must reject `..`, null bytes (`\x00`), and newlines. Write a `ValidatePath()` helper.
- **ID injection**: Event IDs, resource IDs, or entity IDs should not contain `/`, `..`, or control characters.
- **Required field validation**: Don't rely solely on MCP schema `Required()` — validate in the handler too, since clients may not enforce schema.
- **Time parsing**: Use `time.Parse(time.RFC3339, ...)` and return clear error messages with example format. Validate that end > start where applicable.

## 6. Retry Logic & Resilience (P1)

- **No retry on transient failures**: Network errors, 5xx responses, and rate limit responses should be retried with exponential backoff.
- **Retrying non-idempotent operations**: `CreateEvent`, `SendMessage`, `POST` operations must NOT be retried — only idempotent reads and deletes.
- **Missing context timeout**: Wrap each tool handler's context with `context.WithTimeout()` to prevent hung requests from blocking the server. Use `WithToolHandlerMiddleware` for this.
- **No rate limiting**: Add token-bucket rate limiting (`golang.org/x/time/rate`) to external API clients to avoid being throttled or banned.

## 7. Configuration Hardening (P1)

- **No validation**: Config values (URLs, ports, durations, counts) are read from env vars but never validated. Add a `Validate()` method checking ranges, formats, and required fields.
- **Secrets in env vars**: Support `file://` prefix for sensitive values so Docker/K8s secrets can be mounted as files instead of passed as environment variables.
- **Missing defaults**: Every optional config field should have a sensible default. Document defaults in the config struct or help text.
- **go.mod version vs CI mismatch**: Check that CI uses `go-version-file: 'go.mod'` instead of a hardcoded Go version that may drift.

## 8. Observability (P1)

- **No health endpoint**: Add `/healthz` (liveness) and `/readyz` (readiness) HTTP endpoints for container orchestrators. Readiness should be false during startup and shutdown.
- **No metrics**: Add Prometheus counters and histograms for tool calls (`tool_calls_total{tool,status}`, `tool_call_duration_seconds{tool}`) and backend requests. Expose via `/metrics` on the health port.
- **No audit logging**: Mutating operations (create, update, delete) should produce structured audit log entries with tool name, resource ID, and status — but never PII.
- **No request ID**: Generate a UUID per tool call, inject into context and slog fields so logs can be correlated across a single request.

## 9. Interface & Architecture (P2)

- **Concrete client types in handlers**: Tool handlers should accept an interface, not `*Client`. This enables testing, decorator patterns (retry, rate limit, metrics), and multi-account support.
- **Decorator chain**: Wrap the real client in layers: `realClient → rateLimitedClient → retryClient`. Each layer implements the same interface.
- **Middleware chain**: Use `server.WithToolHandlerMiddleware()` for cross-cutting concerns: request ID injection, timeouts, metrics recording. Avoid duplicating this logic in every handler.
- **Update semantics**: If the server supports update operations, check whether omitting a field means "don't change" vs "clear the field". Use pointer fields (`*string`) to distinguish nil (skip) from empty string (clear).

## 10. Build & CI (P2)

- **No Makefile**: Add targets for `build`, `test`, `lint`, `clean`, `docker`, `run`. Inject version via `-ldflags "-X main.version=$(VERSION)"` using `git describe`.
- **No linting**: Add `.golangci.yml` enabling at minimum: errcheck, govet, staticcheck, gosec, unused, ineffassign. Run in CI.
- **No vulnerability scanning**: Add `govulncheck ./...` to CI. Add Dependabot or Renovate for dependency updates.
- **No container image**: Add a multi-stage Dockerfile (builder + distroless/static runtime). Run as non-root. Image should be < 20MB.

## 11. Tool Schema Quality (P1)

An AI agent's only understanding of your server comes from the tool list response. Poorly described tools with missing constraints lead to agents guessing parameter formats, omitting required context, and making unnecessary calls. Audit every tool definition for the following:

### Tool-Level Descriptions
- **Vague descriptions**: Descriptions like "Create an event" or "Search items" don't tell the agent what fields are returned, what prerequisites exist, or what side effects occur. Each tool description should answer: what does it do, what does it return, and what should the agent call first?
- **No workflow guidance**: Tools should cross-reference each other. A search tool should say "Use list_calendars first to discover valid calendarId values." A delete tool should say "Use search_events first to find the event's id." Without this, agents guess at parameter values.
- **Missing response shape**: Describe the key fields in the response so agents know what to expect. e.g., "Returns each calendar's path (use as calendarId in other tools), display name, description, and color."

### Tool Annotations
- **Missing `WithReadOnlyHintAnnotation`**: Set `true` for list/search/get tools, `false` for create/update/delete. Agents use this to assess whether a tool call is safe.
- **Missing `WithDestructiveHintAnnotation`**: Set `true` for delete and any irreversible operation. This signals the agent to confirm with the user before calling.
- **Missing `WithIdempotentHintAnnotation`**: Set `true` for reads, updates, and deletes. Set `false` for creates (calling twice produces duplicates). Agents use this to decide whether retrying a failed call is safe.

### Field Constraints
- **No `DefaultNumber` / `DefaultBool` / `DefaultString`**: If a field has a default value, declare it in the schema — not just in the description text. Agents read schema properties programmatically.
- **No `Min` / `Max` on numbers**: Pagination fields (limit, offset) and quantity fields should have bounds. e.g., `limit`: `Min(1)`, `Max(500)`, `DefaultNumber(50)`.
- **No `MinLength` on required strings**: If a string is required and must be non-empty, add `MinLength(1)` so schema validation catches it before the handler.
- **No `Enum` on constrained strings**: If a field only accepts specific values (status, role, priority, type), use `Enum(...)` to enumerate them. This prevents agents from guessing invalid values.
- **No `Pattern` on formatted strings**: If a field must match a specific format (email, UUID, date), use `Pattern(...)` with a regex.

### Description Consistency
- **Inconsistent format names**: Pick one term and stick with it across all tools. Use "RFC 3339 format" not a mix of "ISO 8601", "RFC 3339", and "ISO format". Always include an example.
- **Inconsistent parameter naming**: If the same concept (e.g., a resource path) appears on multiple tools, use the same parameter name and the same description text everywhere.
- **Ambiguous terminology**: Don't say "ID/path" — pick one. If it's a path, call it a path and explain where to get it ("Calendar path from list_calendars").
- **Update field semantics unexplained**: If the server supports partial updates, explain the semantics in each field description: "Omit to keep the current value. Set to empty string to clear."

### Attendees / Complex Nested Parameters
- **JSON-in-a-string without full field docs**: If a parameter accepts a JSON string encoding a complex object (e.g., attendees, metadata), document every field in the description including which are required, which are optional, and what the valid values are for enum-like fields.

## 12. Advanced Features (P3)

Check whether these are relevant to the specific server's domain:

- **Multi-account support**: If users may have multiple accounts/credentials, support an `ACCOUNTS_FILE` JSON config mapping account names to credentials. Each account gets its own client chain. Add an `account` parameter to all tools.
- **Pagination**: List/search operations returning unbounded results need `limit` and `offset` parameters with sensible defaults (e.g., limit=50).
- **Recurrence / expansion**: Calendar servers need RRULE expansion. Similar domain-specific expansion may apply to other servers (recurring tasks, scheduled jobs).
- **mTLS / custom TLS**: If the server talks to on-prem or enterprise backends, support client certificates (`TLS_CERT_FILE`, `TLS_KEY_FILE`, `TLS_CA_FILE`).
- **Connection pooling config**: Expose `MAX_CONNS_PER_HOST` or equivalent for the underlying HTTP client.

---

## Audit Process

1. **Read the entire codebase** — understand the package structure, tool registrations, and data flow before suggesting changes.
2. **Verify the MCP SDK version** — check `go.mod` for the `mcp-go` version, then confirm which APIs are available (middleware, hooks, stdio server, etc.) before referencing them in your plan.
3. **Categorize findings by priority** — P0 bugs and missing tests first, then P1 resilience, then P2 CI/build, then P3 features.
4. **Establish a dependency graph** — interface extraction must come before tests; tests must come before refactoring; middleware must come before metrics. Plan phases accordingly.
5. **Preserve backward compatibility** — single-account mode, existing env vars, and current tool schemas must continue to work. New features are additive.
6. **Produce an implementation plan** — for each phase, list new files, modified files, and verification steps. Get approval before writing code.
