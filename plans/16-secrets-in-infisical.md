# the live checks take their secrets from Infisical, not a laptop-only .env

> Amended 17/11/2026: removed as irrelevant. This repository is public and
> keeps no active secrets, so Infisical is not needed. Everything this plan
> added is gone; see the section at the foot. Boxes stay as they were.

## The ask

"Have these proper and maintain in infisical ... create proper plan files
... properly move all secrets to infisical ... get the secrets from .env
without reading and if the env only in env example without value, create
that key with empty value in infisical also", then raise a PR and merge it
when CI passes.

Read against `origin/main` at `1c07cbe` on 15/11/2026:

- `.env.example` names six variables: `PENPOT_BASE_URL`,
  `PENPOT_ACCESS_TOKEN`, `PENPOT_TEST_FILE_ID`, `PENPOT_EMAIL`,
  `PENPOT_PASSWORD`, `PENPOT_MCP_URL`.
- The git-ignored local `.env` holds two of them (names checked with
  `sed -E 's/=.*//'`, values never read): `PENPOT_ACCESS_TOKEN` and
  `PENPOT_MCP_URL`. The other four have no local value anywhere.
- Readers: the server (`cmd/penpot-mcp/main.go`, `internal/penpot/`,
  `internal/tools/`, `internal/mcproute/`), the live and unit tests, the
  proof scripts in `scripts/prove/` and the `live` and `live-mcp` targets in
  the `Makefile`. `make check` and CI need none of them.
- There is no start script: users run the released binary with `-e` values
  in their MCP client. The only things a developer starts with secrets are
  `make live`, `make live-mcp` and the proof scripts.
- No Infisical project existed for this repository.

No built snapshot is written for this plan: it moves where local secrets
come from and adds a wrapper script, and changes nothing that is built.

## Constraints

- Infisical, through its CLI, one project per repository, named after it,
  with the bare GitHub URL as its description.
- No value read, printed or put on a command line; `.env` stays in place.
- `make check` and the release workflow keep working with no secrets.

## Alternatives

Infisical was already chosen across the owner's repositories
(`money-logs/plans/18-secrets-in-infisical.md`, Decision 1), so only the
wiring is decided here.

### Decision 1: how the live checks get the secrets (two-way door)

Searched 15/11/2026: `infisical run --help` (CLI 0.45.8) and the
`infisical-secrets` skill's `assets/dev-wrapper.sh`.

| Must | Baseline: export `.env` by hand | `infisical export` into `.env` | `scripts/dev.sh` around the command |
| --- | --- | --- | --- |
| No plaintext copy written on disk | yes | no | yes |
| Works without the CLI or offline | yes | no | yes (falls back to `.env`) |
| **Survives** | yes | no | **yes** |

| Criterion | Weight | Export by hand | `scripts/dev.sh` |
| --- | ---: | ---: | ---: |
| Performance efficiency | 40 | 5 | 4 |
| Reliability | 30 | 2 | 5 |
| Security and control | 30 | 2 | 5 |
| **Weighted total** | **100** | **3.20** | **4.60** |

- Efficiency: `infisical run` adds one network fetch per command (about 2 s
  on money-logs); exporting by hand adds nothing at run time.
- Reliability: by hand depends on a `.env` that exists on one laptop only.
- Security: `scripts/dev.sh` keeps values in the process environment only.

**Chosen: `scripts/dev.sh <command>`**, which re-runs itself under
`infisical run --env=dev` when the checkout is linked and the CLI is
installed, sources `.env` otherwise, then `.env.local` on top. `make live`
and `make live-mcp` call it. Revisit if a real start script appears.

## The decision

Create the Infisical project `penpot-mcp` with the description
`https://github.com/izam-mohammed/penpot-mcp`, link it with both
`.infisical.yaml` and `.infisical.json`, push `.env` into `dev` with
`--file`, and create the four names that exist only in `.env.example` with
empty values through the API (the CLI refuses an empty value). Every secret
is tagged by its readers and kind (`scripts/infisical/tags`). All names
live at `/`: there is one app and no sub-app `.env.example`.

| Name | Value | Tags |
| --- | --- | --- |
| `PENPOT_ACCESS_TOKEN` | from `.env` | `backend,tests,scripts,api-key` |
| `PENPOT_MCP_URL` | from `.env` | `backend,tests,scripts,api-key` |
| `PENPOT_BASE_URL` | empty | `backend,tests,scripts,config` |
| `PENPOT_TEST_FILE_ID` | empty | `scripts,config` |
| `PENPOT_EMAIL` | empty | `backend,tests,scripts,config` |
| `PENPOT_PASSWORD` | empty | `backend,tests,scripts,api-key` |

An empty `PENPOT_BASE_URL` falls back to `https://design.penpot.app`
(`penpot.NewClient`), so injecting it empty changes nothing.

Dependencies: the Infisical CLI and `jq` on a developer machine, the host
`app.infisical.com`. No Go module, no CI change.

Standards: `docs/standards.md` (`make check`, which includes gitleaks).

## Integrations

| Integration | Local docs | Proof script | Env vars |
| --- | --- | --- | --- |
| Infisical CLI | the `infisical-secrets` skill, `infisical run --help` | `scripts/prove/infisical.sh` | none |

## Where it lands

| Path | New or edit | Phase | What changes | Firm? |
| --- | --- | --- | --- | --- |
| `.infisical.yaml`, `.infisical.json` | new | 1 | the project ID, committed (no secret) | fixed |
| `.gitignore` | edit | 1 | `.env.*` except `.env.example`, `.env.local`, `.secrets/`, `*.pem` | reuse |
| `scripts/infisical/tags` | new | 1 | name to tags map, names only | expected |
| `scripts/prove/infisical.sh` | new | 1 | link and every `.env.example` name present, PASS | expected |
| `scripts/dev.sh` | new | 2 | runs a command with the secrets loaded | expected |
| `Makefile` | edit | 2 | `live` and `live-mcp` go through `scripts/dev.sh` | reuse |
| `.env.example`, `README.md` | edit | 2 | values live in Infisical; how a new machine starts | reuse |

*Reused, not rewritten:*

- `scripts/prove/`: the existing proof folder; the new proof follows the
  skill's `assets/prove-infisical.sh`.
- `Makefile` `live` and `live-mcp`: the same tests, wrapped.
- Searched for an existing start or env-loading script: none in `scripts/`
  or the `Makefile`, so `scripts/dev.sh` is new.

## Security

What is protected: the Penpot access token, the official MCP URL (it
carries a key) and, once filled, the Penpot password. Who can reach them:
members of the Infisical project (only the owner), through an Infisical
login.

| Threat | Here | Handled by | Proved by |
| --- | --- | --- | --- |
| Information disclosure | a value printed while moving it | push with `--file`, names through `jq`, hash compare | `verify.sh` and the proof print names only |
| Information disclosure | a value on a command line | values go file to file; empties via an API body with `secretValue: ""` | the push commands in Phase 1 |
| Information disclosure | a value committed | link files hold the project ID; `.env*` ignored | `make check` runs `gitleaks git` |
| Tampering | a value overwritten | `create` only, into a new project | `verify.sh` PASS for both `.env` values |

## Rollback

`.env` stays. Delete `.infisical.yaml` and `.infisical.json` and
`scripts/dev.sh` sources `.env` as before. The Infisical project can be
deleted from the web app.

## Declined

- Exporting `.env` by hand: the baseline, lost with the laptop.
- `infisical export` into `.env`: a plaintext copy on disk again.
- Folders per part (`/server`, `/scripts`): one app, six names; tags do
  that job.

## What the tests hold

`sh scripts/prove/infisical.sh` holds that the checkout is linked, both
link files name the same project and every `.env.example` name is in `dev`.
`verify.sh --env dev --file .env` holds that the two pushed values match
byte for byte. `make check` holds that nothing secret is committed and the
build needs no secret.

## Follow-ups

- `PENPOT_BASE_URL`, `PENPOT_TEST_FILE_ID`, `PENPOT_EMAIL` and
  `PENPOT_PASSWORD` are empty in Infisical; fill them in the web app when
  the export and move proofs are next run.
- `PENPOT_MCP_HTTP_TOKEN`, `PENPOT_PROFILES` and the per-profile token
  variables are read by the server but are user-side settings, not in
  `.env.example`, so not moved.
- All names are UPPER_SNAKE_CASE; nothing to flag.
- Released binaries and CI need no secret; a machine identity is only
  needed if a live check ever runs in CI.

## Phases

### Phase 1: the project and the secrets

- [x] The Infisical project `penpot-mcp` created and linked with
  `setup.sh`, description `https://github.com/izam-mohammed/penpot-mcp`,
  `.infisical.yaml` and `.infisical.json` naming the same project, and
  `.gitignore` covering the env and key files.
  Landed 15/11/2026: project `1b1ad76a-f3ad-4c05-911d-840fd91c3d10`.
- [x] `.env` pushed into `dev` with `push.sh --file .env --tags-file
  scripts/infisical/tags`, no value read or printed.
- [x] The four names only in `.env.example` created in `dev` with empty
  values and their tags.
  Landed 15/11/2026: the CLI refuses an empty value ("has no value"), so
  they went through `POST /api/v3/secrets/raw/<NAME>` with the token held
  in a variable; the `config` tag was created on the way.
- [x] `scripts/prove/infisical.sh` lands and prints PASS.
- [x] Check: `verify.sh --env dev --file .env` PASS (2 of 2 match), the
  proof PASS with 6 secrets, `gitleaks` no findings.

### Phase 2: the rough cut (needs you: try it)

- [x] `scripts/dev.sh` runs a command under `infisical run --env=dev` when
  linked, `.env` otherwise, `.env.local` on top; `make live` and `make
  live-mcp` go through it; `.env.example` and the README say how a new
  machine starts.
- [x] Check: `make check` passes, and `sh scripts/dev.sh` injects 6
  secrets (names checked set, values never printed).
  Landed 15/11/2026: fmt, build, tests and gitleaks pass locally; the
  local golangci-lint panics on `main` too (built with go1.26, a module
  wants go1.27), so lint is held by CI. `make live` is wrapped but not run
  here, since it draws into the real test file.
- [ ] You try it: move `.env` aside, run `sh scripts/dev.sh sh
  scripts/prove/penpot-api.sh` (and `make live` if you like), and say
  whether it passes. Then say whether to delete `.env`; it stays until
  you do.

### Phase 3: open the PR

- [x] The branch pushed and its PR opened against `main` with a
  Conventional Commit title, a body naming this plan, the names moved
  (never values) and the checks run. Merged once CI is green.
  Landed 15/11/2026: https://github.com/izam-mohammed/penpot-mcp/pull/17
  Left open: CI's `govulncheck` fails on GO-2026-6859 in `net/http`
  (go1.26.10, fixed in go1.26.11), new since `main` last went green and not
  from this change. Merge after the toolchain bump.

## Amended, 2026-11-17: removed, irrelevant

The repository is public and holds no active secrets, so a secrets manager
adds a dependency and a setup step for nothing. Removed: `.infisical.json`,
`.infisical.yaml`, `scripts/dev.sh`, `scripts/infisical/`,
`scripts/prove/infisical.sh`, the README's "hacking on it" section, the
`Makefile` wrapper on `live` and `live-mcp` and the Infisical header in
`.env.example`. The `Makefile` and `.env.example` are back to how they were
before this plan. The extra `.gitignore` patterns for `.env.*`, `.secrets/`
and `*.pem` stay. The live checks read `PENPOT_ACCESS_TOKEN` and the other
names from the shell, as they did before. The Infisical project
`penpot-mcp` is deleted too.
