# penpot-mcp: what is left to build, in order

### Phase 1: the first version you can try

- [x] `plans/01-first-version.md` - a Go binary lists your Penpot files and
  draws a board with a rectangle, no browser open.
- [x] `plans/09-skill.md` phase 1 - a skill sets it up and drives it, with
  Claude in Chrome for the token hand-off and visual checks.

### Phase 2: draw for real

- [x] `plans/02-drawing.md` - text, images, paths, groups, layouts and
  whole screens in one call, plus edits and deletes.

### Phase 3: everything the dashboard does

- [x] `plans/03-dashboard.md` - projects, files and pages created, renamed,
  moved and deleted from the agent.
- [x] `plans/04-library.md` - colours, type styles, components and tokens.

### Phase 4: get designs out

- [x] `plans/05-export.md` - boards to PNG, SVG or PDF on disk, and whole
  `.penpot` files.

### Phase 5: the browser-only extras

- [x] `plans/06-mcp-route.md` - plugin code, reflow, text measuring and CSS
  through the official MCP when a tab is connected.
- [x] `plans/09-skill.md` phase 2 - the skill connects and wakes the plugin
  tab with Claude in Chrome.

### Phase 6: working with people

- [x] `plans/07-collaboration.md` - comments, share links, webhooks and
  snapshots.

### Phase 7: ship it

- [x] `plans/08-release.md` - HTTP transport, several accounts and released
  binaries.
- [x] `plans/09-skill.md` phases 3-4 - the skill's full tool list, kept in
  step with the code.

### Phase 8: skills you can install

- [ ] `plans/10-skills.md` - skills split by job, installed by the binary
  into claude code or codex, and the readme says so.

### Phase 9: repository standards

- [ ] `plans/13-quality-gates.md` - a coverage floor that climbs to 90 %,
  the shared security gate on every PR, and actions pinned by SHA.
- [ ] `plans/11-readme-banner.md` - the banner in a light and a dark
  theme.
- [ ] `plans/12-site.md` - a one-page site on GitHub Pages.

### Phase 10: the detailed docs

- [ ] `plans/14-docs.md` - an mdBook documentation site at `/docs/`, with
  the tool and CLI reference generated from the server.
