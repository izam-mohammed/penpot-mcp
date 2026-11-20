# getting listed

where penpot-mcp can be listed, what each place wants, and the bits only
izam can do (logins, forms). researched 2026-11-20 from each directory's own
docs. nothing here has been submitted.

repo-side files already in the root:

- `server.json` ~ the official registry entry. schema checked against
  `https://static.modelcontextprotocol.io/schemas/2025-12-11/server.schema.json`
  (required: `name`, `description`, `version`; description max 100 chars).
- `glama.json` ~ glama ownership claim. `$schema` plus `maintainers`, the two
  fields glama's own blog post documents.

bump `version` in `server.json` on every release, it has to match what you
publish.

## the catch: a go binary has no package type

the official registry only knows `npm`, `pypi`, `nuget`, `cargo`, `oci` and
`mcpb`. a plain go binary on github releases isn't one of them. so
`server.json` ships with no `packages` block, which the schema allows
(`packages` and `remotes` are optional) and the registry docs show in their
"custom installation" example. what i couldn't verify is whether the
publish endpoint accepts a package-less entry for a stdio server. try it;
if it refuses, the options are:

| route | what it takes |
| --- | --- |
| `mcpb` | build a `.mcpb` bundle (a `manifest.json` with `"server": {"type": "binary", ...}` plus the binary), attach it to the github release, add a `packages` entry with `registryType: "mcpb"`, the release url as `identifier`, its `fileSha256` and `transport: {"type": "stdio"}`. the url must contain "mcp" (`penpot-mcp` does). one bundle per os/arch, so goreleaser needs extra work |
| `oci` | push an image to `ghcr.io` with the label `io.modelcontextprotocol.server.name=io.github.izam-mohammed/penpot-mcp`, then add an `oci` package entry |

smithery's local-server route also wants an `.mcpb`, so building one pays
twice.

## official mcp registry

the one that matters. pulsemcp ingests from it, and the
`modelcontextprotocol/servers` readme sends everyone there.

steps:

1. install the publisher: `brew install mcp-publisher` (or a binary from
   github.com/modelcontextprotocol/registry/releases).
2. from the repo root: `mcp-publisher validate`
3. `mcp-publisher login github` ~ shows a device code, you enter it on
   github.com. the `io.github.izam-mohammed/` prefix only works with this
   login, as that github user.
4. `mcp-publisher publish`
5. check: `curl "https://registry.modelcontextprotocol.io/v0.1/servers?search=penpot-mcp"`

ci option: a tag-triggered job with `permissions: id-token: write`, running
`mcp-publisher login github-oidc` then `mcp-publisher publish`. pin the
publisher version and verify its sigstore bundle. do it by hand once first.

entry text:

- name: `io.github.izam-mohammed/penpot-mcp`
- description (100 chars max): `Drive Penpot through its backend API: files, shapes, libraries and exports, no browser tab.`

izam must: run the login (device code) and the publish.

## glama

glama crawls github on its own, so the server may already be there. the file
is for claiming it.

1. `glama.json` is already in the root. merge it to main first.
2. find the server on glama.ai (search `penpot-mcp`), or use "add server" on
   glama.ai/mcp/servers and point it at the repo.
3. run the claim flow. re-run it after any edit to `glama.json`.
4. once claimed: fix name and description, and read the score. the score
   badge is what awesome-mcp-servers wants (see below).

the badge lives at
`https://glama.ai/mcp/servers/izam-mohammed/penpot-mcp/badges/score.svg`
(pattern copied from existing awesome-mcp-servers entries; confirm the url
resolves after claiming).

entry text: take it from the repo description.

izam must: sign in to glama and click claim. the repo is on a personal
account, so signing in with github also links it without the file.

## smithery

the old `smithery.yaml` build-from-repo route targets hosted servers. for a
local stdio server like this one, smithery's current docs say it distributes
a prebuilt `.mcpb` bundle that clients download and run. the page that
spells out the exact cli command (`smithery.ai/docs/build/publish`)
returned 404 when i fetched it, so i couldn't verify the commands. no
smithery file is added to the repo for that reason.

steps (outline, verify on the live docs):

1. build the `.mcpb` (see the table above).
2. publish it through the smithery cli or the dashboard, under your
   smithery namespace.

entry text:

- name: `penpot-mcp`
- description: `drives penpot through its backend api, no browser tab babysat`
- category: design
- install: `brew install --cask izam-mohammed/tap/penpot-mcp`

izam must: create a smithery account, pick the namespace, run the publish.

## pulsemcp

submissions are paused: pulsemcp.com/submit says it isn't accepting new
servers or listing changes, no reopening date. it says it picks up servers
from the official registry automatically. so do the official registry, then
nothing. check the submit page again in a while.

izam must: nothing for now.

## mcp.so

1. open mcp.so/submit.
2. enter the repo url, `https://github.com/izam-mohammed/penpot-mcp`. it
   fetches the details; review name, description, config and tools.
3. submit. it goes live after review.

entry text:

- name: `penpot-mcp`
- description: `the official penpot mcp wants a browser tab babysat forever. got tired of it, so this one just talks to the api`
- install: `brew install --cask izam-mohammed/tap/penpot-mcp`

izam must: submit the form (it may ask for a login).

## punkpeye/awesome-mcp-servers

public github repo, installable server, one line per entry, alphabetical
within its category. a pr with a clear title. the legend for go is the
gopher race-car emoji (`🏎️`), scope `🏠` (local service), os `🍎 🪟 🐧`.
category: the list has `🎨 Art & Culture` and `📐 Architecture & Design`;
design tooling fits `📐 Architecture & Design`. confirm in the readme before
you pick ~ i only read the first 100k characters of it.

entry, in the list's format (this one is not lowercase, the list dictates it):

```md
- [izam-mohammed/penpot-mcp](https://github.com/izam-mohammed/penpot-mcp) 🏎️ 🏠 🍎 🪟 🐧 - Drive Penpot through its backend API: files, shapes, libraries and exports, no browser tab needed.
```

glama badge form, if the maintainers want it after you've claimed:

```md
[![izam-mohammed/penpot-mcp MCP server](https://glama.ai/mcp/servers/izam-mohammed/penpot-mcp/badges/score.svg)](https://glama.ai/mcp/servers/izam-mohammed/penpot-mcp)
```

steps: fork, branch, edit `README.md` (insert alphabetically), push, open the
pr. agents can add `🤖🤖🤖` to the pr title for fast-track merging.

izam must: fork and open the pr from his account.

## modelcontextprotocol/servers

skip it. its contributing guide says the readme's third-party list "has been
retired in favor of the Registry". the official registry above is the entry.

## github topics

already set: `claude-code`, `cli`, `design-tools`, `go`, `mcp`, `mcp-server`,
`model-context-protocol`, `penpot`. missing from the wanted set: `design`,
`golang`, `claude`, `ai-agents`. add them (not run):

```sh
gh repo edit izam-mohammed/penpot-mcp \
  --add-topic penpot --add-topic mcp --add-topic mcp-server \
  --add-topic design --add-topic golang --add-topic claude \
  --add-topic ai-agents
```

re-adding an existing topic is harmless.

## order

1. merge this pr.
2. set the topics.
3. official registry (`login github`, `publish`).
4. glama claim.
5. mcp.so form.
6. awesome-mcp-servers pr, once glama shows the score.
7. smithery, after an `.mcpb` exists.
