---
target_reader: "MCM implementer"
prerequisites: "None"
reading_time: "6 minutes"
reader_can_do_after_reading: "Distinguish source-of-truth, native client configuration, and a client adapter."
---

# Primer: One manifest does not mean one file format

## What problem MCM solves

An MCP server has a small common core: a local command or remote URL, optional
arguments, and a way to obtain credentials. The products that host it are not
one product family. They choose different configuration roots, file locations,
transport labels, variable syntax, and approval behavior.

MCM should therefore own one canonical manifest at ~/.mcm/config.yaml and
generate native configurations through adapters. The canonical manifest is not
copied into a client unchanged.

## Three integration modes

| mode | examples | MCM responsibility |
|---|---|---|
| direct adapter | Cursor, Codex, Claude Code, Qoder, OpenCode | Read and update the native user-level configuration. |
| profile-aware adapter | VS Code | Use a supported command or an explicit user-provided profile path. |
| export adapter | mcpc and philschmid/mcp-cli | Generate a portable config file; use an explicit client command when effective loading matters. |

The modes are distinct. mcpc maintains OAuth profiles and sessions, but it
consumes server definitions from a passed file or automatic discovery. Treating
its profile files as a server registry would be a category error.[source: 10]

## Common mistakes

1. Do not use mcpServers for VS Code. Its current mcp.json uses servers.
2. Do not generate legacy SSE or WebSocket for any MCM target. MCM's current
   contract is stdio plus Streamable HTTP; old HTTP+SSE must be migrated by the
   server owner before it can enter the manifest.[source: 15]
3. Do not store actual token values, variable references, HTTP headers, or
   working directories in the v1 canonical manifest. `command`, `args`, and
   `url` are opaque strings that the caller must ensure are public; MCM rejects
   the structured fields before it reads any native client file.
4. Do not merge a client-owned file by blind replacement. The adapter must
   retain entries that MCM does not manage.
5. Do not collapse Qoder CLI and Qoder IDE into one target. They have separate
   files and independently evolving contracts.

## FAQ

### Why not standardize on the popular mcpServers JSON?

It lowers the number of renderers for several clients, but VS Code requires a
servers root, Codex requires TOML, and OpenCode v2 requires mcp.servers. A small canonical
model plus adapters is simpler than forcing clients to conform.

### Can an old SSE endpoint be entered as an HTTP server?

No. MCM accepts only a Streamable HTTP endpoint. The old HTTP+SSE transport is
deprecated, and silently treating its URL as Streamable HTTP would hide a
server-side migration requirement.[source: 15]

### What is Qoder IDE's confidence level?

The mcpServers schema is documented. The required ~/.qoder/mcp.json location
comes from the user and is the MCM product contract; `status` reports only the
file state, and the user must separately verify loading in the installed IDE.
