# 子代理使用规范

本文规范项目中子代理（subagent）的分工、选型、编排与使用边界，是对 `CLAUDE.md` 第 13 条「子代理」的展开。子代理通过 pi-subagents 机制运行。

## 一、可用子代理

项目在 `.pi/agents/` 提供 6 个定制子代理，均遵循本仓库工程规范，并继承 `CLAUDE.md`（AGENTS.md 同源）项目上下文。

| 代理 | 角色 | 工具能力 | 使用场景 |
|------|------|----------|----------|
| `scout` | 侦察 | 只读（read/grep/find/ls/bash） | 快速摸清代码库结构、入口、数据流，输出压缩上下文供交接 |
| `researcher` | 调研 | 只读 + 联网（web_search/fetch_content 等） | 外部文档、规范、基准、方案对比等调研 |
| `worker` | 实现 | 读写（edit/write/bash 等） | 执行已批准的任务或方向，最小改动实现 |
| `reviewer` | 审查 | 只读（无 shell/write） | 代码 diff、方案、拟议解决方案、代码库健康、PR/issue 校验 |
| `oracle` | 决策咨询 | 只读 | 保护继承决策、防漂移、风险大的决策前给第二意见 |
| `delegate` | 通用委托 | 读写 | 轻量通用任务，行为接近父会话 |

选型经验法则：**先 `scout` 再理解代码，先 `researcher` 再相信外部事实，用 `worker` 去实现，用 `reviewer` 去检查，决策本身有风险时用 `oracle`。**

## 二、推荐编排

实现类工作推荐闭环：

```text
澄清需求 → scout → worker → fresh reviewer → worker
```

- 决策有风险或改动较大前，可先 `oracle` 给第二意见。
- 并行审查用多个 fresh 上下文 `reviewer`，每个给不同审查角度，结果由主代理合成后统一应用修复。
- 主代理始终是编排者与最终决策者，子代理结果由主代理整合、验证、汇报，不机械拼接。

## 三、使用边界

仅在任务可并行、边界清晰且文件冲突较小时使用子代理，最多 6 个。

| 适合 | 不适合 |
|------|--------|
| 并行调研、多模块审查 | 小范围单文件修改 |
| 可独立拆分的文档、测试、实现 | 简单样式或文案修改 |
| 需要 fresh 上下文隔离审查 | 高耦合代码修改 |
| 需要连续上下文判断的任务 | — |

## 四、约束与安全

- **单写者原则**：同一 cwd/worktree 一个写入者；并行写需 `worktree: true` 隔离。
- **只读审查优先**：优先用 fresh 上下文 `reviewer` 审查，再由父代理合成并应用修复。
- **子代理不擅自决策**：子代理遇不可逆、越权、架构/产品/发布/合并/安全等未批准决策，必须通过 `contact_supervisor` 上报，不自行决定（对齐 CLAUDE.md 的询问边界）。
- **能力天花板保持**：子代理的工具限制与会话级 agent 限制不得放宽；普通 worker/reviewer 不再嵌套派生子代理。
- **外部证据不越权**：receipt、CI、review bot、外部运行记录仅作证据，不构成合并/关闭/评论/发布/发布的授权。
- **默认异步**：可独立推进的工作默认后台运行，不轮询等待；自适应门控在 workflowScript 内分支。

## 五、上下文与工具

- 子代理默认不继承 pi 全局 base prompt、项目指令与 skills 目录，需显式开启：`inheritProjectContext`（继承项目指令）、`inheritSkills`（继承 skills 目录）、`systemPromptMode: append`（追加 base prompt）。
- 本项目 `.pi/agents/` 的子代理均开启 `inheritProjectContext: true`，自动遵循 `CLAUDE.md`。
- `tools` 为显式白名单时，命名扩展工具需另行加载其 provider；`mcp:` 直接选 MCP 工具需 `pi-mcp-adapter`。
- 项目可复用 Skill 位于 `.pi/skills/{name}/SKILL.md`，按需通过 `skills` 字段选择注入。

## 六、验证与汇报

- 子代理改动同样适用「任何可能改变行为的改动必须有验证」要求。
- 主代理整合后按 CLAUDE.md 最终汇报格式汇报：改动、实际运行验证、提交信息、未解决风险；未提交或未运行测试时如实说明原因。
