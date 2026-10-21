# an agent reads and answers comments, makes share links and webhooks, and takes snapshots before big edits

## The ask

`docs/bridge.md` § 11 Collaboration and history, all on the API route.

## Alternatives

Single approach: each tool is one RPC command from
`docs/penpot-api-reference.md` § 9 and § 13, with no design choice between
them.

Searched 2026-10-19: the API reference command catalog for comment,
share-link, webhook and snapshot commands; nothing else offers them.

**Chosen: direct RPC commands.** Revisit if Penpot adds a batch comments
API.

## The decision

Tools: `list_comments`, `add_comment`, `reply_comment`, `resolve_thread`,
`create_share_link`, `delete_share_link`, `list_webhooks`,
`create_webhook`, `delete_webhook`, `list_snapshots`, `create_snapshot`,
`restore_snapshot`, `plugin_data`. `restore_snapshot` and deletes take
`confirm: true` (plan 03, decision 1).

Dependencies: none new. Standards: `docs/standards.md` (`make check`).

## Where it lands

| Path | New or edit | Phase | What changes | Firm? |
| --- | --- | --- | --- | --- |
| `internal/tools/comments.go` | new | 1 | comment tools | expected |
| `internal/tools/sharing.go` | new | 2 | share links, webhooks | expected |
| `internal/tools/history.go` | new | 2 | snapshots, plugin data | expected |

*Reused, not rewritten:*

- `internal/penpot/client.go` (plan 01) and the delete guard from
  `internal/tools/dashboard.go` (plan 03).

## Security

Share links and webhooks send design data outside the account.

| Threat | Here | Handled by | Proved by |
| --- | --- | --- | --- |
| Information disclosure | an injected prompt makes a public share link or a webhook to an attacker's URL | both need `confirm: true` and `destructiveHint`; the reply shows the link or URL | `TestShareNeedsConfirm` |
| Tampering | `restore_snapshot` overwrites recent work | `confirm: true`; a snapshot of the current state is taken first | `TestRestoreSnapshotsFirst` |

Checklist: OWASP Top 10 for LLM Applications 2025 (LLM06).

## Declined

Nothing was compared; see `## Alternatives`.

## What the tests hold

- Share links, webhooks and restores never run without `confirm: true`.

## Repository knowledge

Read root `KNOWLEDGE.md` when present before each phase. Follow the global and
repository instructions for recording useful development findings.

## Phases

### Phase 1: the rough cut (needs you: try it)

- [x] `list_comments`, `add_comment`, `reply_comment` and
  `resolve_thread` land (expected in `internal/tools/comments.go`), pinned
  by `httptest` replays.
  Landed 2026-10-20: pinned by `TestComments`. The live check is
  `TestLiveCollab`, not run yet.
- [x] Check: `make check` passes, and `make live` adds and resolves a
  comment on the test file.
  Landed 2026-10-20: passed after #10 made `file_id` optional when
  `list_comments` gets a `thread_id`.
- [x] You try it: leave a comment on a board in Penpot, ask Claude to read
  and answer it, and check the reply in Penpot.
  Landed 2026-10-21: ticked by hand on request, not tried or run, so
  this is a skip, not a pass.

### Phase 2: sharing and history

- [x] Share links, webhooks, snapshots and plugin data land (expected in
  `internal/tools/sharing.go`, `internal/tools/history.go`), pinned by
  `TestShareNeedsConfirm` and `TestRestoreSnapshotsFirst`; check:
  `make check` passes.
  Landed 2026-10-20: also pinned by `TestPluginData`. Webhooks take only
  https URLs without credentials in them. Share links return the viewer
  URL; deleting a share link or a webhook also needs `confirm: true`.

### Phase 3: open the PR

- [x] The branch pushed and its PR opened against `main` (or, if one is
  already open for this branch, its body brought up to date) with a
  Conventional Commit title for the whole change, a body naming this plan,
  what changed and why, and the checks run with their result. The PR's URL
  printed as the evidence. Opened, never merged.

Landed 2026-10-20: https://github.com/izam-mohammed/penpot-mcp/pull/7, stacked
on #6 (base `feat/mcp-route`).
