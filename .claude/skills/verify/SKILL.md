---
name: verify
description: Build the huh binary and run smoke tests against sample inputs to verify the tool works end-to-end. Use after implementing or changing a resolver.
disable-model-invocation: false
---

1. Run `make build`. Fix any build errors before continuing. The binary is written to `./bin/huh`.

2. Run `make test`. Report any failures.

3. Run a quick smoke test against each implemented resolver type. For each, run `./bin/huh <input>` and check that:
   - Output is non-empty and human-readable
   - No panic or error exit
   - Completes in under 500ms

   Sample inputs to try (skip any whose resolver isn't implemented yet):
   - Port: `./bin/huh 22`, `./bin/huh 80`
   - PID: `./bin/huh 1`
   - Process name: `./bin/huh systemd`
   - Binary: `./bin/huh ls`
   - Path: `./bin/huh /dev/sda` (only if block device exists)
   - IP: `./bin/huh 127.0.0.1`

4. Report what passed, what failed, and any unexpected output.
