---
name: add-resolver
description: Scaffold a new resolver module for huh. Use when adding support for a new input type (port, process, device, binary, or a new one). Pass the resolver name as an argument.
disable-model-invocation: false
---

The user wants to add a new resolver named `$ARGUMENTS`.

1. Create `internal/<resolver-name>/resolver.go` with:
   - A `Resolve(input string) (*Result, error)` function
   - A `Result` struct with at minimum a `Summary` string field
   - A comment explaining what input types this resolver handles

2. Create `internal/<resolver-name>/resolver_test.go` with at least one table-driven test covering a happy path and an error case.

3. Wire the new resolver into the input classifier (`internal/classify/classify.go` or equivalent). If the classifier doesn't exist yet, note that it needs to be created.

4. Update the README.md resolver list if this is a new type not already mentioned.

Follow Go standard library only — no third-party runtime imports. Use conventional commit style for any commit messages.
