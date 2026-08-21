---
name: project-engineering
description: 按本仓库工程规范安全地实施代码、配置、文档、依赖或构建改动。用于需要遵循项目约束、保护已有工作区、选择验证方式、维护 TODO.md、创建本地 Git 提交的开发任务。
---

# 项目工程规范

使用本 Skill 完成仓库内的工程改动。将仓库根目录视为本文件的三级父目录。

## 读取规范

1. 先完整读取 [主规范](../../../CLAUDE.md)。
2. 根据任务只读取必要的专题规范：
   * 前端：[frontend.md](../../../docs/development/frontend.md)
   * Rust：[rust.md](../../../docs/development/rust.md)
   * HarmonyOS：[harmonyos.md](../../../docs/development/harmonyos.md)
   * 前后端热加载：[hot-reload.md](../../../docs/development/hot-reload.md)
   * Python：[python.md](../../../docs/development/python.md)
   * CMake：[cmake.md](../../../docs/development/cmake.md)
   * Git：[git-workflow.md](../../../docs/development/git-workflow.md)
   * GitHub：[github.md](../../../docs/development/github.md)
   * 验证：[verification.md](../../../docs/development/verification.md)
   * 写作：[writing.md](../../../docs/development/writing.md)
   * 代码注释：[code-comments.md](../../../docs/development/code-comments.md)
   * 并发与线程：[concurrency.md](../../../docs/development/concurrency.md)
   * 接口文档：[api-documentation.md](../../../docs/development/api-documentation.md)
3. 查找当前目录及子目录中的更具体规范；其优先级高于通用规范。

## 根目录布局

* `app/` 存放项目主体代码。
* `docs/` 存放开发文档、开发日志和开发规范。
* `docs/api/` 按需存放实际接口文档，没有接口时不创建空目录。
* `.pi/skills/` 存放项目可复用的 Skill（pi-subagents 格式）。
* `tests/` 存放跨技术栈测试和验收测试；语言工具链测试放在对应 crate 或模块内。
* 新增文件应遵循上述布局；不创建空目录。
* 迁移或重命名现有目录前，必须先获得工程师同意。

## 实施流程

1. 说明目标、成功标准、修改范围与不修改范围。复杂任务给出不超过三步的计划。
2. 运行 `git status --short` 并检查相关差异。将已有未提交改动视为工程师的工作，禁止覆盖、回滚、格式化或混入提交。
3. 先复用项目已有实现、依赖、包管理器、测试和构建流程；只有在确有必要时新增依赖或工具。
4. 采用最小可验证改动，不顺手重构、升级无关依赖或扩展需求。
5. 根据改动类型执行相应验证，并如实记录实际结果。
6. 修改 [TODO.md](../../../TODO.md)：任务完成后标记完成，不删除历史事项。
7. 精准暂存本次文件，检查暂存差异后创建一个语义清晰的本地提交；未经工程师明确同意不得推送。

## 决策边界

遇到不可逆操作、重要数据迁移、外部接口或持久化格式变更、重大依赖引入、发布部署、敏感信息处理，或无法安全保留已有改动时，暂停并询问工程师。

对于局部且可逆的实现细节、可由现有测试快速验证的选择，或明显符合现有风格的方案，声明简短假设后继续。

## 验证选择

* 行为、构建、依赖、配置、数据格式和 UI 改动必须验证。
* 页面验证优先使用项目已有测试或验证脚本；没有可用脚本时提供可复现的手动步骤，并如实说明未完成自动化验证的范围。
* 纯说明文字、拼写或不参与执行的注释改动可免完整测试，但必须检查差异、链接与示例。
* 只报告实际运行过的验证；未运行或失败时说明原因和未覆盖范围。

## 工具与兼容性

* 优先查阅当前版本的官方文档；context7 不可用时依次使用官方文档、源码及类型定义、官方示例和发行说明。
* 在 Windows 下使用 UTF-8、正确引用含空格的路径，保留项目既有行尾风格，不混用 Windows 与 WSL 路径。
* 终端输出同时提供文字状态标识，例如 `[成功]`、`[警告]`、`[错误]`；CI、重定向或设置 `NO_COLOR` 时关闭颜色。

## 提交与汇报

提交前确认：改动可与已有工作区区分、完成最小验证、暂存区不含敏感或无关文件。提交信息使用中文 Conventional Commits，例如 `fix(scope): 修复具体问题`，不得提及 AI。

最终汇报改动、实际验证、提交信息和未解决风险；没有运行测试或没有提交时明确原因。
