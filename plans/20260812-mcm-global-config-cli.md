# MCM 全局 MCP 配置 CLI 实施计划

## 产物状态

| 字段 | 值 |
|---|---|
| 产物状态 | `review-passed` |
| 审查门禁 | `pass` |
| Skill 标注 | `complete` |
| 最新审查证据 | .auto-runs/v2/review-mcm-contract-r7.md; fresh full-scope 独立审查，proposal/plan 均通过，P0/P1/P2 均为 0。 |
| 阻塞项 | `none` |

## 概述

本轮在空仓库中交付 macOS/Linux 的 Go CLI：以 `~/.mcm/config.yaml` 管理一套公开的 MCP
服务器定义，并在预览和确认后更新九个客户端的用户级配置。它不支持 Windows、项目级配置、
credential 引用、`cwd`、旧 SSE 或 WebSocket。实现从 T1 的清单校验开始；首个检查是
`go version`，随后是 `go test ./internal/manifest`，最终使用隔离 `HOME` 执行 `go test ./...`
与 `go vet ./...`。

## 实现入口

| 项 | 内容 |
|---|---|
| 当前切片 | 一个未发布的本地 `mcm` CLI，包含初始化、服务器管理、九个适配器、preview/apply/status/recover、隔离测试与一轮独立安全审查。 |
| 允许修改范围 | 新建 `go.mod`、`.tool-versions`、`cmd/mcm/`、`internal/`、`README.md`、直接相关的 Go 测试与本计划。 |
| 禁止触碰范围 | 真实 `~/.mcm`、真实客户端配置、客户端安装、Windows、项目配置、OAuth/secret/env/header/cwd 支持、发布与 Git 提交。 |
| 入口任务 | T1：清单与 CLI 输入边界。 |
| 工具链 | `go.mod` 声明 Go `1.26.0` 与 `toolchain go1.26.5`；`.tool-versions` 固定 `golang 1.26.5` 供本仓库的 asdf launcher 选择。其他 Go launcher 依 `go.mod` 的标准 toolchain 规则。 |
| 首个验证命令 | 创建上述两个版本声明后，先运行 `go version`，再运行 `go test ./internal/manifest`。 |
| 阻塞条件 | 当前厂商文件无法按已有调研解析；发现 target 不存在的 parent、symlink/non-regular target、未管理同名项、journal conflict；或出现新的凭据字段需求。 |
| 最小实现约束 | CLI、JSON、hash、锁、临时写入与大多数 fsync 使用标准库；只增加 YAML、TOML 与 Go 官方 `golang.org/x/sys/unix v0.47.0`。后者仅供 macOS/Linux 的 descriptor-bound `openat`/`O_NOFOLLOW`/`fstat`/`renameat` 文件安全适配器。JSONC 用内部小型状态机，不用正则。只暴露 `cmd/mcm`，全部业务包留在 `internal/`。 |

## 系统架构图

```text
cmd/mcm
  │ parse flags / TTY prompts
  ▼
internal/manifest ── validate ──┐
                                  ▼
internal/app ── plan ──> internal/adapter ──> native JSON / JSONC / TOML
  │                    ▲                 │
  │                    │                 ▼
  └──> internal/store ─┴── state + journal + private atomic files
                  │
                  └── MCM lock / recovery / target preconditions
```

`internal/adapter` 是当前切片唯一需要稳定的内部边界：它接收已校验服务器和已有原生
文档，返回新文档、受管名称集合与无值变更摘要。`internal/manifest` 与 `internal/store` 的
descriptor-bound file operation seams 仅为测试和 macOS/Linux G4/G7 实现服务，不是插件或 public Go API。两者都不对
`internal/` 之外公开。

## 目录结构

```text
cmd/mcm/
  main.go                 # 进程入口与退出码映射
internal/
  cli/                    # 子命令、flags、TTY 引导和文本输出
  manifest/               # YAML 模型、校验、私有清单读写
  adapter/                # target registry、JSON/JSONC/TOML renderer 和路径
  app/                    # plan、apply、status、recover 编排
  store/                  # 私有 MCM 文件、lock、原子写入和 journal recovery
```

| 目录 | 职责 | 稳定性 |
|---|---|---|
| `cmd/mcm/` | 进程入口与依赖组装 | 低 |
| `internal/cli/` | 用户输入和可读输出 | 中 |
| `internal/manifest/` | 全局清单的结构与输入不变量 | 中 |
| `internal/adapter/` | 每个原生客户端契约与值保留 | 中 |
| `internal/app/` | 命令流程与确认边界 | 中 |
| `internal/store/` | 本地持久化、锁与恢复 | 低 |

## 模块职责

### `internal/manifest`

- 输入：`config.yaml` 文本或 `server add` 已归一化字段。
- 输出：已验证的服务器和 target 设置，或不包含原文值的错误。
- 规则：只接受 `stdio` / `streamable-http`；拒绝 `sse`、`ws`、`cwd`、`env`、`headers`、OAuth
  和未知服务器字段。`command`、`args`、`url` 是调用者保证公开的 opaque 输入，不做 secret
 词形猜测。`init` 只创建新的 `config.yaml`、以 `0600` 写入且遇既有文件拒绝覆盖；每次成功的
 `server add` 清单重写同样保持 `0600`，并在 MCM lock 内完成 read-modify-write。`--home` 与 `--config` 的解析只影响 MCM 清单或其私有
 state 根，不会改变 native target 的 home。

### MCM location flag matrix

全局 flag 位于 command 之前，形式为 `mcm [--home DIR] [--config FILE] <command> ...`。`--home DIR`
与 `--config FILE` 都必须是绝对路径。`--home` 选择 MCM 私有根（默认由当前用户 home 推导的
`~/.mcm`），因而决定 `state.json`、`journal`、`exports`、`lock` 与未覆盖的 `config.yaml`；
`--config` 仅选择精确的 manifest 文件。两者可组合：先由 `--home` 选择私有根，再由 `--config`
覆盖 manifest 路径。native target 的默认路径始终只由进程的用户 home 推导，绝不由 `--home`
或 `--config` 推导；mcpc 则是 MCM private export，固定在 B 的 `exports/mcpc.json`，不属于 A
下的 native target。

| command | `--home` | `--config` | manifest write |
|---|---|---|---|
| `init` | accepts | accepts | create exact selected config only; existing config fails without overwrite |
| `server add` | accepts | accepts | rewrite exact selected config only after successful validation/confirmation; final mode `0600` |
| `validate`, `server list`, `plan`, `apply`, `status` | accepts | accepts | none (`apply` may write native/state/journal, never config; plan/apply/status require at least one `--target`) |
| `recover` | accepts | rejects | none |

T1/T4/T5 integration uses native `HOME=A`, MCM private root `B`, and external manifest `C`: the complete
`mcm --home B --config C init → server add → validate → list → plan → apply → status` sequence mutates the
manifest only at C, keeps state/journal/lock/exports only under B, leaves no A/.mcm, resolves direct adapter
defaults beneath A, and writes mcpc only at B/exports/mcpc.json. `recover --home B --config C` rejects before
reading or writing C. A companion case with C's immediate parent absent fails without creating it, while init may
create only B. Under a permissive umask, `config.yaml` is `0600` after init and each successful add; init refuses
an existing file, while cancelled or invalid add leaves it byte-identical.

`plan`、`apply` 和 `status` 省略 `--target` 时，在读取任何 native path 前失败；重复 target 按首次
出现去重并按 target registry 的稳定顺序处理。`init` may create only the selected MCM root (default or `--home`) plus its `journal` and `exports` children;
each must be an ordinary non-symlink directory. It never creates an external `--config` parent. For every selected
manifest path, `init` accepts only an absent final file with an existing immediate ordinary, non-symlink parent;
it creates it exclusively at `0600`. Reading and `server add` accept only an ordinary, non-symlink final file.
`internal/manifest` opens the immediate parent through `golang.org/x/sys/unix.Open` with
`O_DIRECTORY|O_NOFOLLOW`, and uses that descriptor for `Openat` with `O_NOFOLLOW` plus post-open `Fstat` regular
file validation. Add creates its `0600` temp via the same descriptor, then file-fsyncs, `Renameat`s and directory-
fsyncs. A pre-rename fault leaves the old bytes unchanged; after a completed rename the file is valid YAML and
`0600` even if the following directory sync reports an error. T1 injects temp/write/file-sync/rename/directory-
sync faults and swaps the final entry to a symlink-to-sentinel after preflight, proving no redirected read/write or
corrupt manifest. This is the G7 private-file baseline, not a fallback.

### `internal/adapter`

- 输入：服务器清单、target、已存在原生文件的 bytes、MCM ownership state。
- 输出：可解析的新原生文档、每个受管服务器的 digest、`add` / `update` / `remove` 名称摘要，
  或不含原生值的错误。
- 边界：JSON/JSONC 以 `json.Decoder.UseNumber` 解码；未管理字段和值以解码语义保留，数字保持
  原始词法；空白、对象键顺序和 JSONC 注释不保证保留。`json.RawMessage` 只可暂存片段，不能作为
  byte-identical 序列化承诺。OpenCode JSONC 状态机准确跳过 string 与 escape 中的注释符号和尾逗号，
  输出严格 JSON。TOML 仅修改 `mcp_servers`，保留其他语义值。

### T2 adapter contract oracle

下表是 T2 golden fixture 的唯一格式依据。每个 fixture 文件名以 target 和 transport 命名，
并在测试 case 中引用本表的 target 与 source ID；不得由测试作者另行发明字段。`命令/参数` 与
`URL` 在 fixture 中使用公开的虚构值。`type` 列出的字段必须写出；`—` 表示该最小格式不写
transport 字段。所有行都拒绝 `sse`、`ws`、`cwd`、`env`、`headers` 和 OAuth 输入。

| target | source / confidence | path and root | stdio output | Streamable HTTP output | mode / status limitation |
|---|---|---|---|---|---|
| cursor | 1 / medium | `~/.cursor/mcp.json`; `mcpServers` | `type:"stdio"`, `command`, `args` | `url` | direct file |
| claude-code | 3 / medium | `~/.claude.json`; `mcpServers` | `type:"stdio"`, `command`, `args` | `type:"http"`, `url` | direct file |
| codex | 4, 5 / medium | `~/.codex/config.toml`; `mcp_servers.<name>` | `command`, `args` | `url` | direct TOML |
| vs-code | 2, 14 / medium | explicit profile `mcp.json`; `servers` | `type:"stdio"`, `command`, `args` | `type:"http"`, `url` | explicit `--path` or manifest path; no profile discovery |
| qoder-cli | 6, 7 / medium | `~/.qoder/settings.json`; `mcpServers` | `command`, `args` | `type:"http"`, `url` | direct file |
| qoder-ide | 12, 13 / low | `~/.qoder/mcp.json`; `mcpServers` | `command`, `args` | `url` (IDE detects Streamable HTTP) | file-only status; installed IDE verification remains manual |
| opencode | 8, 9 / high | unique `~/.config/opencode/opencode.json(c)`; `mcp.servers` | `type:"local"`, `command:[command,...args]` | `type:"remote"`, `url` | direct JSON/JSONC file |
| mcp-cli | 11 / medium | `~/.config/mcp/mcp_servers.json`; `mcpServers` | `command`, `args` | `url` | file/export only; advise `mcp-cli --config <file>` |
| mcpc | 10 / medium | selected MCM root `exports/mcpc.json`; `mcpServers` | `command`, `args` | `url` | export only; sorted `mcpc connect <file>:<name>` hints |

测试还必须逐行断言：原有未管理字段未改变，受管条目不产生本轮拒绝的结构化 credential
字段，Qoder IDE 只断言该文件的状态而不声称进程已加载。若任一 source 更新导致字段不一致，
先更新研究包与本表，重新审查对应 adapter，再改 renderer。

### `internal/store`

- 输入：MCM home、已打开的 native target parent descriptor、计划的 old/new state 与 desired bytes。
- 输出：持久化的 state/journal、原子替换结果或 recovery conflict。
- 规则：`~/.mcm`、`journal`、`exports` 目录为 `0700`；lock/state/journal/temp/new target 为
  `0600`；既有 target mode 若不宽于 `0600` 则保持，否则收紧到 `0600`。`init` 只创建 MCM
  目录树，native parent 缺失、symlink 或非目录时 fail-fast。每个 direct target 的读取、OpenCode
  `.json` / `.jsonc` resolver、apply 最终 digest recheck 和 replace 共用 FD-relative primitive：
  parent 以 `O_DIRECTORY|O_NOFOLLOW` 打开，已有 final entry 以 `openat(O_NOFOLLOW)` 加 `fstat`
  验证 regular file，不存在的 entry 只在该 parent descriptor 创建；temp/rename/fsync 也经同一
  descriptor。测试 hook 可在 preflight 后替换 final entry 为 sentinel link，操作必须不跟随、
  不输出或持久化 sentinel。

### `internal/app`

- 输入：命令选择、已验证 manifest、adapter registry、store 与 confirmation reader。
- 输出：value-free plan、目标文件状态或命令错误。
- 规则：`plan` / `status` 不写任何文件；有 journal 时仅报告 recovery-required 或 conflict。
  `recover` 只更新 MCM state/journal，永不写 native config；`apply` 在新计划前 recovery，
  之后确认、全局 lock、重新校验 manifest 与 native digest、写 journal/native/state。apply 的
  preview 记录 descriptor-bound manifest digest；取得 lock 后重读该 manifest，不同时在任何
  journal/native/state 写入前终止。`server add` 在同一 lock 内完成 manifest read-modify-write，
  因而共享 MCM root 的 add/apply 不会提交过时 plan。

### `internal/cli`

- 输入：标准输入、标准输出/错误、flags。
- 输出：确定的命令文本和退出码。
- 规则：`apply` 在 TTY 显示无值摘要与 L1 警告后确认；非 TTY 要求 `--yes`。`server add`
  缺参仅在 TTY 中逐项引导，取消不写清单。`plan` / `apply` / `status` 至少一个 `--target`，重复
  target 去重；`--path` 最多一次、绝对路径且只搭配一个 target。

## 核心流程

```text
manifest -> validate -> adapters read/render -> value-free plan -> confirm
                                                  │                   │
                                             no native write       MCM lock
                                                                      │
                           journal fsync -> native fsync -> state fsync -> journal remove
```

1. 所有命令先解析参数；`plan` / `apply` / `status` 缺少 target，或任何清单/路径输入错误，都在读取 native 文件前返回。
2. `plan` 只通过 FD-relative no-follow target reads 校验 ownership 和 native 语法，输出名称级摘要；不会输出不受管值。
3. `apply` 先从持久化 journal 恢复 MCM 自身 state，再构建完整计划并确认。
4. 确认后取得 `~/.mcm/lock`，重读并比较 manifest digest；不一致即终止。每个 target 通过相同
   parent descriptor 重新读取 digest；可观察 drift 则终止，不写任何 target。
5. 写入顺序是：fsync intent 与 journal 目录、fsync native temp/rename/parent 目录、fsync new
   state/rename/MCM 目录、remove intent/fsync journal 目录。任何不匹配由 `recover` 报 conflict。
6. 不参与 lock 的外部写者仍可能落在最终重验与 rename 的极小窗口；这是用户已接受的 L1，
   每次 `apply` 都显示该警告，不能测试或宣称其会被阻断。

## 扩展点设计

不适用。九个当前 adapter 固定注册，后续 client、credential 引用与 Windows 支持必须在
独立 enhancement 中重新审查；本轮不建立 plugin、registry API 或运行时开关。

## 测试策略

| 范围/任务 | 行为类型 | TDD 结论 | 受测 seam（Red 观察边界） | 首个失败测试 / 最小验证场景 | Red 命令 | 最小可运行检查 | 验证层次 | 进程外依赖分类 | 运行路由 / 命令 |
|---|---|---|---|---|---|---|---|---|---|
| T1 | 新功能 / 输入校验 | tdd-required | `cli.Run(args, IO, Env, filesystem)` 与 `manifest.Load/Write` 内部行为边界 | `HOME=A`、MCM root B、external C 的 combined lifecycle；仅 B root 可由 init 创建、mcpc only B/export；config 每次 `0600`；post-preflight manifest link swap 不读写 sentinel；缺少 target、`sse`、`cwd`、`env` 和未知字段均被拒绝且不读取 native 路径 | `go test ./internal/manifest ./internal/cli -run 'Test(Init|Manifest)'`（先因缺 seam/行为失败） | `go version`、flag matrix、descriptor-bound config resolution、YAML read/validate 分支 | unit + integration | 集成真实（临时 HOME） | `go version && go test ./internal/manifest ./internal/cli` |
| T2 | 新功能 / 语义转换 | tdd-required | `adapter.Plan(manifest, existing, state)` 的 render/summary 结果 | 上文 oracle 每个 target 的 stdio/HTTP golden；OpenCode JSONC string/escape 与 `UseNumber` semantic preservation | `go test ./internal/adapter -run 'TestPlan'`（先因 renderer 缺失失败） | JSON/JSONC/TOML parse-render | unit | 单元 mock | `go test ./internal/adapter` |
| T3 | 新功能 / 持久化状态 | tdd-required | `store.Commit/Recover` 与 `store.TargetFile` 的注入式 FD/file-op seam | intent 成功、native rename 后 state 失败时 exact newState recovery；default/`--path` target and OpenCode resolver link swaps do not touch sentinel | `go test ./internal/store -run 'Test(Commit|Recover|TargetFile)'`（先因 store 行为缺失失败） | fsync/rename/permission/final-entry no-follow boundaries | unit + integration | 集成真实（临时目录） | `go test ./internal/store` |
| T4 | 新功能 / 用户操作 | tdd-required | `app.Run` 的 plan/apply/status/recover 结果，注入 confirmation reader 与 store | missing target, cancelled apply、non-TTY without `--yes`、native/manifest drift、unmanaged conflict 均无 native write/state/journal | `go test ./internal/app ./internal/cli -run 'Test(Apply|Status)'`（先因 orchestration 缺失失败） | command orchestration | integration | 集成真实（临时 HOME） | `go test ./internal/app ./internal/cli` |
| T5 | 端到端验收 | tdd-required | 编译的 `cmd/mcm` 子进程（temp `HOME`、stdin/stdout、exit code） | 临时 HOME 内 init/add/plan/apply/status/recover 覆盖九 target；S1 reports zero P0/P1 | `go test ./cmd/mcm -run TestCLIWorkflow`（先因 process behavior 缺失失败） | process-level CLI flow | integration | 集成真实（临时 HOME，无 MCP/client 网络请求） | `go test ./... && go vet ./...` |

测试 fixture 中只能使用 `fixture-only-sensitive-sentinel` 等明显虚构的标记，绝不放入真实
token、账号、个人路径或客户配置。所有 native target 均位于 `t.TempDir()` 下的临时 HOME。

## 任务规划

```text
T1 ─┬─> T2 ─┐
    └─> T3 ─┼─> T4 ─> S1 ─> T5
```

| ID | 任务 | 来源 | 范围 | 非目标 | 现成能力 | 复杂度边界 | 前置依赖 | Guard Policy | TDD | 验证方式 | 升级触发条件 | Skill 路由 |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| T1 | 建立固定 Go 工具链、清单和 CLI 输入边界 | Goal 1、3；D4、D7、D8、D9、D10；G3、G7、G9 | `go.mod`（Go `1.26.0`、toolchain `go1.26.5`、`x/sys v0.47.0`）、`.tool-versions`（`golang 1.26.5`）、`cmd/mcm`、`internal/manifest`、`internal/cli` | native 写入、adapter 渲染 | `flag`、`os`、YAML library、`x/sys/unix`、现有 asdf launcher | 不加 CLI framework、secret scanner、公共 Go API；只建供测试注入的内部 seam 和 private descriptor-bound file-op adapter | None | G3 block；G7 private config/atomic no-follow write; G9 opaque-input contract | required | `go version`; native `HOME=A`、MCM root B、external C 的 combined-flag lifecycle；mcpc in B/export；permissive-umask 下 init/add 后 `0600`、init 不覆盖、invalid/cancelled add byte-identical、post-preflight manifest symlink swap/fault injection；missing/duplicate target; validate；server list；TTY reader fake | 需要 credentials/cwd 时新提案；固定 Go toolchain 或 x/sys unix primitive 不可运行时停止并报告 | `tdd`、`cli-design` |
| T2 | 实现九个 adapter 与只读计划 | Goal 2、3；D2、D3、D5、D6；G2、G4、G8 | `internal/adapter` 与上文唯一 adapter contract oracle | 写入、客户端启动、profile 自动发现 | `encoding/json`、`json.Number`、TOML library | 固定 registry；不加 plugin；JSONC 仅 lexer 状态机；保留语义，不承诺 JSON fragment byte identity | T1 | G2/G4/G8 block | required | 每行 oracle 的 stdio/HTTP golden、`UseNumber` semantic preservation 与 JSONC no-write fixtures；Qoder IDE file-only assertion | 第十个 client、source/schema 漂移或 JSONC 完整编辑要求 | `tdd` |
| T3 | 实现私有存储、锁与可恢复 journal | Goal 3、5、6；D1、D10；G4、G6、G7、G8 | `internal/store` | 全局事务、后台恢复、远程存储 | `os`、`x/sys/unix`、SHA-256、临时目录 | 一把 MCM lock；不加数据库/daemon；故障注入仅内部测试 seam；direct target 的 read/recheck/replace 共用 FD-relative no-follow 原语 | T1 | G4/G6/G7/G8 block | required | recovery state table、mode、sync-boundary、default/`--path`/OpenCode link-replacement tests | 多进程/跨机器协调需求 | `tdd` |
| T4 | 组装 plan/apply/status/recover 体验 | Goal 2–6；D9；G1、G2、G5；L1 | `internal/app`、`internal/cli` | 客户端运行状态检测、自动回滚 | T1-T3 的内部边界 | 不新增 second confirmation、silent fallback 或全局 native transaction | T2、T3 | G1/G2/G5 block；L1 manual warning | required | missing/duplicate target、confirmation、native/manifest drift、mcp-cli/Qoder file-only status tests、D9 flag matrix | 用户要求 client runtime probe | `tdd`、`cli-design` |
| S1 | 独立安全专项审查 | G2、G4、G6、G7、G8；D10 | 只读 `internal/manifest`、`store`、`adapter`、`app`、`cli`、`cmd/mcm` 与关联测试；输出独立 findings report | 生产修改、附加 guard 或测试修复 | `review-security` deep contract | 不渗透、无攻击 payload、不改实现、不以测试/vet 代替专项审查 | T4 | pass 需验证 manifest 和 default/`--path`/OpenCode target 的 descriptor-bound no-follow safety、non-regular preflight、manifest recheck、0600/0700、atomic failure、recover no-native-write、no-sentinel output；P0/P1 先按 admission 闭合 | required independent review | report with zero P0/P1 or revised plan/fix loop | fix 后再次 scoped independent review | `review-security`、`auto-agent` |
| T5 | 端到端验证与中文 README | Goal success signals；G1–G9 | `README.md`、全套测试 | 发布、安装脚本、真实 client test | `testing`、临时 HOME、Go vet | MCM/client runtime 不连接网络；Go module download 只允许在开发 bootstrap，测试阶段使用已解析 module cache | S1 | G1–G9 coverage audit；L1 documented；S1 pass evidence | required | full test/vet、manual command smoke with temporary HOME、S1 finding report | 官方格式漂移或 Go toolchain incompatibility | `tdd` |

## 后续增强

- 独立设计环境/HTTP header 的安全引用与 OAuth。
- Windows、项目级配置和客户端运行态探测。
- 已发布清单版本的迁移工具与可选客户端安装。

## 风险、假设与回退

| ID | 类型 | 触发条件 | 影响范围 | 检测方式 | 回退 / 降级动作 | 关联任务 |
|---|---|---|---|---|---|---|
| R1 | 已接受限制 | 非协作编辑器在 final recheck 后写文件 | 单个 native target 的最后窗口 | `apply` 通用警告；事后状态可能 drift | 用户重新 `plan`；MCM 不承诺恢复该写入 | T4，L1 |
| R2 | 假设 | Qoder IDE 实例不加载 `~/.qoder/mcp.json` | Qoder IDE runtime | status 仅文件状态，用户在 IDE 验证 | 不宣称运行时同步；调整 target 需新证据 | T2、T4 |
| R3 | 风险 | OpenCode `.json` 与 `.jsonc` 同时存在或 JSONC malformed | OpenCode target | path/parse error | 要求单 target `--path`，不写入 | T2、T4 |
| R4 | 风险 | parent 缺失、symlink/non-regular、preflight 后 final-entry link replacement、manifest digest 漂移或 lock 被占用 | 任意 manifest / native target | descriptor-bound open 或 commit-time recheck 失败 | 不写 native/journal/state；用户恢复普通 entry、重新 plan 或重试 | T1、T3、T4 |
| R5 | 假设 | 当前 macOS/Linux filesystem 支持 `fsync` directory | durable journal ordering | `Sync` 返回错误 | 停止后续步骤，保留 intent；下次 `recover` 或报告 conflict | T3 |

## 实现指引

1. **选择单一 manifest + per-target adapter，不选择共享原生文件**：客户端根字段和格式不同；若
   不能解析某一原生文件，返回无值错误并不写入。
2. **选择 value-free 摘要，不选择 unified diff**：保留 native 未管理项时完整 diff 可能泄漏值；
   若用户需要查看值，必须直接检查自己的本地文件。
3. **选择 journal 的 exact state，不选择从当前 manifest 重建**：crash 后 manifest 可变化；若
   digest/state 不完全对应 recovery table，保持冲突而不猜测。
4. **选择要求现有 client parent 与 FD-relative target I/O，不选择递归创建或路径 reopen**：MCM 只
   拥有 `~/.mcm`；若用户先创建了正常 client 目录，再重新运行 plan/apply；final entry 被换成 link
   时阻断而不跟随。
5. **选择用户负责 opaque public inputs，不选择 secret scanner**：无法可靠自动识别任意 argv/URL；
   若出现凭据引用需求，停止并新建设计，而不是把值写入清单。

## 质量自检清单

- [x] 当前切片、入口任务、首个验证命令和阻塞条件可扫描。
- [x] 当前任务 DAG 有 T1–T5 与 T4 后的独立 S1 门禁，按依赖排序。
- [x] G1–G9 与 L1 都映射到实现或验证任务。
- [x] 每个非平凡任务都有 TDD 结论和最小检查。
- [x] 当前范围不含 Windows、凭据字段、项目级配置或发布。
- [x] 风险、假设与回退集中在唯一章节。
- [x] 没有 HTML 注释、占位符或未填写模板字段。
