# Codex skills

Fetched: 21/10/2026 from https://learn.chatgpt.com/docs/build-skills.md
(redirected from https://developers.openai.com/codex/skills.md).
Version: Codex docs as of that date (no version number on the page).

## Authentication

None.

## What this project uses

### Location

Codex follows the Agent Skills format (agentskills.io): a directory with a
`SKILL.md`. It scans:

| Scope | Directory |
| --- | --- |
| User | `$HOME/.agents/skills` |
| Repo | `$CWD/.agents/skills`, `$CWD/../.agents/skills`, `$REPO_ROOT/.agents/skills` |
| Admin | `/etc/codex/skills` |
| System | bundled with Codex |

`penpot-mcp skills install --agent codex` writes into the user directory.

### Frontmatter

`name` and `description` are required. Other fields are ignored by Codex,
so the Claude-only `when_to_use` does no harm.

### Loading

Codex reads each skill's name, description and path first, and loads the
full `SKILL.md` only when it decides to use the skill. Symlinks are
followed when scanning.

## Gotchas

- The page does not mention `$CODEX_HOME/skills`; do not rely on it.
- Tool names differ per client (`mcp__penpot-api__draw` in Claude Code), so
  the skills name tools by their bare MCP name (`draw`).
