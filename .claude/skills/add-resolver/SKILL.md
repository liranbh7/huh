---
name: add-resolver
description: Scaffold a new resolver module for huh. Use when adding support for a new input type (port, process, device, binary, or a new one). Pass the resolver name as an argument.
disable-model-invocation: false
---

The user wants to add a new resolver named `$ARGUMENTS`.

## 1. Create the resolver package

Create `src/resolvers/<resolver-name>/resolver.go` with:
- A `Resolve(input string) (*Result, error)` function
- A `Result` struct with at minimum a `Summary` string field
- A comment explaining what input types this resolver handles

Create `src/resolvers/<resolver-name>/resolver_test.go` with at least one table-driven test covering a happy path and an error case.

Note: nested namespaces are fine when the resolver belongs to a logical group (e.g. `src/resolvers/net/port`, `src/resolvers/net/ip`).

## 2. Wire into the classifier

Edit `src/classify/classify.go`:
- Add a new `Type` constant in the `const` block (e.g. `MyType`)
- Add a `case MyType: return "mytype"` in `Type.String()`
- Add detection logic in `Classify()` so inputs matching this type return `[]Type{MyType}`

## 3. Wire into the dispatcher (main.go)

Edit `src/main.go`:
- Add an import for `"github.com/liranbh7/huh/src/resolvers/<resolver-name>"`
- Add a `case classify.MyType:` branch in the switch that calls `<resolver>.Resolve(input)` and then `print.MyType(r)`

## 4. Add a print function

Edit `src/print/print.go`:
- Add an import for the new resolver package
- Add a `func MyType(r *myresolver.Result)` function that builds a `[]format.Row` and calls `format.Print(title, rows)`

## 5. Verify

Run `make build && make test` to confirm everything compiles and tests pass. Follow Go standard library only — no third-party runtime imports. Use conventional commit style for any commit messages.
