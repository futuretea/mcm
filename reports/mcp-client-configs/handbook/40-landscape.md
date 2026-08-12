---
target_reader: "MCM implementer"
prerequisites: "handbook/10-primer.md"
reading_time: "7 minutes"
reader_can_do_after_reading: "Prioritize adapters by contract stability and understand their incompatibilities."
---

# Landscape: Stable groups and rollout order

## Group 1: Standard JSON producers and consumers

Cursor, Claude Code project files, Qoder CLI, Qoder IDE, and
philschmid/mcp-cli all use a top-level mcpServers structure. This does not make
their files interchangeable: scope, authentication, accepted transport strings,
and management UX still differ. Reuse a shared JSON server renderer only after
the adapter chooses its target root and supported fields.

mcpc belongs beside this group as a consumer. It can receive a standard JSON
file by path and can auto-discover several common client files, but it does not
offer one stable server-registry file to overwrite.[source: 10]

## Group 2: Native schema outliers

Codex uses TOML and supports only stdio plus Streamable HTTP. VS Code uses a
servers map and profile-owned configuration. OpenCode v2 uses mcp.servers with
local and remote variants. These should be separate packages from day one; hiding them
behind a large conditional JSON renderer will make validation ambiguous.

## Recommended delivery sequence

1. Build the canonical schema, syntax validation, value-free plan summary, and no-write status.
2. Add Cursor, Claude Code, Qoder CLI, and philschmid/mcp-cli adapters.
3. Add Codex and OpenCode as distinct schema renderers.
4. Add VS Code with an explicit profile path or code CLI integration.
5. Add Qoder IDE with the user-confirmed path; status must remain limited to
   file state and loading is manually verified in the installed IDE.
6. Add mcpc as an export-and-connect integration.
7. Keep legacy SSE and WebSocket out of the manifest; document Streamable HTTP
   migration guidance instead.[source: 15]

This order starts with direct, inspectable files and postpones profile or
discovery behavior. It minimizes adapter-specific state while preserving the
user's required client list.

## Open questions

- Which Qoder IDE release and platform confirm ~/.qoder/mcp.json at runtime?
- Should VS Code integration require code on PATH or an explicit profile path?
- Should MCM own entries by a persisted registry, a naming convention, or an
  adapter-side managed list?

These are product decisions or runtime facts, not configuration guesses. They
must be closed before apply behavior claims success.
