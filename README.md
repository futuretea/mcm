# MCM

[中文文档](README.zh-CN.md)

MCM is a local Go CLI for managing one public MCP server manifest and rendering it into multiple user-level MCP client configurations on macOS and Linux.

It supports Cursor, Codex, Claude Code, VS Code, Qoder CLI, OpenCode, [philschmid/mcp-cli](https://github.com/philschmid/mcp-cli), and mcpc. Qoder IDE has file-rendering support only; its runtime loading behavior has not been independently verified.

## Scope

- MCM manifest: `~/.mcm/config.yaml` by default.
- Supported transports: `stdio` and `streamable-http`.
- Rejected: legacy SSE, WebSocket, `cwd`, `env`, headers, OAuth, and credential fields.
- MCM never creates client directories. Client target parents must already exist.
- A direct target may not be a symlink or another non-regular file.

MCM only accepts public `command`, `args`, and `url` values. Do not put credentials or tokens in the manifest.

## Install and build

The repository pins Go 1.26.5 in `.tool-versions`.

```bash
go build -o ./mcm ./cmd/mcm
```

## Docker E2E

Run the Linux end-to-end suite in a clean Docker build:

```bash
docker build --target e2e .
```

The suite builds and invokes the real `mcm` binary with an isolated `HOME`. It covers the public CLI workflow, all supported target files, and stdout/stderr warning contracts without mounting any host configuration.

## First use

```bash
./mcm init
./mcm server add --name filesystem --command npx --arg -y --arg @modelcontextprotocol/server-filesystem
./mcm server add --name remote-tools --url https://example.test/mcp
./mcm validate
./mcm server list
./mcm plan --target cursor --target codex
./mcm apply --target cursor --target codex --yes
./mcm status --target cursor --target codex
```

`plan`, `apply`, and `status` require at least one explicit `--target`. Repeated targets are deduplicated in stable target order.

Interactive `apply` asks for one `yes` confirmation. Non-interactive usage requires `--yes`.

Use `./mcm --help` or `./mcm <command> --help` to discover commands and flags. `server add` only creates a new name; use `server update` to replace an existing definition. Move the binary to a directory on your `PATH` if you prefer to invoke it as `mcm`.

## Global manifest location flags

Global flags appear before the command:

```text
mcm [--home DIR] [--config FILE] <command> ...
```

- `--home DIR` selects MCM's private root for state, journal, lock, exports, and the default manifest. It must be absolute.
- `--config FILE` selects one exact manifest file. It must be absolute and can be combined with `--home`.
- Client target paths continue to resolve from the process `HOME`; `--home` never changes them.
- `recover` accepts `--home` but rejects `--config`, because it never reads a manifest.

For example, an external manifest can use an isolated MCM state root:

```bash
mcm --home /absolute/mcm-root --config /absolute/manifest.yaml init
mcm --home /absolute/mcm-root --config /absolute/manifest.yaml server add --name local --command node
```

## Targets

| Target | Default output |
|---|---|
| `cursor` | `~/.cursor/mcp.json` |
| `claude-code` | `~/.claude.json` |
| `codex` | `~/.codex/config.toml` |
| `vs-code` | Requires manifest `targets.vs-code.path` or `--path` |
| `qoder-cli` | `~/.qoder/settings.json` |
| `qoder-ide` | `~/.qoder/mcp.json` (file rendering only; runtime loading is unverified) |
| `opencode` | `~/.config/opencode/opencode.json` or its sole existing `.jsonc` sibling |
| `mcp-cli` | `~/.config/mcp/mcp_servers.json` |
| `mcpc` | Selected MCM root `exports/mcpc.json` |

After applying `mcpc`, MCM prints a shell-quoted `mcpc connect <file>:<server>` hint. `mcp-cli` and Qoder IDE status remain file-only: MCM does not claim either client has loaded the file.

## Configuration example

```yaml
version: 1
servers:
  filesystem:
    transport: stdio
    command: npx
    args: ["-y", "@modelcontextprotocol/server-filesystem"]
  remote-tools:
    transport: streamable-http
    url: https://example.test/mcp
targets:
  vs-code:
    path: /absolute/path/to/mcp.json
```

## Safety and recovery

- MCM uses descriptor-bound no-follow file operations and owner-only permissions for its manifest, state, journal, and replacements.
- It rejects unmanaged native entries with the same server name rather than claiming ownership.
- `recover` only reconciles MCM ownership state from its journal; it never writes a native client configuration.
- MCM rechecks the manifest and target content before a write. A non-cooperating external writer can still modify a file after that last check and before rename; MCM warns on every apply and does not claim to eliminate that platform-level window.
- MCM reserializes selected native configuration files when it writes them. Existing formatting and JSONC comments may change; `plan` and `apply` warn before writing.

## License

MCM is available under the [MIT License](LICENSE).
