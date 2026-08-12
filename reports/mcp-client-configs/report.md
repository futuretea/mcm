---
title: "MCP 多客户端配置兼容性调研"
date: "2026-08-11"
description: "面向 MCM 的客户端配置格式、路径和传输能力结论。"
---

# 结论

MCM 不应把一份 JSON 原样复制给所有客户端。目标产品至少有四种配置
根：标准 JSON 的 mcpServers、VS Code 的 servers、Codex 的 TOML
mcp_servers，以及 OpenCode 的 mcp.servers。它们对用户级路径、变量替换和远程
传输的定义也不同。[source: 1][source: 2][source: 4][source: 9]

首期规范仅接受 stdio 和 Streamable HTTP。旧 HTTP+SSE 在当前 MCP
规范中已废弃；即使个别客户端仍能读取它，MCM 也必须返回明确的迁移错误，
而不是生成、猜测或降级该传输。WebSocket 同样不在当前范围内。
[source: 4][source: 10][source: 15]

Qoder 是两个目标：

- qoder-cli 写入 ~/.qoder/settings.json 的 mcpServers。
- qoder-ide 写入 ~/.qoder/mcp.json 的 mcpServers。

第一项由 Qoder CLI 官方文档证明。第二项是用户确认的项目约束；官方
IDE 文档证明了 mcpServers 格式与 STDIO/HTTP 行为，但公开文档未
给出该文件的确切位置。因此 qoder-ide 的 `status` 只能报告文件状态；
用户须在已安装 IDE 中自行验证是否加载。[source: 6][source: 7][source: 12][source: 13]

## 对 MCM 的影响

1. 全局来源保持在 ~/.mcm/config.yaml，不作为任何客户端的原生文件。
2. 每个客户端使用独立 Adapter，负责读取、保留非 MCM 管理项、生成不含配置值的
   受管变更摘要和写入原生格式。
3. mcpc 是配置消费者，不是单一注册表：导出标准 mcpServers JSON，并为每个
   服务输出 `mcpc connect <file>:<server-name>`；不依赖自动发现。
4. philschmid/mcp-cli 应直接生成 mcp_servers.json。它依次搜索当前目录、
   ~/.mcp_servers.json 和 ~/.config/mcp/mcp_servers.json；全局适配器默认选择
   最后一个标准位置。该位置的 `status` 仅报告导出文件是否匹配，不能声称它是
   mcp-cli 实际加载的配置；用户可通过 `mcp-cli --config <file>` 显式选择它。
   [source: 10][source: 11]
5. OpenCode v2 的全局文件可为 opencode.json 或 opencode.jsonc。只有一个存在时
   MCM 自动选择它；两个都存在时，MCM 在读取内容前失败并要求单目标 `--path`。
   JSONC 输入重新写为严格 JSON，保留未修改字段和值，但不承诺保留注释或格式。
   [source: 8][source: 9]
5. VS Code 的用户级路径是 profile 抽象，官方没有承诺一个跨安装稳定的
   文件系统路径。适配器应优先调用 code --add-mcp，或要求用户提供
   --path；不能硬编码 Linux 或 macOS 的 profile 目录。[source: 2][source: 14]

## 凭据边界

MCM v1 不接受环境变量引用、HTTP header、OAuth 或任何凭据字段，因此既不
读取环境也不生成 token、API key 或 OAuth client secret 字段。`command`、`args` 与
`url` 是调用者保证公开的不透明输入，MCM 不做无法可靠证明的 secret 分类。客户端各自
支持的安全引用与凭据存储保留给后续、按目标适配器单独审查的增强；MCM 在本轮对结构化
凭据字段 fail-fast。[source: 2][source: 3][source: 4][source: 9][source: 10][source: 11]

## 限制

单个客户端的配置路径或字段通常只有一个权威供应商来源，故表格中的此类
事实为 medium。Qoder IDE 路径为 user-confirmed 且与公开 CLI 文档分离，
故为 low，直到真实安装验证。此调研不授权写入任何客户端配置。
