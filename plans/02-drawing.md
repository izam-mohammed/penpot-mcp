# an agent draws whole screens (text, images, paths, groups, layouts) in one call, and edits or removes what it drew

## The ask

`docs/bridge.md` § 7 Drawing: every drawing tool on the API route. The rough
cut in `plans/01-first-version.md` draws a board and a rect; this plan adds
the rest of the shape set, edits, moves and deletes, and the declarative
`draw` tool that builds a whole screen in one `update-file` batch.

## Alternatives

### Decision 1: how is text sized without a browser? (two-way door)

The backend does not measure text (`docs/external/penpot-backend-api.md`
§ Gotchas), so a text shape's width and height are ours to compute.

Searched 2026-10-19: "golang.org/x/image/font/opentype measure text width
go"; the x/image `font.Drawer.MeasureString` docs, go-opentype/opentype.

| Criterion | Weight | Fixed ratio (0.55 em per char) | Real font metrics (x/image/opentype) |
| --- | ---: | ---: | ---: |
| Performance efficiency | 40 | 5 | 4 |
| Fit to the ask | 40 | 2 | 5 |
| Maintainability of the result | 20 | 4 | 3 |
| **Weighted total** | **100** | **3.60** | **4.20** |

- Metrics efficiency 4: parsing a font once per family at start, then
  `MeasureString` per text, microseconds per call; a few hundred KB per
  embedded font.
- Fit: a fixed ratio is off by 20% or more for narrow and wide faces;
  real advances match the browser to within kerning.

**Chosen: real font metrics** with `golang.org/x/image/font/opentype`,
fonts fetched from Penpot's font list on first use and cached on disk,
falling back to the fixed ratio for a font that cannot be fetched. Revisit
if exported text and Penpot's own layout differ by more than 5%.

### Decision 2: how are shapes moved and reordered? (two-way door)

`mov-objects` and `reorder-children` over JSON probably match nothing
(`docs/penpot-api-reference.md` § 5.5).

Searched 2026-10-19: Penpot source `changes.cljc` for `:mov-objects`
decoding; the API reference § 5.5.

| Criterion | Weight | Delete and re-add, same id | `mov-objects` in transit |
| --- | ---: | ---: | ---: |
| Performance efficiency | 40 | 4 | 5 |
| Reliability | 40 | 4 | 5 |
| Fit to the ask | 20 | 4 | 5 |
| **Weighted total** | **100** | **4.00** | **5.00** |

- Transit reliability 5: `scripts/prove/penpot-move.sh` on 2026-10-19 sent
  `mov-objects` both ways to design.penpot.app (2.18): JSON answered 200 and
  moved nothing; transit answered 200 and moved the rect into the new board.
- Delete and re-add efficiency 4: two changes per move and the shape's
  children have to be re-added too.

**Chosen: `mov-objects` in transit** for moves and reorders, with a
minimal transit encoder for those changes only (expected in
`internal/transit/`); JSON for everything else. Revisit if Penpot decodes
`mov-objects` from JSON.

## The decision

All tools from `docs/bridge.md` § 7 on the API route: `create_ellipse`
(type `circle`), `create_path` (from SVG `d`), `create_text` (decision 1),
`create_image` (upload then a `fillImage` rect), `create_group`,
`create_bool`, `update_shape` (`mod-obj` with `assign`), `move_shape`,
`reparent_shape` (decision 2), `delete_shapes`, `draw` (a nested tree in
one batch), `apply_changes` (raw `changes[]`, the escape hatch), and
`set_layout` (flex and grid attributes). Geometry gains rotation.

New dependency: `golang.org/x/image` (font parsing). Network host: the
font file URLs Penpot's font list gives (Google Fonts CDN). No new secret.

Standards: `docs/standards.md` (`make check`).

## Where it lands

| Path | New or edit | Phase | What changes | Firm? |
| --- | --- | --- | --- | --- |
| `internal/penpot/geometry.go` | edit | 2 | rotation, paths' bounds | expected |
| `internal/penpot/shapes.go` | new | 1 | builders per shape type with defaults | expected |
| `internal/penpot/text.go` | new | 1 | text content tree and measuring | expected |
| `internal/tools/draw.go` | new | 1 | the `draw` tree tool | expected |
| `internal/tools/edit.go` | new | 2 | update, move, delete | expected |

*Reused, not rewritten:*

- `internal/penpot/writer.go` (plan 01): every new tool writes through its
  queue and revn cache.
- `internal/penpot/geometry.go` (plan 01): extended with rotation, not
  copied.

## Security

Adds outside input with more shape (SVG path strings, image files from
local paths, a raw `changes[]` escape hatch) and destructive tools.

| Threat | Here | Handled by | Proved by |
| --- | --- | --- | --- |
| Tampering | a malformed SVG `d` or a huge `draw` tree | path parser with a command cap; tree depth and node caps | `TestPathRejectsGarbage`, `TestDrawCaps` |
| Information disclosure | `create_image` reads any local path the model names | only regular files with an image content type, size capped, path printed back | `TestImageRejectsNonImage` |
| Elevation of privilege | an injected prompt deletes a board | `delete_shapes` returns the names it removed; refuses the root frame | `TestDeleteRefusesRoot` |
| Tampering | `apply_changes` sends arbitrary changes | same file guard and validation as every write; marked last resort | `TestApplyChangesValidates` |

Data and secrets: unchanged from plan 01. Logged: shape counts, never
image bytes. Checklist: OWASP Top 10 for LLM Applications 2025 (LLM06).

## Declined

- Fixed text ratio: lost on fit to the ask (3.60 against 4.20); kept only as
  the fallback.
- Delete and re-add for moves: lower on efficiency and reliability once the
  transit check passed (4.00 against 5.00).

## What the tests hold

- Each shape builder's output passes the shape schema from
  `docs/penpot-api-reference.md` § 7, checked against recorded fixtures.
- Measured text width for Inter 16 px matches a recorded browser width to
  within 2 px.
- A `draw` tree becomes one `update-file` call.

## Repository knowledge

Read root `KNOWLEDGE.md` when present before each phase. Follow the global and
repository instructions for recording useful development findings.

## Phases

### Phase 1: the rough cut (needs you: try it)

- [x] A throwaway check in `scripts/prove/penpot-move.sh` sends one
  `mov-objects` in transit and one over JSON to a test file and prints which
  moved the shape; the result is written into decision 2.
- [x] `create_text`, `create_ellipse` and the `draw` tree tool land
  (expected in `internal/penpot/shapes.go`, `text.go`,
  `internal/tools/draw.go`), pinned by `TestTextMeasure`, `TestDrawOneBatch`
  and `TestDrawCaps`.
- [x] Check: `make check` passes, and `make live` draws a login screen
  (board, title text, two input rects, a button) in one call.
- [ ] You try it: ask Claude to draw a simple login screen in your test
  file, open it in Penpot, and say whether the text sits where it should.

### Phase 2: the rest of the shapes, and edits

- [x] `create_path`, `create_image`, `create_group`, `create_bool`,
  `set_layout` and rotation land, pinned by `TestPathRejectsGarbage`,
  `TestImageRejectsNonImage` and `TestRotatedGeometry`.
- [x] `update_shape`, `move_shape`, `reparent_shape`, `delete_shapes` and
  `apply_changes` land (expected in `internal/tools/edit.go`), pinned by
  `TestDeleteRefusesRoot` and `TestApplyChangesValidates`; check:
  `make check` passes.

Landed 2026-10-19: the transit encoder is `internal/transit/` (encode only,
for `mov-objects` and `reg-objects`); the SVG path parser is
`internal/penpot/path.go` (every SVG command; arcs and quadratics become
cubics); layout and image code went to `internal/tools/layout.go` and
`media.go`, raw page edits to `raw.go`. Layouts are placed by us, because a
live check showed Penpot does not reflow when a file opens; Penpot's own
reflow, triggered in the editor, gave the same order (top layer first).
`move_shape` moves by dx, dy with `mod-obj` on every descendant;
`reparent_shape` does moves between parents and reorders. `create_group`
and `create_bool` take two writes (JSON add, transit move) and delete the
empty group if the second fails. `reg-objects` computes a bool's content,
so bools need no geometry of ours. `make live` (`TestLiveShapes`) drew every
type and ran every edit tool; the result was checked by eye in Penpot.

### Phase 3: open the PR

- [x] The branch pushed and its PR opened against `main` (or, if one is
  already open for this branch, its body brought up to date) with a
  Conventional Commit title for the whole change, a body naming this plan,
  what changed and why, and the checks run with their result. The PR's URL
  printed as the evidence. Opened, never merged.

Landed 2026-10-19: https://github.com/izam-mohammed/penpot-mcp/pull/2,
stacked on #1 (base `feat/first-version`) so it shows only this plan.
