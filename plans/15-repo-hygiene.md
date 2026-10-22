# the repository keeps its Go modules and actions current and says how to report a hole

## The ask

"have dependabot in all of my repos", "have proper security.md file for
all of my repos", and "proper licence for all of the repos ... which is a
product in nature ... even for private ones". Also: "for these changes,
have proper plans and complete that plans and raise prs. don't merge
anything to main". (22/10/2026)

Read against `origin/main` at `0aea8a7` on 22/10/2026:

- No `.github/dependabot.yml`, no Renovate config, no `SECURITY.md`, and no
  GitHub topics on the repository. `LICENSE` already holds the MIT text and
  the README ends with a licence section, so neither changes.
- Ecosystems found by their manifests: Go modules (`go.mod`, `go.sum` at the
  root) and GitHub Actions in `.github/workflows/check.yml` and
  `.github/workflows/release.yml`. No Dockerfile or other language manifest.
- The repository is public with private vulnerability reporting enabled; the
  latest release is `v0.1.6` (21/10/2026), shipped as binaries and the
  `izam-mohammed/tap/penpot-mcp` cask.
- Open PR #11 plans `plans/13-quality-gates.md` under `PLAN.md` Phase 9 and
  10; this plan takes the next free numbers, `15` and `PLAN.md` Phase 11, so
  they merge without renumbering.

No built snapshot is written for this plan: it adds repository
configuration and documents only, and changes nothing that is built.

## Alternatives

### Decision 1: which bot keeps dependencies current (two-way door)

Searched 22/10/2026: "Renovate vs Dependabot comparison grouping
cooldown minimumReleaseAge"; read the Dependabot options reference
(docs.github.com, `cooldown`, `groups`, the `package-ecosystem` values),
the supported ecosystems page, and secondary comparisons (vulert.com,
appsecsanta.com).

| Must | Dependabot (user's pick) | Renovate |
| --- | --- | --- |
| Updates every package manager used here | yes | yes |
| Runs with no extra app or service to install | yes (built into GitHub) | no (GitHub app or self-hosted runner) |
| **Survives** | **yes** | no |

| Criterion | Weight | Dependabot |
| --- | ---: | ---: |
| Performance efficiency | 40 | 4 |
| Reliability | 30 | 4 |
| Consistency with the other repositories | 30 | 5 |
| **Weighted total** | **100** | **4.30** |

- Efficiency 4: one grouped PR per ecosystem a week, with a 7 day
  cooldown, keeps the PR count low.
- Consistency 5: `decisionsmith` already runs the same file.

**Chosen: Dependabot.** It is the user's pick and the only option needing
nothing installed. Revisit if a package manager appears that Dependabot cannot
update.

## The decision

Dependabot (Decision 1) watches `gomod` and `github-actions` at `/`, weekly,
with a 7 day cooldown, one group per ecosystem, prefixed `chore(deps)` for
Go and `ci(deps)` for actions, in the shape of
`decisionsmith/.github/dependabot.yml`. `SECURITY.md` says only the latest
release gets fixes, points reports at GitHub private vulnerability
reporting, and scopes it to the `penpot-mcp` binary (stdio and `--http`
server), its bundled skills, and the check and release workflows. The
existing MIT `LICENSE` and README section stay as they are. Topics are set
as a repository setting.

New dependencies or permissions: none. Dependabot is a GitHub feature.

Standards: `docs/standards.md` (`make check`). Nothing here is code, so each
phase's check is the YAML parse and the file read through.

Security: no code, input, secret or permission changes; the change adds
an update bot config and a reporting policy.

## Where it lands

| Path | New or edit | Phase | What changes | Firm? |
| --- | --- | --- | --- | --- |
| `.github/dependabot.yml` | new | 2 | weekly grouped bumps per ecosystem | fixed |
| `SECURITY.md` | new | 3 | the reporting policy | fixed |

*Reused, not rewritten:*

- `decisionsmith/.github/dependabot.yml` and `decisionsmith/SECURITY.md`:
  the shape both new files follow.

## Phases

### Phase 1: topics

- [x] The repository carries the topics `go`, `mcp`, `mcp-server`,
  `model-context-protocol`, `penpot`, `design-tools`, `claude-code` and
  `cli` (set with `gh repo edit --add-topic`; it had none); check: `gh repo
  view --json repositoryTopics` lists them.

### Phase 2: Dependabot

- [x] `.github/dependabot.yml` bumps Go modules and GitHub Actions weekly,
  one group each, with a 7 day cooldown and `chore(deps)` / `ci(deps)`
  prefixes; check: the file parses as YAML and names only documented
  ecosystem values.

### Phase 3: security policy

- [x] `SECURITY.md` says only the latest release gets fixes, links GitHub
  private vulnerability reporting for this public repository, says what
  happens next and names this repository's scope, with no invented email or
  response time; check: read through against the README.

### Phase 4: open the PR

- [ ] The branch pushed and its PR opened against `main` (or, if one is
  already open for this branch, its body brought up to date) with the title
  `chore: dependabot and security policy`, a body naming this plan, what
  changed and why, and the checks run with their result. The PR's URL
  printed as the evidence. Opened, never merged.
