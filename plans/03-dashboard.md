# an agent creates, renames, moves and deletes projects, files and pages without touching the dashboard

## The ask

`docs/bridge.md` § 5 Dashboard and § 6 Pages: the work the official MCP
cannot do at all, which the `penpot` skill does today with Chrome clicks.

## Alternatives

### Decision 1: how are destructive tools guarded? (two-way door)

`delete_project`, `delete_file` and `delete_page` cannot be undone from the
API once Penpot purges them.

Searched 2026-10-19: Penpot source for `delete-file` soft delete
(`get-team-deleted-files`, `restore-deleted-team-files`); MCP spec tool
annotations (`destructiveHint`).

| Criterion | Weight | `confirm: true` argument + `destructiveHint` | Auto snapshot before delete | No guard |
| --- | ---: | ---: | ---: | ---: |
| Performance efficiency | 30 | 5 | 3 | 5 |
| Reliability | 40 | 4 | 5 | 1 |
| Security | 30 | 5 | 4 | 1 |
| **Weighted total** | **100** | **4.60** | **4.10** | **2.20** |

- `confirm` with `destructiveHint`: MCP clients prompt the user before a
  destructive tool runs; the argument forces the model to say it twice.
- Snapshot efficiency 3: one extra `create-file-snapshot` per delete, and
  it does not cover a deleted project.

**Chosen: `confirm: true` plus `destructiveHint`**, and a `restore_file`
tool over Penpot's soft-delete. Revisit if users lose work anyway.

## The decision

Tools: `create_project`, `rename_project`, `move_project`, `pin_project`,
`delete_project`, `create_file`, `rename_file`, `duplicate_file`,
`move_files`, `delete_file`, `restore_file`, `set_file_shared`,
`link_library`, `unlink_library`, `search_files`, `add_page`,
`rename_page`, `delete_page`, `reorder_pages`, `list_members`. Ids for new
projects and files come from Penpot's reply (2.19 rejects client ids).

Dependencies: none new. Standards: `docs/standards.md` (`make check`).

## Where it lands

| Path | New or edit | Phase | What changes | Firm? |
| --- | --- | --- | --- | --- |
| `internal/tools/dashboard.go` | new | 1 | project and file tools | expected |
| `internal/tools/pages.go` | new | 1 | page tools via `update-file` | expected |

*Reused, not rewritten:*

- `internal/penpot/client.go` (plan 01) for every RPC.
- `internal/penpot/writer.go` (plan 01) for page changes, which go through
  `update-file`.

## Security

Adds destructive actions on the user's account.

| Threat | Here | Handled by | Proved by |
| --- | --- | --- | --- |
| Elevation of privilege | an injected prompt deletes a project | `confirm: true` required, `destructiveHint` set, reply names what was removed | `TestDeleteNeedsConfirm` |
| Repudiation | nobody can tell what the agent deleted | every destructive call logged with id and name (no token) | `TestDeleteLogged` |

Checklist: OWASP Top 10 for LLM Applications 2025 (LLM06 excessive agency).

## Declined

- Auto snapshot before delete: lower on efficiency and misses project
  deletes (4.10).
- No guard: fails reliability and security (2.20).

## What the tests hold

- A delete without `confirm: true` sends no request.
- Create replies' ids are used, never generated ones.

## Repository knowledge

Read root `KNOWLEDGE.md` when present before each phase. Follow the global and
repository instructions for recording useful development findings.

## Phases

### Phase 1: the rough cut (needs you: try it)

- [x] `create_project`, `create_file`, `rename_file`, `add_page` and
  `rename_page` land (expected in `internal/tools/dashboard.go`,
  `pages.go`), pinned by `httptest` replays.
  Landed 2026-10-20: pinned by `TestCreateProjectUsesPenpotID`,
  `TestCreateFileReturnsPages`, `TestRenameFileDropsCache`,
  `TestAddAndRenamePage` and `TestDashboardRejectsBadInput`. The live check
  is `TestLiveDashboard`; it reuses the `penpot-mcp live` project and file
  on later runs, since deletes land in phase 2.
- [x] Check: `make check` passes, and `make live` creates a project and a
  file named `penpot-mcp live`, adds a page, and renames both.
  Landed 2026-10-20: `TestLiveDashboard` passed (8.4 s): it created the
  project and file, then ran every phase 1 and phase 2 tool against them.
- [ ] You try it: ask Claude to make a new project with a file and two
  named pages, and check them in the Penpot dashboard.

### Phase 2: the rest, with the delete guard

- [x] The remaining tools from `## The decision` land, deletes guarded
  (expected in `internal/tools/dashboard.go`, `internal/tools/pages.go`),
  pinned by `TestDeleteNeedsConfirm` and `TestDeleteLogged`; check:
  `make check` passes.
  Landed 2026-10-20: project and file tools in `dashboard.go`, page tools
  in `pages.go`, `list_members` in `team.go`. `confirm` is optional in the
  schema so a missing one gets the guard's own message, not a schema error.
  Deletes also refuse the team's default project and a file's last page.
  `restore_file` with no ids lists the team's deleted files. Destructive
  calls log id and name through `Deps.Log`. The reply shapes of the newer
  commands are checked by `TestLiveDashboard`, which now also cleans up the
  page it adds.

### Phase 3: open the PR

- [x] The branch pushed and its PR opened against `main` (or, if one is
  already open for this branch, its body brought up to date) with a
  Conventional Commit title for the whole change, a body naming this plan,
  what changed and why, and the checks run with their result. The PR's URL
  printed as the evidence. Opened, never merged.

Landed 2026-10-20: https://github.com/izam-mohammed/penpot-mcp/pull/3.
