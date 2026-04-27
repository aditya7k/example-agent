# Agent Guidelines for claude-dojo-one


### Rules
 - Always use Context7 MCP when I need library/API documentation, code generation, setup or configuration steps without me having to explicitly ask.
 - Use github MCP for any questions related to PR's and Github issues in this repository.

This file provides context and instructions for autonomous coding agents working in this repository.

## 1. Build, Test & Lint Commands

The project uses a `Makefile` for standard operations. Agents should always run verification commands after making changes.

### Common Commands
- **Build**: `make build` - compiles the binary to `./bin/devportal`
- **Test (All)**: `make test` - runs all unit tests
- **Test (Verbose)**: `make test-verbose` - runs tests with verbose output
- **Lint**: `make lint` - runs `go vet` and other checks
- **Format**: `make fmt` - formats code using standard Go tools
- **Tidy**: `make tidy` - cleans up `go.mod` dependencies

### Running Specific Tests
To run a single test or a subset of tests, use `go test` directly:
```bash
# Run a specific test function
go test -run TestName ./...

# Run a specific subtest (e.g. TestGetCmd/Configuration)
go test -run TestGetCmd/Configuration ./...

# Run tests in a specific package
go test ./internal/app/...
```

### Coverage
- **Generate Report**: `make coverage` (creates `coverage.html`)
- **Print Summary**: `make coverage-report`

---

## 2. Code Style & Architecture

### Core Principles
- **Hexagonal Architecture**:
  - **Adapters (Outer)**: `cmd/`, `internal/adapters/` (HTTP, CLI, DB). Depend on ports.
  - **Ports (Middle)**: `internal/ports/` (Interfaces). Define contracts.
  - **Domain (Inner)**: `internal/domain/`. Pure business logic. Zero dependencies.
  - **Dependencies**: Always point inward. `cmd` -> `adapters` -> `ports` -> `domain`.
- **SOLID**: Adhere strictly, especially Single Responsibility and Dependency Inversion.
- **DDD**: Use domain language. Separate domain logic from infrastructure.

### Go Conventions
- **Naming**:
  - `camelCase` for unexported, `PascalCase` for exported identifiers.
  - Short but descriptive (`cfg` over `configuration`, `repo` over `repository`).
  - Avoid stuttering (`user.Service` instead of `user.UserService`).
  - Interfaces: Method name + "er" (e.g., `Reader`, `Writer`) for single-method interfaces.
- **Project Structure**:
  - `cmd/`: Entry points. Thin shells only. Extract flags, call service, print output. No business logic.
  - `internal/`: Private application code.
- **Error Handling**:
  - **Never panic**. Return errors.
  - Wrap errors with context: `fmt.Errorf("failed to process: %w", err)`.
  - Handle errors immediately after the call (guard clauses).
- **Concurrency**:
  - Use `context.Context` as the first argument for I/O or long-running tasks.
  - Avoid shared mutable state. Use channels or immutable data.

### Functional Idioms
- **Pure Functions**: Prefer functions with no side effects.
- **Immutability**: Return new values instead of mutating existing ones.
- **Option Pattern**: Use `Option[T]` or functional options for configuration.
- **Result Pattern**: Encapsulate success/failure for chaining operations.

---

## 3. Testing Guidelines

### TDD Methodology
Follow the Red-Green-Refactor cycle:
1. **Red**: Write a failing test.
2. **Green**: Write minimal code to pass.
3. **Refactor**: Improve structure while keeping tests green.

### Structure
- **Table-Driven Tests**: Use strictly for multiple scenarios.
- **Grouping**: Group tests by concern: `TestSubject_Concern` (e.g., `TestGetCmd_Configuration`, `TestGetCmd_Execution`).
- **Teardown**: Use `t.Cleanup` for cleanup, not `defer`.
- **Helpers**: Use `t.Helper()` for shared setup code.

### Mocking
- **Library**: `stretchr/testify`.
- **Location**: Mocks reside in `internal/testutil/`. One mock per interface.
- **Factory**: If testing a service factory, override it in a helper and restore via `t.Cleanup`.
- **Assertions**: Strictly verify expectations: `mockObj.AssertExpectations(t)`.

### Fixtures
Use helper functions that return initialized objects and register cleanup.
```go
func newTestServer(t *testing.T) (*http.ServeMux, *Client) {
    t.Helper()
    // ... setup ...
    t.Cleanup(server.Close)
    return mux, client
}
```

---

## 4. Libraries
- **CLI**: `spf13/cobra`
- **Testing**: `stretchr/testify`
