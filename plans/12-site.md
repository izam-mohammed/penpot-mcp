# penpot-mcp has a one-page site that says what it does and how to install it

## The ask

Every repository should have a `site/` folder holding a presentable
one-page site for users, modelled on `mac-notch/site/index.html`, and a
recorded way it is published.

Built so far: `docs/built/BUILT-2026-10-22-1826.md`.

What it means here: there is no site. The reference page is one
self-contained HTML file (inline CSS, no scripts, no fetched fonts): a
header, one pitch, one button to the latest release, a footer linking
GitHub. How mac-notch's page is published is not recorded in that
repository (no Pages workflow; GitHub's Pages API answered 404 for it on
22/10/2026), so this plan decides publishing for itself. `penpot-mcp` is
public, which GitHub Pages needs on a free plan. The README's voice is
lowercase and casual ("yo bro"), and the page keeps it.

## Alternatives

### Decision 1: where is the page published? (two-way door)

The page has to be reachable at a URL, built from `site/` on `main`, with
no person deploying it.

Searched 2026-10-22: "GitHub Pages deploy static site from folder
actions/deploy-pages vs Cloudflare Pages vs Netlify 2026", "GitHub Pages
publishing source custom workflow upload-pages-artifact"; GitHub's docs on
custom Pages workflows (`actions/upload-pages-artifact@v4`,
`actions/configure-pages@v5`, the `github-pages` environment, no deploy on
pull requests), 2026 comparisons of Cloudflare Pages and Netlify.

| Must | Baseline: no site | GitHub Pages, deployed by a workflow | Cloudflare Pages | Netlify |
| --- | --- | --- | --- | --- |
| Reachable at a URL | no | yes | yes | yes |
| No new account, service or secret | yes | yes | no (a Cloudflare account and API token) | no (a Netlify account and token) |
| **Survives** | no | **yes** | no | no |

Only GitHub Pages survives, so there is nothing to score against it.

**Chosen: GitHub Pages, deployed from `site/` by a workflow on push to
`main`.** It needs only the repository's own token. Revisit if the page
needs redirects or analytics Pages cannot give.

### Decision 2: how is the page made? (two-way door)

One page, no content pipeline; it has to load fast and match the reference.

Searched 2026-10-22: "single html landing page vs hugo vs jekyll for a
one-page project site"; the reference `mac-notch/site/index.html`
(140 lines, inline CSS, no script), Hugo's and Jekyll's docs on Pages.

| Criterion | Weight | Hand-written single HTML | Hugo | Jekyll rendering the README |
| --- | ---: | ---: | ---: | ---: |
| Performance efficiency | 45 | 5 | 4 | 3 |
| Fit to the ask | 30 | 4 | 5 | 2 |
| Maintainability of the result | 25 | 4 | 3 | 5 |
| **Weighted total** | **100** | **4.45** | **4.05** | **3.20** |

- Single HTML efficiency 5: one request, nothing fetched; Hugo 4: same
  output with a build step in CI; Jekyll 3: a theme's CSS and fonts.
- Fit: Jekyll 2, the README's mermaid diagram does not render outside
  GitHub and the page would not be a one-pager.
- Maintainability: the single file copies the install lines from the
  README (4); Hugo adds a toolchain (3).

**Chosen: hand-written single HTML.** The reference's model, nothing to
build. Revisit if a second page is wanted.

## The decision

`site/index.html`, one self-contained page in the reference's structure: a
header with the name and links to Releases and GitHub, the pitch ("talks to
penpot's api directly, no tab to babysit"), the dark banner from
`plans/11-readme-banner.md` inlined, the paste-in install prompt and the
`brew install --cask izam-mohammed/tap/penpot-mcp` line, a short version of
the README's "how it's different" table, a "latest release" link to
`https://github.com/izam-mohammed/penpot-mcp/releases/latest` (archives
carry the version in their names, so no stable direct download), and a
footer (decision 2). `.github/workflows/pages.yml` uploads `site/` and
deploys it to GitHub Pages on push to `main` and by hand (decision 1).

New dependencies: `actions/checkout`, `actions/configure-pages`,
`actions/upload-pages-artifact`, `actions/deploy-pages`, pinned by SHA.
New permission: `pages: write` and `id-token: write` on the deploy job
only. A person turns Pages on with the source set to GitHub Actions.

Standards: `docs/standards.md` (format, lint, build, tests, govulncheck and
gitleaks through `make check`).

## Where it lands

| Path | New or edit | Phase | What changes | Firm? |
| --- | --- | --- | --- | --- |
| `site/index.html` | new | 1 | the page | fixed |
| `.github/workflows/pages.yml` | new | 1 | deploy `site/` | expected |
| `docs/standards.md` | edit | 1 | the Pages workflow under CI | reuse |
| `README.md` | edit | 2 | a link to the site | reuse |

*Reused, not rewritten:*

- `mac-notch/site/index.html`: the structure and approach (one file,
  inline CSS, no scripts), not its content or colours.
- `README.md`'s install prompt and comparison table, copied, not
  rewritten.
- `docs/assets/banner-dark.svg` from `plans/11-readme-banner.md`, inlined.

## Security

The page is static and public; the risk is the deploy path and the install
text it shows.

| Threat | Here | Handled by | Proved by |
| --- | --- | --- | --- |
| Elevation of privilege: a PR deploys the site | `pages.yml` with `pages: write` | triggers are `push` to `main` and `workflow_dispatch`; write permissions on the deploy job, `permissions: {}` at the top | the PR shows no Pages run |
| Tampering: an action replaced under its tag | four actions | pinned by full SHA | read in the PR diff; zizmor once `plans/13-quality-gates.md` lands |
| Spoofing: the page's install text differs from the README's | hand-copied | copied in the same phase, both in the PR diff | read side by side in the try |
| Information disclosure: a third-party request from the page | fonts or scripts | none fetched; inline CSS only | the browser's network panel shows one request |

The page never shows or asks for a token; the install prompt says the
agent must not ask for it. No data collected, no cookies.

## Open decisions

- A custom domain: default, none; the page lives at
  `https://izam-mohammed.github.io/penpot-mcp/`.

## Declined

- No site: failed the must.
- Cloudflare Pages and Netlify: failed the must (a new account and token).
- Hugo: lost on efficiency and maintainability for one page.
- Jekyll rendering the README: lost on fit.

## What the tests hold

`make check` holds that no code changed. The try checks the page at phone
width with no horizontal scroll, every link resolving, and one network
request.

## Repository knowledge

Read root `KNOWLEDGE.md` when present before each phase. Follow the global and
repository instructions for recording useful development findings.

## Phases

### Phase 1: the rough cut: the page, published (needs you: try it)

- [ ] `site/index.html` with the header, pitch, banner, install prompt and
  brew line, comparison, latest-release link and footer, self-contained,
  readable at 360 px wide.
- [ ] `.github/workflows/pages.yml` uploads `site/` and deploys it on push
  to `main` and by hand, write permissions on the deploy job only, actions
  pinned by SHA; `docs/standards.md` names it under CI.
- [ ] Check: `make check` passes, and opening `site/index.html` from disk
  shows the page with one network request.
- [ ] You try it: turn Pages on (Settings, Pages, source GitHub Actions),
  run the workflow on the branch by hand, open the URL on phone and laptop,
  and say whether it reads as penpot-mcp's page.

### Phase 2: point people at it

- [ ] `README.md` links the site under the banner, and the repository's
  homepage is set to the Pages URL (`gh repo edit --homepage`).
- [ ] Check: `make check` passes, and the site URL answers 200.

### Phase 3: open the PR

- [ ] The branch pushed and its PR opened against the branch it was cut from
  (or, if one is already open for this branch, its body brought up to date)
  with a Conventional Commit title for the whole change, a body naming this
  plan, what changed and why, and the checks run with their result. The PR's
  URL printed as the evidence. Opened, never merged.
