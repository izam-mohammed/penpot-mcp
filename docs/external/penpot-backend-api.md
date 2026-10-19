# Penpot backend RPC API

Fetched: 19/10/2026 from https://design.penpot.app/api/main/doc/openapi.json
(`info.version` 2.18, 149 commands), the `penpot/penpot` source (`main` and
`develop`: `backend/src/app/rpc/commands/*.clj`,
`backend/resources/rlimit.edn`, `backend/resources/climit.edn`,
`common/src/app/common/files/changes.cljc`) and
https://help.penpot.app/technical-guide/integration/.
Version: Penpot 2.18 (hosted); source is newer (163 commands).

The full reference is `../penpot-api-reference.md`. This file is the summary
the plans cite; where the two differ, the full reference wins.

## Authentication

- Header `Authorization: Token <access-token>`. `Bearer` is treated as a
  session token and fails for an access token.
- Env var: `PENPOT_ACCESS_TOKEN`. Base URL: `PENPOT_BASE_URL` (default
  `https://design.penpot.app`).
- The exporter (`/api/export`) accepts only the session cookie
  `auth-token` from `login-with-password`. Env vars: `PENPOT_EMAIL`,
  `PENPOT_PASSWORD`.
- Self-hosted servers need `enable-access-tokens` in `PENPOT_FLAGS`.

## What this project calls

- Transport: `POST {base}/api/main/methods/{command}`, JSON body, always
  `Accept: application/json` (otherwise the reply is transit), plus
  `Content-Type: application/json` and `X-Client: penpot-mcp/<version>`.
  Reference § 1.
- Token check: `get-profile`; an `id` of
  `00000000-0000-0000-0000-000000000000` means the token was not accepted.
  Reference § 2.1, § 10.1.
- Navigation: `get-teams`, `get-projects`, `get-project-files`, `get-file`,
  `get-page`. Reference § 10.2 to § 10.6, § 13.
- Writes: `update-file {id, sessionId, revn, vern, changes}`; reply
  `{revn, lagged}` where `revn` is the value before the update. Reference
  § 5, change types § 6, shape schema § 7, minimal shapes § 7.8.
- Media: `upload-file-media-object` (multipart). Reference § 8, § 10.10.
- Export: `POST /api/export`, transit body, cookie auth. Reference § 10.12,
  § 12.

## Errors

Body `{type, code, hint, explain?}`. Retry: `429` (honour `Retry-After`).
Re-fetch then retry: `400 revn-conflict`, `409 vern-conflict`. Fix the
request: `400 params-validation`, `500 assertion/data-validation` (invalid
shape, read `explain`). Reference § 3.1.

## Limits

- Rate (when `rpc-rlimit` is on): `update-file` bucket 1000, refill
  500/min; `get-file` 180, refill 90/min; full table in reference § 3.2.
- Concurrency: one `update-file` in flight per profile, queue 5, 60 s
  timeout, even across files; 10 calls in flight per profile overall.
  Reference § 3.3.

## Gotchas

- The backend recomputes no geometry: send `selrect`, `points`,
  `transform`, `transformInverse` consistent with `x, y, width, height`.
- Use `assign`, not `set`, in `mod-obj` from JSON.
- `mov-objects` and `reorder-children` over JSON probably do nothing; use
  transit or delete and re-add with the same id. [UNCERTAIN]
- Board type is `frame`, ellipse is `circle`.
- Since 2.19 the server rejects client ids on `create-project`,
  `create-file` and media creation.
