# an agent fills a file's library with colours, type styles, components and design tokens, and uses them on shapes

## The ask

`docs/bridge.md` § 8 Library and tokens, on the API route. Component
overrides and variants stay on the MCP route (`plans/06-mcp-route.md`).

## Alternatives

### Decision 1: how is a component instance made without the Plugin API? (two-way door)

Searched 2026-10-19: Penpot source `common/src/app/common/logic/libraries.cljc`
(instantiate-component), the API reference § 6.3.

| Criterion | Weight | Copy the main shapes with new ids and component refs | Only on the MCP route |
| --- | ---: | ---: | ---: |
| Performance efficiency | 40 | 5 | 2 |
| Fit to the ask | 35 | 4 | 5 |
| Reliability | 25 | 3 | 4 |
| **Weighted total** | **100** | **4.15** | **3.55** |

- MCP-route efficiency 2: needs a connected tab and a 120 s-bounded call.
- Copy reliability 3: nested copies need `shape-ref` chains right; a test
  per nesting level holds it.

**Chosen: copy on the API route**, nested components to one level in this
plan, the MCP route for deeper nesting. Revisit if Penpot reports broken
copies after opening.

## The decision

Tools: `add_color`, `update_color`, `delete_color`, `add_typography`,
`update_typography`, `delete_typography`, `list_library`,
`create_component`, `instantiate_component` (decision 1),
`delete_component`, `sync_tokens`, `apply_token`, `import_palette` (hex
lists, CSS custom properties and JSON token files from a repo path).
Writes by name are idempotent: an existing asset with the same path and
name is updated, not duplicated.

Dependencies: none new. Standards: `docs/standards.md` (`make check`).

## Where it lands

| Path | New or edit | Phase | What changes | Firm? |
| --- | --- | --- | --- | --- |
| `internal/tools/library.go` | new | 1 | colour, typography, component tools | expected |
| `internal/tools/tokens.go` | new | 2 | token sets, themes, apply | expected |
| `internal/palette/` | new | 2 | repo palette parsers | expected |

*Reused, not rewritten:*

- `internal/penpot/writer.go` (plan 01) for all changes.
- `~/.claude/skills/penpot/scripts/make_bootstrap.py`: its Swift, Kotlin
  and markdown-table parsing rules are ported, not reinvented.

## Security

`import_palette` reads files from a local path the model names.

| Threat | Here | Handled by | Proved by |
| --- | --- | --- | --- |
| Information disclosure | the model points `import_palette` at a secret file | only text files under the working directory, size capped, only colour values extracted | `TestPaletteRefusesOutsideCwd` |

Checklist: OWASP Top 10 for LLM Applications 2025.

## Declined

- Instances only on the MCP route: needs a tab for a common action (3.55).

## What the tests hold

- Re-running `import_palette` creates nothing new.
- An instance's shapes reference the main component and survive the shape
  schema check.

## Repository knowledge

Read root `KNOWLEDGE.md` when present before each phase. Follow the global and
repository instructions for recording useful development findings.

## Phases

### Phase 1: the rough cut (needs you: try it)

- [x] `add_color`, `add_typography`, `list_library`, `create_component`
  and `instantiate_component` land (expected in
  `internal/tools/library.go`), pinned by `TestColorIdempotent` and
  `TestInstanceRefs`.
  Landed 2026-10-20: `create_component` turns an existing board into the
  main instance (draw it first), rather than drawing one itself. A name
  like `Brand / Primary` is split into path and name. A copy is the main's
  subtree with new ids, each shape's `shapeRef` on its original, moved by
  `shiftRaw`. Also pinned by `TestTypographyStrings`, `TestCreateComponent`
  and `TestListLibrary`.
- [ ] Check: `make check` passes, and `make live` adds two colours and a
  button component, then places two instances.
- [ ] You try it: ask Claude to add your brand colours and a button
  component to the test file, open the assets panel, and say whether they
  look right.

### Phase 2: tokens and palettes

- [x] `sync_tokens`, `apply_token`, `import_palette` and the update and
  delete tools land (expected in `internal/tools/tokens.go`,
  `internal/palette/`), pinned by `TestPaletteRefusesOutsideCwd`; check:
  `make check` passes.
  Landed 2026-10-20: `sync_tokens` sends `set-tokens-lib` with the DTCG
  multi-set JSON, which Penpot decodes with its own `read-multi-set-dtcg`
  (`common/src/app/common/types/tokens_lib.cljc`); it replaces the file's
  tokens, so it carries `destructiveHint`. `apply_token` sets the value and
  `appliedTokens` (fill, stroke colour, radius, opacity) and reads the value
  from `get-file`'s `tokensLib` unless `value` is given. Both are unproven
  against a live file until `TestLiveTokens` runs. `import_palette` reads
  Swift, Kotlin, markdown tables, CSS custom properties, DTCG or Style
  Dictionary JSON and hex lists (`internal/palette/`), only under the
  server's working directory (`Deps.Root`), never `.env` or `.git`.
  `delete_component` also deletes the main instance, as Penpot's editor does.

### Phase 3: open the PR

- [ ] The branch pushed and its PR opened against `main` (or, if one is
  already open for this branch, its body brought up to date) with a
  Conventional Commit title for the whole change, a body naming this plan,
  what changed and why, and the checks run with their result. The PR's URL
  printed as the evidence. Opened, never merged.
