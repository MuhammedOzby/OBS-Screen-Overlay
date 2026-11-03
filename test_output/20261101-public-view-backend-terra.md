# Public view backend verification

Date: 2026-11-01

Scope: token lookup, separate public router, saved initial values (including SSE-disabled), HTML escaping and JS/SVG preservation, admin route isolation and listener/origin configuration validation. Tests use in-memory SQLite; the real overlay database and existing binary were not used.

Commands and actual results:

- `go test -count=1 ./...` — passed all packages; `lib` has no test files.
- `go vet ./...` — passed, no output.
- `go test -race -count=1 ./...` — passed all packages; `lib` has no test files.

Browser and OBS acceptance were not run. SSE, runtime, panel URL controls, template publishing and authentication are outside this change.
