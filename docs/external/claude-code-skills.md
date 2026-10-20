# Claude Code skills

Fetched: 19/10/2026 from https://code.claude.com/docs/en/skills
Version: Claude Code docs as of that date (no version number on the page).

## Authentication

None.

## What this project uses

### Location

A skill is a directory with a `SKILL.md`. Personal skills live in
`~/.claude/skills/<skill-name>/SKILL.md`, project skills in
`.claude/skills/<skill-name>/SKILL.md`. A personal skill wins over a project
skill of the same name. This repo keeps the source in
`skills/penpot-mcp/` and `make install-skill` symlinks it into the personal
directory.

### Frontmatter

YAML between `---` lines, which must start the file. All fields optional;
unknown fields are ignored. The ones this skill uses:

| Field | Meaning |
| --- | --- |
| `name` | command name in the `/` menu; defaults to the directory name |
| `description` | what the skill does and when to use it; Claude loads the skill when a request matches it |
| `when_to_use` | extra trigger phrases, appended to `description` |

Other fields exist (`allowed-tools`, `disallowed-tools`, `model`, `effort`,
`context`, `paths`, `disable-model-invocation`, `user-invocable`, ...); this
skill sets none of them.

### Supporting files

Files beside `SKILL.md` load only when the skill points Claude at them
("For X, see [file.md](file.md)"). `${CLAUDE_SKILL_DIR}` names the skill's
directory for bundled scripts.

## Limits

- `description` plus `when_to_use` is truncated at 1,536 characters in the
  listing: put the main use first.
- Keep `SKILL.md` under 500 lines; move detail to supporting files.
- `name` may not contain `<`, `>`, control or zero-width characters, and
  may not be `anthropic-skills`.

## Gotchas

- The description decides whether the skill loads at all: say both what it
  does and when to use it.
- A skill added mid-session shows in `/skills` after the session reloads
  skills (a new session is certain).
