# make check holds coverage at 90 % and every PR passes the shared security gate

## The ask

Every repository should have proper lint and format, tests, a test
coverage floor of 90 % (target 95 %) and security checks (dependency audit,
secret scan, static analysis), all run by one local check command and by CI
on every pull request, reusing the shared security gate in
`izam-mohammed/.github` rather than a parallel one.

Built so far: `docs/built/BUILT-2026-10-22-1826.md`.

What it means here: most of it exists. `make check` runs golangci-lint
fmt and lint, the build, `go test -race`, govulncheck and gitleaks, and
`check.yml` runs it on every push and PR (`docs/standards.md`). On
22/10/2026 fmt, lint, govulncheck and the tests were all clean. Three gaps:
coverage is 64.3 % of 4,401 statements with no floor; the shared gate is not
called, so there is no semgrep, osv-scanner or zizmor; and the workflows pin
actions by tag (`actions/checkout@v4`), which the gate's conventions
forbid. Reaching 90 % means about 1,130 more statements covered, most of
them in `internal/tools` (2,948 statements, 63.1 %) and `internal/penpot`
(693, 61.5 %); `cmd/penpot-mcp` has no tests (92 statements, 0 %).

## Alternatives

### Decision 1: how is the floor held while coverage climbs? (two-way door)

A floor of 90 % today fails every PR until the tests land; no floor lets it
drop meanwhile.

Searched 2026-10-22: "go test coverage threshold enforce CI
go-test-coverage vladopajic vs codecov"; the go-test-coverage project page
(per-package thresholds, server-less), Codecov's Go guide (hosted, needs an
account), `mac-notch`'s `scripts/coverage-floor` (one number, `make
coverage`).

| Must | Fixed 90 % now | A ratchet floor file raised each phase to 90 % | go-test-coverage per package | Codecov |
| --- | --- | --- | --- | --- |
| Runs locally with no account | yes | yes | yes | no (hosted) |
| **Survives** | **yes** | **yes** | **yes** | no |

| Criterion | Weight | Fixed 90 % now | Ratchet floor file | go-test-coverage |
| --- | ---: | ---: | ---: | ---: |
| Reliability | 40 | 5 | 4 | 5 |
| Fit to the ask | 35 | 2 | 5 | 4 |
| Compatibility | 25 | 2 | 5 | 3 |
| **Weighted total** | **100** | **3.20** | **4.60** | **4.15** |

Performance efficiency is left out: all three read the profile `go test`
already writes, so it cannot differ.

- Fixed 90 % fit 2 and compatibility 2: `make check` is red on `main` and
  every unrelated PR until the last phase.
- Ratchet reliability 4: one total can hide a weak package; the per-package
  table is printed lowest first so it is seen. Compatibility 5: the same
  `scripts/coverage-floor` pattern as `mac-notch`.
- go-test-coverage compatibility 3: one more tool for `make tools` to pin.

**Chosen: a ratchet floor file raised each phase to 90 %.** It never
drops, and every phase leaves the check green. Revisit once at 95 %: then
per-package floors may be worth the tool.

### Decision 2: how do the API-facing packages get tested? (two-way door)

`internal/tools` and `internal/penpot` call Penpot; the standards say
Penpot is never called from unit tests and no token is in CI.

Searched 2026-10-22: "go-vcr vs httptest recorded responses Go testing
HTTP client"; go-vcr's README (a recording `http.Transport`, cassettes,
custom matchers), a write-up noting cassettes make error paths harder to
test; the repository's `internal/penpot/fake_test.go`.

| Must | Baseline: extend the `httptest` fakes and `testdata/*.json` | go-vcr cassettes | Live tests in CI |
| --- | --- | --- | --- |
| No Penpot token in CI | yes | yes (recorded once, by hand) | no |
| **Survives** | **yes** | **yes** | no |

| Criterion | Weight | Extend the fakes | go-vcr cassettes |
| --- | ---: | ---: | ---: |
| Performance efficiency | 40 | 5 | 4 |
| Reliability | 30 | 4 | 4 |
| Compatibility | 30 | 5 | 3 |
| **Weighted total** | **100** | **4.70** | **3.70** |

- Fakes efficiency 5: in-process handlers, no file matching per request;
  go-vcr 4, a matcher over each cassette.
- Fakes compatibility 5: `fake_test.go` and the recorded replies already
  exist; go-vcr 3, a new dependency and a second fixture format.
- Both reliability 4: fixtures drift from the live API; `make live` stays
  the check against the real one.

**Chosen: extend the `httptest` fakes.** Revisit if the fixtures drift
from Penpot often enough that `make live` keeps failing.

### Decision 3: how does the shared gate reach this repository? (two-way door)

`make check` already runs govulncheck and gitleaks; the ask wants the
shared gate reused, not a parallel one.

Searched 2026-10-22: "GitHub reusable workflow security scan gitleaks
osv-scanner semgrep zizmor vs CodeQL default setup"; the gate's
`README.md` and `security-scan.yml` (inputs `ecosystem`, `go-version`;
stage 1 blocks), `claim-your-code`'s `security.yml` calling it.

| Criterion | Baseline: `make check` only | `security.yml` calling the shared gate, `make check` kept | CodeQL default setup added |
| --- | --- | --- | --- |
| Performance efficiency | one job | four parallel jobs on PRs and weekly | a CodeQL build per PR |
| Fit to the ask | no SAST, no workflow audit | secrets, deps, SAST, workflows | SAST only |
| Compatibility | as today | the same caller file as `claim-your-code` | a second SAST beside semgrep |

**Chosen: `security.yml` calling the shared gate, `make check` kept**,
with `ecosystem: go` and `go-version: '1.26'`, pinned to the gate's commit
SHA. Revisit if the gate gains a blocking SAST stage that fails here.

## The decision

`make cover` writes the profile from `go test -race -coverprofile`, prints
the per-package table lowest first and fails under the number in
`scripts/coverage-floor`; `make check` runs it in place of `test`
(decision 1). The floor starts at 64, the measured total rounded down,
rises with each phase's tests and ends at 90, with 95 the target. New tests
extend the `httptest` fakes and `testdata/*.json` (decision 2); `main.go`'s
`run` is tested by calling it with fake args and environment.
`.github/workflows/security.yml` calls the shared gate (decision 3). Every
action in `check.yml`, `release.yml` and the new file is pinned to a full
commit SHA with the version in a comment, and `permissions: {}` sits at the
top of each with grants per job.

New dependencies: none in the binary or `go.mod`. New permission:
`pull-requests: write` for the gate's report comment, on its job only.

Standards: `docs/standards.md` (format, lint, build, tests, govulncheck and
gitleaks through `make check`; this plan adds the coverage floor and the
shared gate to it).

## Where it lands

| Path | New or edit | Phase | What changes | Firm? |
| --- | --- | --- | --- | --- |
| `scripts/coverage-floor` | new | 1 | the floor, one number | fixed |
| `Makefile` | edit | 1 | `cover` target, `check` calls it | reuse |
| `docs/standards.md` | edit | 1 | coverage row, floor, gate | reuse |
| `cmd/penpot-mcp/main_test.go` | new | 2 | `run` tested | expected |
| `internal/penpot/*_test.go` | edit | 2 | writer and client paths | reuse |
| `internal/tools/*_test.go` | edit | 3 | tool handlers over the fakes | reuse |
| `.github/workflows/security.yml` | new | 4 | the shared gate | expected |
| `.github/workflows/check.yml` | edit | 4 | SHA pins, per-job permissions | reuse |
| `.github/workflows/release.yml` | edit | 4 | SHA pins | reuse |

*Reused, not rewritten:*

- `internal/penpot/fake_test.go` and `internal/*/testdata/*.json`: the fake
  Penpot every new test runs against.
- `Makefile` `check`, `tools`: extended, not replaced.
- `izam-mohammed/.github/.github/workflows/security-scan.yml`: called,
  not copied.
- `mac-notch/scripts/coverage-floor`: the one-number floor pattern.

## Security

The change touches CI permissions and adds a workflow; it adds nothing to
the binary.

| Threat | Here | Handled by | Proved by |
| --- | --- | --- | --- |
| Tampering: an action replaced under its tag | `actions/checkout@v4`, `goreleaser-action@v6` and others pinned by tag | full SHA pins with the version in a comment | the gate's zizmor stage reports no unpinned action |
| Elevation of privilege: a workflow with more than it needs | `release.yml` has `contents: write` at the top | `permissions: {}` at the top, grants per job | zizmor clean on the PR |
| Information disclosure: a token in test data | new fixtures recorded from real replies | fixtures written by hand or scrubbed; gitleaks in `make check` and stage 1 of the gate | `make check` and the gate's secrets stage green |
| Information disclosure: the token in logs | the Penpot access token | the existing absence test kept in the suite | `go test ./internal/...` |

Secrets stay as named in `CLAUDE.md`; no value is read or written. The
checklist followed: the shared gate's workflow conventions and SLSA build
level 1 for releases, unchanged.

## Edge cases

- Live tests (`-tags live`) are not in the profile; coverage counts only
  what runs without a token.
- A test that needs real font metrics uses the embedded fonts
  (`internal/fonts`), never the system's.

## Declined

- Fixed 90 % now: lost on fit and compatibility (red until the end).
- go-test-coverage: lost on compatibility for no reliability gain at this
  stage.
- Codecov: failed the must.
- go-vcr cassettes: lost on efficiency and compatibility.
- Live tests in CI: failed the must (a Penpot token in CI).
- `make check` only, and CodeQL: no SAST or workflow audit, or a second
  SAST.

## What the tests hold

Each phase's new tests pin the tool handlers' happy path, the Penpot error
mapping and the refusals, against the fakes. The floor file holds that
coverage never drops. The gate holds secrets (blocking), dependencies,
SAST and workflow findings (reported).

## Repository knowledge

Read root `KNOWLEDGE.md` when present before each phase. Follow the global and
repository instructions for recording useful development findings.

## Phases

### Phase 1: the rough cut: a floor that never drops (needs you: try it)

- [ ] `make cover` prints the per-package table lowest first and fails
  under `scripts/coverage-floor` (64), and `make check` runs it in place
  of `test` (expected in `Makefile`), pinned by the floor failing when set
  to 100 in a scratch copy.
- [ ] `docs/standards.md` gains the coverage row, the floor and the target.
- [ ] Check: `make check` passes, and `make cover` prints 64 % or more.
- [ ] You try it: run `make check` and say whether the coverage table is
  the view you want on every run.

### Phase 2: the client, the small packages and main

- [ ] `internal/penpot`, `internal/httpserve`, `internal/transit`,
  `internal/mcproute` and `internal/fonts` reach 90 % each against the
  fakes, and `cmd/penpot-mcp`'s `run` is tested with fake args (expected
  in `cmd/penpot-mcp/main_test.go`); `scripts/coverage-floor` raised to
  the new total, rounded down.
- [ ] Check: `make check` passes at the new floor.

### Phase 3: the tools

- [ ] `internal/tools` handlers tested against the fakes until the total
  is at least 90 %, lowest files first, and `scripts/coverage-floor` set
  to 90.
- [ ] Check: `make check` passes at 90.

### Phase 4: the shared gate and pinned workflows

- [ ] `.github/workflows/security.yml` calls the shared gate with
  `ecosystem: go`, `go-version: '1.26'`, pinned to its commit SHA, and
  every action in `check.yml` and `release.yml` is pinned by SHA with
  `permissions: {}` at the top and grants per job.
- [ ] `docs/standards.md`'s CI and Decided sections name the gate.
- [ ] Check: `make check` passes, and the PR's Security run reports stage
  1 green and zizmor with no unpinned action.

### Phase 5: open the PR

- [ ] The branch pushed and its PR opened against the branch it was cut from
  (or, if one is already open for this branch, its body brought up to date)
  with a Conventional Commit title for the whole change, a body naming this
  plan, what changed and why, and the checks run with their result. The PR's
  URL printed as the evidence. Opened, never merged.
