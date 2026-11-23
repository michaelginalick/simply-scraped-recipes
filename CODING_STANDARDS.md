# Go Coding Standards & Guidelines

This document defines the coding conventions, project structure, and best practices for this repository. All contributors and automated tools must follow these standards.

**Reference:** [Google Go Style Best Practices](https://google.github.io/styleguide/go/best-practices)


## 1. Tooling & Formatting

* **Formatting:** All code must be formatted with `gofmt` (or `goimports` to manage import grouping). Never commit unformatted code.
* **Linting:** Run `golangci-lint run` before opening a pull request. CI enforces zero linter warnings.
* **Dependencies:** Manage dependencies with Go Modules (`go.mod` / `go.sum`). Prefer the standard library over third-party packages unless a dependency provides substantial value. Do not commit a `vendor/` directory.

## 2. Project Layout

Follow the community-standard layout (see [golang-standards/project-layout](https://github.com/golang-standards/project-layout/blob/master/README.md "golang-standards/project-layout")):

```text
├── cmd/                # Application entrypoints (main.go per binary)
├── internal/           # Private application and library code
│   ├── handler/        # HTTP/gRPC transport handlers
│   ├── service/        # Core business logic
│   └── repository/     # Data persistence and external integrations
├── pkg/                # Public libraries safe for external import (use sparingly)
├── api/                # OpenAPI/Swagger specs, Protobuf definitions
└── scripts/            # Build, install, and CI helper scripts
```

* Keep `main.go` minimal: parse configuration, wire dependencies, and start the server.
* Avoid package-level global state; pass dependencies explicitly via structs and constructors (`NewService(...)`).

## 3. Naming & Style

* **Packages:** Use short, lowercase, single-word names (`user`, `auth`, `billing`). Avoid generic names like `util`, `common`, or `helpers`.
* **Identifiers:** Use `MixedCaps` or `mixedCaps` rather than underscores. Keep initialisms consistent in case (`userID`, `HTTPServer`, `URL`).
* **Interfaces:** Name single-method interfaces with an `-er` suffix (`Reader`, `Validator`, `UserStore`). Define interfaces in the consumer package, not the producer package.
* **Control Flow:** Keep the happy path left-aligned. Use early returns (`if err != nil { return err }`) to minimize indentation and avoid unnecessary `else` blocks.
* **Zero Values:** Design structs so their zero values are valid and ready to use without mandatory initialization where possible.

## 4. Error Handling

* **Never ignore errors:** Explicitly handle every returned `error`, or document why it is safely ignored (e.g., `_ = resp.Body.Close()`).
* **Wrap with context:** Add operational context when propagating errors using `fmt.Errorf("failed to fetch user %s: %w", id, err)`.
* **Inspecting errors:** Always use `errors.Is` and `errors.As` instead of string matching or direct equality checks.
* **Sentinel errors:** Define package-level sentinel errors as `var ErrNotFound = errors.New("not found")`.
* **`%w` vs `%v`:** Use `%w` when callers need to inspect the error via `errors.Is`/`errors.As`. Use `%v` when simply annotating or translating to an external error space (e.g. gRPC status codes) where unwrapping is not meaningful.
* **Panics:** Libraries must return errors rather than panic. Panics must never cross package boundaries; recover only at public API entry points.

## 5. Concurrency & Context

* **Context propagation:** Pass `context.Context` as the first parameter (`ctx context.Context`) to any function performing I/O, network calls, or long-running work. Never store `context.Context` inside a struct.
* **Goroutine lifecycle:** Never fire-and-forget goroutines without a clear shutdown mechanism. Ensure every goroutine has a bounded lifetime tied to a `context.Context` or `sync.WaitGroup`.

## 6. Testing

* **Table-driven tests:** Use table-driven tests with `t.Run` subtests for testing multiple input/output cases.
* **Standard library first:** Use the `testing` package alongside `://github.com` or `://github.com` for assertions.
* **Parallelization:** Mark independent unit tests with `t.Parallel()`.
* **File naming:** Place unit tests in `foo_test.go` alongside `foo.go`. Use package `foo_test` for black-box testing of exported APIs.
* **`t.Fatal` vs `t.Error`:** Use `t.Fatal` (or `t.FailNow`) when a setup failure makes the rest of the test meaningless, or inside `t.Run` subtests for per-entry failures. Use `t.Error` with `continue` when a single table-entry failure should not stop the remaining entries from running.
* **Goroutine safety in tests:** Never call `t.Fatal`, `t.FailNow`, or `t.SkipNow` from a goroutine spawned inside a test — only the test goroutine may call these. Use `t.Error` from spawned goroutines instead.
* **Test helpers:** Call `t.Helper()` at the top of any helper function so failures are attributed to the call site. Use `t.Fatal` inside helpers when a setup failure makes continuing impossible.

## 7. Documentation & Comments

* Write complete-sentence comments starting with the name of the symbol for all exported packages, types, functions, and constants.
* Comment *why* a non-obvious decision was made, rather than narrating *what* the code does.

## 8. Rules

- Use the modern Go skill whenever writing Go code
- When scanning code ignore directories starting with _ or the testdata directory
- Interfaces should not have more than 4 methods, ask if you go above it
- Avoid using pointers as much as possible
- Use `go run` to check that code compiles, not `go build`
- Add an empty line after closing }, for example:

    if x > 10 {
        fmt.Println("x is big")
    }

    fmt.Println("done")
- Avoid global variables and init()
- Lower level packages should not import higher level packages (imports go down)
