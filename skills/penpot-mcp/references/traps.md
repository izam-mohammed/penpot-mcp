# Penpot API traps

What the backend API does that an agent does not expect. Source:
`docs/penpot-api-reference.md` in the penpot-mcp repo.

| Trap | What happens | What the server or you do |
| --- | --- | --- |
| No geometry from the backend | Penpot stores exactly the box it is sent; nothing is recomputed | the server computes `selrect`, `points`, `transform`; you give only x, y, width, height |
| Absolute coordinates | a child's x and y are page coordinates, not relative to its board | add the board's x and y yourself |
| One write at a time | Penpot runs one `update-file` per account at once; more queue, then fail with 429 | call draw tools one after another |
| Rate limits | `update-file` allows bursts of 1000 a minute; reads are lower (`get-file` 180, then 90 a minute) | prefer `get_page_tree` over re-listing files |
| Version restored in the browser | the next write fails with `vern-conflict` | the server re-reads once; if it still fails, re-read the tree and redo |
| Board type name | boards are `frame` in the tree, ellipses `circle` | match on these names |
| Text, images, paths, edits, deletes | not in this version of the server | say so; do not fake text with rectangles |
| Exports | not in this version | look at the drawing with Claude in Chrome instead |
| Token in the browser | the token dialog shows it once | the user copies it; never you |
