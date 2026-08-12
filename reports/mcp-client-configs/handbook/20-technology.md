---
target_reader: "MCM implementer"
prerequisites: "handbook/10-primer.md"
reading_time: "12 minutes"
reader_can_do_after_reading: "Implement a canonical transport model and select the right adapter behavior for each client."
---

# Technology: Adapter model and compatibility matrix

## Canonical model

The current model has two transport variants:

~~~yaml
servers:
  example:
    transport: stdio | streamable-http
    command: optional string
    args: optional string array
    url: optional string
~~~

The model rejects `cwd`, `env`, `headers`, OAuth fields, secret values, client
UI preferences, tool approval rules, and sandbox policies. Those are
client-specific fields and are out of the first renderer contract.

## Target matrix

| adapter | native root and user target | documented transports | rendering decision |
|---|---|---|---|
| cursor | mcpServers in ~/.cursor/mcp.json | stdio, Streamable HTTP | direct JSON writer; stdio emits `type: "stdio"`, command and args |
| claude-code | mcpServers in ~/.claude.json user scope | stdio, Streamable HTTP | direct JSON writer within user scope |
| codex | mcp_servers tables in ~/.codex/config.toml | stdio, Streamable HTTP | direct TOML writer |
| vs-code | servers in active user-profile mcp.json | stdio, Streamable HTTP | profile-aware writer or code CLI |
| qoder-cli | mcpServers in ~/.qoder/settings.json | stdio, Streamable HTTP | direct JSON writer |
| qoder-ide | mcpServers in ~/.qoder/mcp.json | stdio, Streamable HTTP | direct JSON writer; status reports only file state |
| opencode | mcp.servers in ~/.config/opencode/opencode.json(c) | local, remote | JSON/JSONC reader; map stdio to local and Streamable HTTP to remote |
| mcpc | standard mcpServers export | stdio, Streamable HTTP | export file plus connect workflow |
| philschmid-mcp-cli | mcpServers in mcp_servers.json | stdio, Streamable HTTP | export/direct file writer |

The Qoder IDE path is user-confirmed, while its public documentation only
proves the JSON schema. It is intentionally not promoted above low confidence.

## Transport rules

| transport | allowed targets | handling for every other target |
|---|---|---|
| stdio | all adapters | supported core |
| streamable-http | all adapters | supported core; client-native field names remain adapter-specific |

OpenCode v2 nests every server under mcp.servers and intentionally models
local and remote rather than a separate SSE type. Emit remote only for the
verified Streamable HTTP contract; do not infer SSE support from another
client.[source: 9]

## Renderer requirements

1. Read the native file before generating a plan.
2. Assign MCM ownership by server name or a stable MCM metadata registry,
   not by assuming every entry is managed.
3. Preserve unknown top-level fields and unmanaged server definitions.
4. Validate transport support before serializing.
5. Validate a rendered file according to its syntax before apply.
6. Report the native target and the managed server names that will change,
   never unmanaged configuration values.
7. For OpenCode, select the only existing `opencode.json` or `opencode.jsonc`;
   if both exist, require an explicit one-target path. Rewriting JSONC may
   remove comments but preserves every unmodified value, including nested mcp
   siblings and numeric literals.

## Client-specific differences worth preserving

- VS Code has inputs and sandbox fields; do not synthesize either in MCM v1.
- Codex, OpenCode, and philschmid/mcp-cli have distinct environment-variable
  and authentication facilities. MCM v1 deliberately does not model or render
  any of them.[source: 4][source: 9][source: 11]
- OpenCode distinguishes local command arrays from remote URL objects and
  supplies its own OAuth flow.[source: 9]
- mcpc can auto-discover standard files. MCM should make exported-file use
  explicit so an unrelated discovery file does not silently alter a session.
  [source: 10]

## Validation matrix

| scenario | evidence required |
|---|---|
| stdio export | generated command and args parse in target syntax |
| HTTP export | generated URL parses in target syntax |
| unsupported legacy SSE or WebSocket | adapter returns the target name, requested transport, migration guidance, and no writes |
| Qoder IDE apply | configured file exists; installed-IDE loading is a separate manual verification |
| VS Code apply | supported user-profile command succeeds or explicit --path was supplied |
| unmanaged config | plan names only MCM-owned server entries and preserves unmanaged values without printing them |
