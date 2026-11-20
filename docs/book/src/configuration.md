# Configuration

penpot-mcp is configured with environment variables, set on the
`claude mcp add` line with `-e NAME=value`. Only `PENPOT_ACCESS_TOKEN` is
required, unless you use profiles.

Never put a value in a file in a repository. The examples use
placeholders (`<your token>`, `<mcp url>`); the server never logs a token.

## Environment variables

| Variable | Needed | Does |
| --- | --- | --- |
| `PENPOT_ACCESS_TOKEN` | yes, unless `PENPOT_PROFILES` is set | The personal access token of the account the server acts as. Made under *your account > integrations > personal access tokens*. |
| `PENPOT_BASE_URL` | no | The Penpot instance. Default `https://design.penpot.app`. Set it for a [self-hosted Penpot](#self-hosted-penpot). |
| `PENPOT_MCP_URL` | no | The URL of the official Penpot MCP, key included, from *your account > integrations > mcp server*. It switches on the tab-only tools. See [routes](routes.md). |
| `PENPOT_EMAIL` | no | The Penpot login email, for board exports without a tab. See [routes](routes.md). |
| `PENPOT_PASSWORD` | no | The password for `PENPOT_EMAIL`. A Google or SSO account has none: leave both out and exports go through the tab. |
| `PENPOT_PROFILES` | no | The path of a JSON file that names several accounts. See [several accounts](#several-accounts). |
| `PENPOT_MCP_HTTP_TOKEN` | with `--http` off loopback | The bearer token clients must send in [HTTP mode](#http-mode). 16 characters or more. |

`CLAUDE_CONFIG_DIR` is read only by `penpot-mcp skills install`, to find
Claude Code's skills directory.

### What the server does with them

- The token is checked at start with one call to Penpot. A wrong or expired
  token stops the server with `PENPOT_ACCESS_TOKEN was not accepted`.
- Files the server reads or writes (palettes, token files, exports,
  `.penpot` files) must be inside the directory it was started in. Paths
  outside it are refused.
- Fonts the server needs to measure text are cached under the user cache
  directory, in `penpot-mcp/fonts`.

## Self-hosted Penpot

Set `PENPOT_BASE_URL` to the instance, and make sure the instance has
the `enable-access-tokens` flag, or there are no access tokens to make:

```sh
claude mcp add penpot -s user \
  -e PENPOT_ACCESS_TOKEN=<your token> \
  -e PENPOT_BASE_URL=https://penpot.example.com \
  -- "$(command -v penpot-mcp)"
```

## Several accounts

`PENPOT_PROFILES` is the path of a JSON file with one entry per account. The
file holds the names of environment variables, never the secrets
themselves:

```json
[
  {"name": "work", "token_env": "PENPOT_WORK_TOKEN"},
  {"name": "home", "token_env": "PENPOT_HOME_TOKEN", "base_url": "https://penpot.example.com"}
]
```

| Field | Needed | Meaning |
| --- | --- | --- |
| `name` | yes | Lowercase letters, digits, `_` or `-`, up to 32. Unique. |
| `token_env` | yes | The environment variable that holds this account's token. It must be set. |
| `base_url` | no | The instance for this account. Default `https://design.penpot.app`. |
| `email_env`, `password_env` | no | Variables that hold the login for exports. |
| `mcp_url_env` | no | The variable that holds this account's MCP URL. |

Then set each token in the server's environment, next to
`PENPOT_PROFILES`. Every tool takes a `profile` argument naming one entry;
call `whoami` with each profile to see which account it is. Without
`PENPOT_ACCESS_TOKEN`, the first entry is the default.

## HTTP mode

For clients that only speak HTTP:

```sh
penpot-mcp --http 127.0.0.1:8787
```

The same tools are served at `/mcp`. The server answers statelessly and
caps a request at 4 MB.

It refuses to listen on an address that is not loopback unless
`PENPOT_MCP_HTTP_TOKEN` is set to 16 characters or more, because anyone who
reaches the port could act as your Penpot account. Clients then send
`Authorization: Bearer <token>`. On loopback the token is optional; when
set, it is checked.

See the [command line](reference/cli.md) for the flags.
