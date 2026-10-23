# the readme banner follows the viewer's github theme, light or dark

## The ask

Every repository should have a proper minimal README with a banner (a light
and a dark SVG, made with the `github-banner` skill) or a UI screenshot near
the top.

Built so far: `docs/built/BUILT-2026-10-22-1826.md`.

What it means here: the README is already minimal (cut to install and why
in PR `#2`) and already opens with a banner (PR `#8`), but it is one SVG,
`docs/assets/banner.svg`, drawn on a dark night sky with a rounded card. On
GitHub's light theme it sits as a dark slab on a white page. The skill's
first rule is two files picked by `<picture>`. The concept (a pot of pens
rocketing through a portal, leaving a sleepy browser tab behind) is not yet
in the skill's `concepts-used.md`.

## Alternatives

### Decision 1: how does the banner follow the theme? (two-way door)

GitHub applies `prefers-color-scheme` inside `<picture>` from the viewer's
GitHub theme; a `@media` query inside one SVG follows the operating system
instead (`github-banner` skill, rule 1).

Searched 2026-10-22: "github readme picture prefers-color-scheme dark light
image", "svg media query prefers-color-scheme github readme"; GitHub's docs
on specifying the theme an image is shown to, and the skill's rules.

| Must | Baseline: one dark SVG | Light and dark SVGs in `<picture>` | One SVG with an inner `@media` query |
| --- | --- | --- | --- |
| Follows the viewer's GitHub theme | no (dark in both) | yes | no (follows the OS) |
| **Survives** | no | **yes** | no |

Only the pair survives, so there is nothing to score against it.

**Chosen: light and dark SVGs in `<picture>`.** Revisit if GitHub starts
honouring media queries inside SVG images by its own theme.

### Decision 2: same concept or a new one? (two-way door)

The skill wants a fresh idea for every new banner; this is the same banner
in a second theme.

Searched 2026-10-22: the skill's `concepts-used.md` (no penpot entry) and
rule 4 (no idea reused from another banner).

| Criterion | Keep the portal concept in both themes | A new concept |
| --- | --- | --- |
| Performance efficiency | same size | same size |
| Fit to the ask | the README's voice and PR `#8`'s joke kept | the user's chosen joke thrown away |
| Compatibility | alt text and README wording unchanged | both rewritten |

**Chosen: keep the portal concept in both themes**, and log it in
`concepts-used.md` so no other banner takes it. Revisit if the user wants
a new joke.

Revisited 23/10/2026: the user wanted a new joke, so both themes now
carry a lost poster for the browser tab instead of the portal.

## The decision

`docs/assets/banner-dark.svg` is the current banner with the card's
backdrop kept; `docs/assets/banner-light.svg` redraws it on a light sky with
the same layout and joke, each built and linted by the skill's `build.py`
(decision 1, decision 2). `README.md`'s first line becomes a `<picture>`
with both, the current alt text kept. `docs/assets/banner.svg` is replaced
by the pair. The files stay in `docs/assets/`, where the repository keeps
them, not the skill's default `assets/`.

No new dependency or permission.

Standards: `docs/standards.md` (format, lint, build, tests, govulncheck and
gitleaks through `make check`).

Security: no new input, data, secret, dependency or permission; two static
SVGs with no scripts or external URLs.

## Where it lands

| Path | New or edit | Phase | What changes | Firm? |
| --- | --- | --- | --- | --- |
| `docs/assets/banner-dark.svg` | new | 1 | dark banner | fixed |
| `docs/assets/banner-light.svg` | new | 1 | light banner | fixed |
| `docs/assets/banner.svg` | edit | 1 | replaced by the pair | reuse |
| `README.md` | edit | 1 | `<picture>` on the first line | reuse |

*Reused, not rewritten:*

- `docs/assets/banner.svg`: its shapes, title and description become the
  dark file; the light file reuses the geometry with a light palette.
- The skill's `scripts/build.py` and `scripts/preview.sh`: the lint and
  the side-by-side preview.

## Declined

- One dark SVG: fails the must on GitHub's light theme.
- One SVG with an inner `@media` query: follows the OS, not the GitHub
  theme.
- A new concept: throws away the user's chosen joke for no gain.

## What the tests hold

The skill's `build.py` lint (no overlap, text contrast at least 4.5:1 in
each theme, no animation, no external URL). `make check` holds that no code
changed.

## Repository knowledge

Read root `KNOWLEDGE.md` when present before each phase. Follow the global and
repository instructions for recording useful development findings.

## Phases

### Phase 1: the rough cut: both themes (needs you: try it)

- [x] `docs/assets/banner-dark.svg` and `docs/assets/banner-light.svg` built
  from the current banner by the skill's `build.py`, its lint passing, and
  `docs/assets/banner.svg` removed.
- [x] `README.md`'s first line is a `<picture>` with the pair and the
  current alt text.
- [ ] Check: `make check` passes, and the skill's `preview.sh` renders both
  files with nothing overlapping.
- [ ] You try it: open the branch's README on GitHub in light and in dark
  theme and say whether both read.

### Phase 2: log the concept

- [x] The skill's `concepts-used.md` gains the portal line, so no other
  banner reuses it, and `CLAUDE.md` or `docs/features.md` paths naming
  `banner.svg` are updated if any do.
- [ ] Check: `make check` passes, and `grep -rn banner.svg` finds nothing
  stale.

### Phase 3: open the PR

- [x] The branch pushed and its PR opened against the branch it was cut from
  (or, if one is already open for this branch, its body brought up to date)
  with a Conventional Commit title for the whole change, a body naming this
  plan, what changed and why, and the checks run with their result. The PR's
  URL printed as the evidence. Opened, never merged.
