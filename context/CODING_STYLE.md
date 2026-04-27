# Coding Style Guide

## Design Principles

### SOLID Principles

- **Single Responsibility**: Each struct/function should have one reason to change
- **Open/Closed**: Open for extension, closed for modification
- **Liskov Substitution**: Interfaces should be substitutable by their implementations
- **Interface Segregation**: Prefer small, focused interfaces over large ones
- **Dependency Inversion**: Depend on abstractions (interfaces), not concrete implementations

### Domain-Driven Design (DDD)

- Organize code around business domains, not technical concerns
- Use ubiquitous language from the domain in code (types, functions, variables)
- Define clear bounded contexts with explicit boundaries
- Separate domain logic from infrastructure concerns
- Use value objects for immutable domain concepts
- Use entities for objects with identity and lifecycle
- Use aggregates to enforce consistency boundaries

### Hexagonal Architecture (Ports & Adapters)

```
┌─────────────────────────────────────────┐
│            Adapters (outer)             │
│  ┌───────────────────────────────────┐  │
│  │         Ports (interfaces)        │  │
│  │  ┌─────────────────────────────┐  │  │
│  │  │     Domain (core logic)     │  │  │
│  │  └─────────────────────────────┘  │  │
│  └───────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

- **Domain**: Pure business logic with no external dependencies
- **Ports**: Interfaces defining how the domain interacts with the outside world
- **Adapters**: Implementations that connect ports to external systems (DB, HTTP, etc.)

Dependencies point inward—adapters depend on ports, ports depend on domain. Domain depends on nothing external.

#### Composition Root at the Edge

The composition root — where concrete adapters are wired to ports and services — belongs in `cmd/`, the outermost layer. Moving it inward (e.g., into `internal/app/`) would force the app layer to depend on concrete adapters, violating dependency inversion.

```
cmd/         → wires github.NewClient (adapter) into app.NewRepoService (service)
internal/app → depends only on ports.CodeRepositoryProvider (interface)
```

#### Thin Shell, Rich Core

Commands (`cmd/`) should only extract input and print output. All business logic — including formatting and error wrapping — belongs in the app or domain layer where it's testable without framework dependencies.

```go
// Good: command is a thin shell
func runList(cmd *cobra.Command, _ []string) error {
    opts := extractFlags(cmd)
    output, err := svc.ListCodeRepositories(ctx, opts)
    if err != nil { return err }
    fmt.Print(output)
    return nil
}

// Bad: command contains business logic
func runList(cmd *cobra.Command, _ []string) error {
    repos, err := svc.ListRepositories(ctx, opts)
    if err != nil { return fmt.Errorf("failed to list: %w", err) }
    formatted := formatTable(repos) // formatting in cmd layer
    fmt.Print(formatted)
    return nil
}
```

### Functional Idioms

Apply functional programming concepts where they improve clarity and correctness:

#### Pure Functions

- Prefer functions that return results based only on inputs, with no side effects
- Pure functions are easier to test, reason about, and compose
- Isolate side effects (I/O, state mutation) at the boundaries of the system

```go
// Pure - easy to test and reason about
func CalculateTotal(prices []float64, taxRate float64) float64 {
    sum := 0.0
    for _, p := range prices {
        sum += p
    }
    return sum * (1 + taxRate)
}

// Impure - depends on external state
func CalculateTotal() float64 {
    return globalCart.Sum() * (1 + config.TaxRate)
}
```

#### Immutability

- Prefer returning new values over mutating existing ones
- Use value types and avoid pointer receivers when mutation isn't needed
- Make structs immutable by design where practical

```go
// Prefer: return new slice
func Append(items []Item, item Item) []Item {
    return append(items, item)
}

// Avoid: mutate in place
func (c *Cart) AddItem(item Item) {
    c.items = append(c.items, item)
}
```

#### First-Class Functions

- Pass functions as arguments for flexible, reusable code
- Use function types to define contracts

```go
type Predicate[T any] func(T) bool
type Mapper[T, R any] func(T) R

func Filter[T any](items []T, pred Predicate[T]) []T {
    result := make([]T, 0)
    for _, item := range items {
        if pred(item) {
            result = append(result, item)
        }
    }
    return result
}
```

#### Function Composition

- Build complex behavior by combining simple functions
- Keep individual functions small and focused

```go
func Compose[T any](fns ...func(T) T) func(T) T {
    return func(input T) T {
        result := input
        for _, fn := range fns {
            result = fn(result)
        }
        return result
    }
}

// Usage
normalize := Compose(strings.TrimSpace, strings.ToLower)
```

#### Option Pattern for Nullable Values

- Use explicit types instead of nil pointers for optional values
- Makes absence of value explicit in the type system

```go
type Option[T any] struct {
    value   T
    present bool
}

func Some[T any](v T) Option[T] { return Option[T]{value: v, present: true} }
func None[T any]() Option[T]    { return Option[T]{} }

func (o Option[T]) GetOrElse(defaultVal T) T {
    if o.present {
        return o.value
    }
    return defaultVal
}
```

#### Result Pattern for Error Handling

- Encapsulate success/failure in a single return type when chaining operations
- Useful for pipelines where errors should short-circuit

```go
type Result[T any] struct {
    value T
    err   error
}

func (r Result[T]) Map(fn func(T) T) Result[T] {
    if r.err != nil {
        return r
    }
    return Result[T]{value: fn(r.value)}
}

func (r Result[T]) FlatMap(fn func(T) Result[T]) Result[T] {
    if r.err != nil {
        return r
    }
    return fn(r.value)
}
```

#### Avoid Shared Mutable State

- Pass dependencies explicitly rather than using package-level variables
- Use channels or immutable data for concurrent code
- When mutation is necessary, keep it localized and well-documented

## Development Methodology

### Test-Driven Development (TDD)

Follow the Red-Green-Refactor cycle:

1. **Red**: Write a failing test that defines expected behavior
2. **Green**: Write minimal code to make the test pass
3. **Refactor**: Improve code structure while keeping tests green

Guidelines:
- Write tests before implementation code
- Each test should verify one behavior
- Tests should be independent and isolated
- Use table-driven tests for multiple similar cases in Go