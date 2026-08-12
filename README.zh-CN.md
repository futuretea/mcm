# MCM

[English README](README.md)

MCM 是一个本地 Go CLI，用于管理一份公开的 MCP server 清单，并在 macOS 和 Linux 上将其渲染为多个 MCP 客户端的用户级配置。

它支持 Cursor、Codex、Claude Code、VS Code、Qoder CLI、Qoder IDE、OpenCode、[philschmid/mcp-cli](https://github.com/philschmid/mcp-cli) 和 mcpc。

## 范围

- MCM 清单默认位于 `~/.mcm/config.yaml`。
- 支持的传输方式：`stdio` 与 `streamable-http`。
- 拒绝：旧版 SSE、WebSocket、`cwd`、`env`、headers、OAuth 和 credential 字段。
- MCM 不会创建客户端目录；客户端目标文件的父目录必须已经存在。
- 直接目标文件不能是符号链接或其他非普通文件。

MCM 仅接受公开的 `command`、`args` 和 `url` 值。不要在清单中放入凭据或 token。

## 安装与构建

仓库通过 `.tool-versions` 固定 Go 1.26.5。

```bash
go build ./cmd/mcm
```

## 首次使用

```bash
mcm init
mcm server add --name filesystem --command npx --arg -y --arg @modelcontextprotocol/server-filesystem
mcm server add --name remote-tools --url https://example.test/mcp
mcm validate
mcm server list
mcm plan --target cursor --target codex
mcm apply --target cursor --target codex --yes
mcm status --target cursor --target codex
```

`plan`、`apply` 和 `status` 至少需要一个显式的 `--target`。重复的 target 会按稳定顺序去重。

交互式 `apply` 会请求一次 `yes` 确认。非交互环境必须传入 `--yes`。

可使用 `mcm --help` 或 `mcm <command> --help` 查看命令和参数。`server add` 只创建新名称；替换已有 server 定义请使用 `server update`。

## 全局清单路径参数

全局参数位于命令之前：

```text
mcm [--home DIR] [--config FILE] <command> ...
```

- `--home DIR` 选择 MCM 的私有根目录，用于 state、journal、lock、exports 和默认清单。它必须是绝对路径。
- `--config FILE` 选择一份精确的清单文件。它必须是绝对路径，且可与 `--home` 一起使用。
- 客户端目标路径仍从进程的 `HOME` 解析；`--home` 永远不会改变它们。
- `recover` 接受 `--home`，但拒绝 `--config`，因为它从不读取清单。

例如，可以让外部清单使用隔离的 MCM state 根目录：

```bash
mcm --home /absolute/mcm-root --config /absolute/manifest.yaml init
mcm --home /absolute/mcm-root --config /absolute/manifest.yaml server add --name local --command node
```

## 目标客户端

| Target | 默认输出位置 |
|---|---|
| `cursor` | `~/.cursor/mcp.json` |
| `claude-code` | `~/.claude.json` |
| `codex` | `~/.codex/config.toml` |
| `vs-code` | 需要清单中的 `targets.vs-code.path` 或 `--path` |
| `qoder-cli` | `~/.qoder/settings.json` |
| `qoder-ide` | `~/.qoder/mcp.json` |
| `opencode` | `~/.config/opencode/opencode.json`，或唯一存在的 `.jsonc` 同级文件 |
| `mcp-cli` | `~/.config/mcp/mcp_servers.json` |
| `mcpc` | 所选 MCM 根目录的 `exports/mcpc.json` |

应用 `mcpc` 后，MCM 会输出 shell 转义的 `mcpc connect <file>:<server>` 提示。mcp-cli 和 Qoder IDE 的 status 仅检查文件：MCM 不会声称客户端已经加载该文件。

## 配置示例

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

## 安全与恢复

- MCM 对其清单、state、journal 和替换文件使用基于文件描述符的禁止跟随操作，以及仅所有者可访问的权限。
- 如果原生配置中存在同名但未由 MCM 管理的 server，MCM 会拒绝写入，而不会声称拥有它。
- `recover` 只根据 journal 调和 MCM 的所有权 state；它绝不会写入原生客户端配置。
- MCM 会在写入前重新检查清单和目标文件内容。不合作的外部写入者仍可能在最后一次检查后、rename 前修改文件；MCM 会在每次 apply 时警告，且不声称能够消除这个平台级窗口。
- MCM 写入时会重新序列化选中的原生配置文件。已有格式和 JSONC 注释可能变化；`plan` 与 `apply` 会在写入前给出警告。
