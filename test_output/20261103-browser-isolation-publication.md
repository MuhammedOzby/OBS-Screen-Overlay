# Browser test isolation — 3 November 2026

Scope: isolate the two optional browser tests with temporary working directories, configuration, templates, ports and SQLite databases; remove the unexpected field from the blank example.

- `python -m py_compile tests/browser_access.py tests/browser_refactor.py`: passed.
- `GOFLAGS=-buildvcs=false GOCACHE=/tmp/obs-overlay-publication/isolation-gocache GOPROXY=off GOTOOLCHAIN=local python tests/browser_isolation.py`: passed. This standard-library harness uses the real built application and checks temporary configuration and listeners; access server restart also passed. Browser interactions are stubbed, so this is not a browser acceptance test. `-buildvcs=false` was needed for this temporary snapshot's Git metadata.
- `python tests/browser_access.py` and `python tests/browser_refactor.py`: blocked before startup because Python Playwright is not installed. Chrome exists; neither browser acceptance flow was run.
- An initial isolated startup attempt was blocked by the environment's socket restriction; the successful isolation run used approved local listener access. The first harness attempt failed due to its own build-process interception and was corrected before the successful run.
