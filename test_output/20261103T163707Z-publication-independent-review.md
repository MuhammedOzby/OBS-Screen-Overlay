# GitHub publication preparation — independent verification

Date: November 3, 2026, 19:37:07 Europe/Istanbul (UTC+03:00).

## Scope

English README.md, Turkish README-TR.md, AI development notes and language links; installation/CGO/working-directory requirements; updated YAML example instructions and SSE status; browser test isolation and valid blank example HTML. Existing MIT/theme licenses and safe loopback configuration are retained. No active local installation or live SQLite database was changed.

## Baseline and source review

The GitHub main snapshot was a4e97113ff714ccc07b7de8368af6d759d8d4508. Its recursive tree contained no SQLite database, WAL/SHM sidecars or compiled application. The isolated source copy was checked by Git blob SHA against the remote tree. No existing source mismatch was found; the remote LICENSE was fetched separately. Code changes were made by the Terra agent per AGENT.md and independently reviewed by the coordinator.

## Commands and results

Commands ran in /tmp/obs-overlay-publication/repo with a temporary GOCACHE, GOPROXY=off and GOTOOLCHAIN=local.

- Initial `go test -count=1 ./...`: exit 1 because the environment prevented the SSE integration test from opening a loopback listener. Other packages passed. This was an environment restriction, not an application failure.
- `go test -count=1 -v ./...`, rerun with local listener access: exit 0, all Go packages passed. Node-dependent panel/runtime checks were explicitly skipped because Node.js was unavailable.
- `go vet ./...`: exit 0.
- `go test -race -count=1 ./...` with local listener access: exit 0 for all packages.
- `GOFLAGS=-buildvcs=false python3 tests/browser_isolation.py` with local listener access: exit 0. Real application build/start, private working directory/configuration/database, separate loopback listeners and access-server restart passed. Browser interactions are stubbed; this is not DOM/browser acceptance.
- Python document checks: exit 0. README code fences balanced; local links resolve; English and Turkish command/config blocks are identical; AI notes appear at the top.
- Source scan confirmed no generated database or executable in the prepared source tree. Original configuration.yaml was not modified.

## Limits

Python Playwright is unavailable; real Chrome flows were not rerun even though Chrome exists. Node.js is unavailable; JavaScript behavioral checks were skipped. No physical phone/LAN, QR-camera scan, OBS Browser Source, public deployment or performance acceptance was performed. Earlier reports retain their historical scope. See also the agent's browser-isolation report.
