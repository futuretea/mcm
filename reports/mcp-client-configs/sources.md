# Sources

1. [Cursor MCP documentation](https://cursor.com/docs/mcp) - official documentation - accessed 2026-08-12 - medium; documents `~/.cursor/mcp.json`, `mcpServers`, and required `type: "stdio"` for a stdio entry.
2. [VS Code MCP configuration reference](https://code.visualstudio.com/docs/agents/reference/mcp-configuration) - official documentation - accessed 2026-08-11 - high.
3. [Claude Code MCP documentation](https://code.claude.com/docs/en/mcp) - official documentation - accessed 2026-08-11 - high.
4. [Codex MCP documentation](https://developers.openai.com/codex/mcp/) - official documentation - accessed 2026-08-11 - high.
5. [Codex configuration reference](https://developers.openai.com/codex/config-reference/) - official documentation - accessed 2026-08-11 - high.
6. [Qoder CLI MCP servers](https://docs.qoder.com/cli/mcp-servers) - official documentation - accessed 2026-08-11 - high.
7. [Qoder CLI MCP reference](https://docs.qoder.com/cli/mcp-reference) - official documentation - accessed 2026-08-11 - high.
8. [OpenCode configuration](https://opencode.ai/v2/docs/config) - official documentation - accessed 2026-08-12 - high; defines the global configuration location and JSON/JSONC support for OpenCode v2.
9. [OpenCode MCP servers](https://opencode.ai/v2/docs/mcp-servers) - official documentation - accessed 2026-08-12 - high; OpenCode v2 nests named servers under `mcp.servers` and maps local and remote transports there.
10. [Apify mcpc README](https://github.com/apify/mcpc) - first-party source repository - accessed 2026-08-12 - high; documents `mcpc connect <config-file>:<server-name> [@session]` for an entry in a standard config file.
11. [philschmid/mcp-cli README](https://github.com/philschmid/mcp-cli) - first-party source repository - accessed 2026-08-11 - high.
12. User-confirmed project constraint: Qoder IDE output path is ~/.qoder/mcp.json - user-provided - 2026-08-11 - authoritative for MCM target selection, but needs installed-product validation.
13. [Qoder IDE MCP guide](https://docs.qoder.com/user-guide/chat/model-context-protocol) - official documentation - accessed 2026-08-11 - medium; confirms the mcpServers schema and supported IDE setup flow, not the filesystem location.
14. [Add and manage MCP servers in VS Code](https://code.visualstudio.com/docs/agent-customization/mcp-servers) - official documentation - accessed 2026-08-11 - high; documents the user-profile command and `code --add-mcp`.
15. [MCP 2026-07-28 Specification](https://blog.modelcontextprotocol.io/posts/2026-07-28/) - official MCP specification update - accessed 2026-08-12 - high; states that the legacy HTTP+SSE transport is deprecated and gives its migration off-ramp.
