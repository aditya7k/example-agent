# Go Best Practices

## Project Structure

```
project-root/
├── cmd/
│   └── appname/
│       └── main.go          # Application entry point
├── internal/                 # Private application code
│   ├── domain/              # Domain models and business logic
│   ├── ports/               # Interface definitions
│   └── adapters/            # Interface implementations
├── pkg/                     # Public libraries (if any)
├── go.mod
├── go.sum
└── README.md
```

### Directory Conventions

- **cmd/**: Entry points for executables. Each subdirectory is a separate binary
- **internal/**: Code that cannot be imported by other projects (enforced by Go)
- **pkg/**: Code that can be imported by external projects (use sparingly)

## Coding Conventions

### Naming

- Use `camelCase` for unexported, `PascalCase` for exported
- Keep names short but descriptive: `repo` not `repository`, `cfg` not `configuration`
- Interfaces with single method: name after the method + "er" (`Reader`, `Writer`, `Stringer`)
- Avoid stuttering: `user.User` is fine, but prefer `user.Service` over `user.UserService`

### Error Handling

```go
// Return errors, don't panic
func DoSomething() error {
    if err := operation(); err != nil {
        return fmt.Errorf("operation failed: %w", err)
    }
    return nil
}

// Check errors immediately
result, err := SomeFunction()
if err != nil {
    return err
}
// use result
```

- Always handle errors explicitly
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Use sentinel errors for expected error conditions
- Use custom error types when callers need to inspect error details

### Interfaces

```go
// Define interfaces where they are used, not where implemented
// Keep interfaces small (1-3 methods)
type Repository interface {
    FindByID(ctx context.Context, id string) (*Entity, error)
}
```

- Accept interfaces, return concrete types
- Define interfaces in the consumer package, not the provider
- Prefer composition of small interfaces over large interfaces

### Context

- First parameter of functions that do I/O or long-running operations
- Never store context in structs
- Use for cancellation, deadlines, and request-scoped values only

```go
func (s *Service) Process(ctx context.Context, data Data) error {
    // ...
}
```

### Struct Initialization

```go
// Use named fields for clarity
client := &Client{
    Timeout:  30 * time.Second,
    MaxRetry: 3,
}

// Constructor functions for complex initialization
func NewClient(opts ...Option) *Client {
    // ...
}
```

### Package Design

- One package = one purpose
- Avoid circular dependencies
- Keep package APIs small and focused
- Use `internal/` to hide implementation details
- Avoid `init()` functions — they run automatically before tests, mutate global state, and cannot be controlled or reset. Use explicit builder/constructor functions that return fresh instances instead

```go
// Avoid: init registers on global state, untestable in isolation
var listCmd = &cobra.Command{Use: "list", RunE: runList}

func init() {
    rootCmd.AddCommand(listCmd)
    listCmd.Flags().StringP("sort", "s", "name", "Sort field")
}

// Prefer: explicit builder, each call returns a fresh instance
func newListCmd() *cobra.Command {
    cmd := &cobra.Command{Use: "list", RunE: runList}
    cmd.Flags().StringP("sort", "s", "name", "Sort field")
    return cmd
}
```

## Testing

### File Naming

- Test files: `*_test.go` in the same package
- External tests: `package foo_test` for black-box testing

### Test Structure

```go
func TestFunctionName_Scenario_ExpectedBehavior(t *testing.T) {
    // Arrange
    input := setupInput()

    // Act
    result, err := FunctionUnderTest(input)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

### Grouping Tests by Concern

When a single subject (e.g. a command, service, or struct) has tests that span different concerns, split them into separate top-level test functions rather than nesting everything under one parent. Group by _what_ is being verified, not _what_ is being tested.

- **Configuration/structure** tests — verify static setup: metadata, flags, argument rules, registration. These inspect the object without executing it.
- **Execution/behavior** tests — verify runtime outcomes: happy paths, error propagation, missing inputs. These call `Execute()` or the function under test.

Use the `TestSubject_Concern` naming convention so each top-level function communicates both the subject and the category of tests it contains.

```go
// Avoid: one monolithic parent mixing concerns
func TestGetCmd(t *testing.T) {
    t.Run("CommandSetup", func(t *testing.T) { /* ... */ })
    t.Run("RequiresExactlyOneArg", func(t *testing.T) { /* ... */ })
    t.Run("IsRegisteredOnRoot", func(t *testing.T) { /* ... */ })
    t.Run("NoToken", func(t *testing.T) { /* ... */ })
    t.Run("Success", func(t *testing.T) { /* ... */ })
    t.Run("ServiceError", func(t *testing.T) { /* ... */ })
}

// Prefer: separate parents per concern
func TestGetCmd_Configuration(t *testing.T) {
    t.Run("CommandSetup", func(t *testing.T) { /* ... */ })
    t.Run("RequiresExactlyOneArg", func(t *testing.T) { /* ... */ })
    t.Run("IsRegisteredOnRoot", func(t *testing.T) { /* ... */ })
}

func TestGetCmd_Execution(t *testing.T) {
    t.Run("NoToken", func(t *testing.T) { /* ... */ })
    t.Run("Success", func(t *testing.T) { /* ... */ })
    t.Run("ServiceError", func(t *testing.T) { /* ... */ })
}
```

Splitting by concern keeps each test function focused, makes `go test -run` filtering more useful (e.g. `go test -run TestGetCmd_Execution`), and mirrors the Single Responsibility principle at the test level.

### Test Constants for Repeated Literals

Extract magic strings (URLs, usernames, paths) into composable constants at the top of the test file. Build specific values from a shared base to ensure consistency and make changes single-point.

```go
// Avoid: magic strings scattered across tests
mux.HandleFunc("/api/v3/user/repos", handler)
mux.HandleFunc("/api/v3/repos/testuser/my-repo", handler)

// Prefer: composable constants
const (
    apiBase      = "/api/v3/"
    userReposAPI = apiBase + "user/repos"
    reposAPI     = apiBase + "repos/"
    testUser     = "testuser"
)

mux.HandleFunc(userReposAPI, handler)
mux.HandleFunc(reposAPI+testUser+"/my-repo", handler)
```

### Test Fixtures with `t.Cleanup`

When multiple tests need the same infrastructure (HTTP server, database, temp files), extract it into a helper that returns ready-to-use values and registers cleanup via `t.Cleanup` rather than `defer`. This keeps each test to a single setup call and guarantees cleanup runs regardless of scope.

```go
// Avoid: repeated boilerplate in every test
mux := http.NewServeMux()
server := httptest.NewServer(mux)
defer server.Close()
client := createTestClient(server.URL)

// Prefer: single helper, cleanup via t.Cleanup
func newTestServer(t *testing.T) (*http.ServeMux, *Client) {
    t.Helper()
    mux := http.NewServeMux()
    server := httptest.NewServer(mux)
    t.Cleanup(server.Close)
    // ... create client pointing to server ...
    return mux, client
}

// Usage — one line replaces four
mux, client := newTestServer(t)
```

Mark helpers with `t.Helper()` so failure stack traces point to the calling test, not the helper.

### Shared Handler Helpers

When multiple tests register the same HTTP handler (e.g. a user endpoint returning a fixed response), extract it into a small function that takes the mux. Keep the helper focused on one endpoint — tests that need different behaviour (e.g. returning an error) should still use inline handlers.

```go
// Avoid: identical handler duplicated across tests
mux.HandleFunc(userAPI, func(w http.ResponseWriter, _ *http.Request) {
    json.NewEncoder(w).Encode(&User{Login: "testuser"})
})

// Prefer: shared helper for the common case
func handleUserAPI(mux *http.ServeMux) {
    mux.HandleFunc(userAPI, func(w http.ResponseWriter, _ *http.Request) {
        json.NewEncoder(w).Encode(&User{Login: testUser})
    })
}

// Usage
mux, client := newTestServer(t)
handleUserAPI(mux)
```

Only extract handlers that are truly identical. If a test needs a different response (e.g. 401 Unauthorized), keep it inline — that's the test's distinguishing behaviour.

### Table-Driven Tests

```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive numbers", 2, 3, 5},
        {"with zero", 0, 5, 5},
        {"negative numbers", -1, -2, -3},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Add(tt.a, tt.b)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Testing Commands

- Test through production builders with `SetArgs()` and `Execute()`, not hand-wired test commands. This ensures tests exercise the same code path as real users

```go
// Avoid: test helper duplicates production wiring
func testListCmd(token, sort string) *cobra.Command {
    cmd := &cobra.Command{RunE: runList}
    cmd.Flags().StringP("token", "t", token, "")
    cmd.Flags().StringP("sort", "s", sort, "")
    return cmd
}

// Prefer: use production builder, simulate args
func TestListCmd_Success(t *testing.T) {
    t.Setenv("GITHUB_TOKEN", "test-token")
    cmd := newListCmd()
    cmd.SetArgs([]string{"--sort", "stars"})

    err := cmd.Execute()

    assert.NoError(t, err)
}
```

- Don't test framework behavior — verifying Cobra's help output, exit codes, or arg validation logic tests the framework, not your code. These tests are brittle and break when commands or flags change

### Mocking with Testify

Mocks live in `internal/testutil/` and are shared across all test packages. Each mock implements a port interface and uses `testify/mock`.

#### One Mock Per Interface, in `testutil`

Define a single mock struct per port interface. Add a compile-time interface check so the mock stays in sync with the interface it implements.

```go
// internal/testutil/mock_provider.go
type MockRepositoryProvider struct {
    mock.Mock
}

func (m *MockRepositoryProvider) GetCodeRepository(ctx context.Context, name string) (*domain.CodeRepository, error) {
    args := m.Called(ctx, name)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*domain.CodeRepository), args.Error(1)
}

// Compile-time check — fails if the interface changes
var _ ports.CodeRepositoryProvider = (*MockRepositoryProvider)(nil)
```

Never define ad-hoc mock structs inside individual test files — duplicated mocks drift out of sync with the interface and with each other.

#### Nil-Guard Return Values

Mock methods that return pointers or slices must nil-guard `args.Get(0)` before type-asserting. Without this, `.Return(nil, err)` panics because `nil` cannot be asserted to a concrete type.

```go
// Correct: nil-guard before type assertion
args := m.Called(ctx, opts)
if args.Get(0) == nil {
    return nil, args.Error(1)
}
return args.Get(0).([]domain.CodeRepository), args.Error(1)
```

#### Setting Up Expectations

Use `.On()` to declare expected calls, and match arguments precisely when possible. Use `mock.Anything` only for parameters the test doesn't care about (typically `context.Context`).

```go
// Precise: verifies the exact repo name is passed
mockProvider.On("GetCodeRepository", mock.Anything, "my-repo").Return(&repo, nil)

// Less precise: use when the test doesn't care about filter values
mockProvider.On("ListCodeRepositories", mock.Anything, mock.Anything).Return(repos, nil)
```

#### Injecting Mocks via Factory Override

When the system under test creates its own service (e.g. commands that call `ServiceFactory`), override the factory in a helper that restores the original via `t.Cleanup`.

```go
func withMockService(t *testing.T, provider *testutil.MockRepositoryProvider) {
    t.Helper()
    original := ServiceFactory
    t.Cleanup(func() { ServiceFactory = original })
    ServiceFactory = func(token string) *app.CodeRepositoryService {
        return app.NewRepoService(provider)
    }
}

// Usage in a test
mockProvider := new(testutil.MockRepositoryProvider)
mockProvider.On("GetCodeRepository", mock.Anything, "my-repo").Return(&repo, nil)
withMockService(t, mockProvider)
```

When the test directly constructs the service, pass the mock through the constructor — no factory override needed.

```go
func newTestService() (*testutil.MockRepositoryProvider, *app.CodeRepositoryService) {
    mockProvider := new(testutil.MockRepositoryProvider)
    return mockProvider, app.NewRepoService(mockProvider)
}
```

#### Verifying Expectations

Always call `mockProvider.AssertExpectations(t)` in the Assert phase. This fails the test if any `.On()` expectation was not called, catching wiring bugs where the code silently skips the provider.

```go
// Assert
assert.NoError(t, err)
assert.Contains(t, output, "my-repo")
mockProvider.AssertExpectations(t)  // verifies all .On() calls were made
```

## Common Patterns

### Functional Options

```go
type Option func(*Config)

func WithTimeout(d time.Duration) Option {
    return func(c *Config) {
        c.Timeout = d
    }
}

func New(opts ...Option) *Service {
    cfg := defaultConfig()
    for _, opt := range opts {
        opt(cfg)
    }
    return &Service{cfg: cfg}
}
```

### Dependency Injection

```go
// Inject dependencies via constructor
func NewService(repo Repository, logger Logger) *Service {
    return &Service{
        repo:   repo,
        logger: logger,
    }
}
```