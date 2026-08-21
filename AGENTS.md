# NodePaper Agent

## 项目概况

NodePaper 是面向 Windows 的命令行工具，将包含 `nodepaper.yaml` 的 Markdown Project 构建为 PDF（Pandoc + XeLaTeX 工具链），当前面向 CUMCM 电子版论文场景；正在向桌面端（Tauri + React）扩展。

## 0. 规范优先级

在不违反安全、法律和平台限制的前提下，规则优先级如下：

1. 工程师在当前任务中的明确要求。
2. 当前目录或子目录中更具体的项目规范。
3. 项目已有代码、配置、测试和构建约定。
4. 本文件中的通用规范。
5. 通用行业惯例和工具默认行为。

规则冲突时，遵循范围更具体、距离当前任务更近的规则。高优先级要求可能造成数据损失、明显风险或不可逆结果时，应先说明风险。

## 1. 身份与交流

* 使用简体中文交流。
* 以开发者和科学工作者的标准拆解问题，明确假设、边界、风险与取舍。
* 回答和汇报应简洁、具体、可执行，不掩盖困惑、失败或未验证状态。

## 2. 执行原则

* 开始前明确目标、成功标准、修改范围、排除项和验证方式。
* 复杂任务给出简短计划；简单、低风险的局部任务可直接执行。
* 采用满足需求的最简单方案，不扩大范围，不为一次性需求或未来扩展增加抽象。
* 只修改与任务直接相关的内容，不顺手重构、格式化或清理无关代码。
* 遵循现有架构与代码风格；只清理本次改动产生的孤立代码。
* 注释应解释原因、约束与风险，不重复翻译代码行为。
* 默认使用单线程；仅按实际需求或测量结果引入有界并发，异步不等同于多线程。
* 发现无关问题时可以汇报，但不得擅自修改。

### 根目录结构

* `nodepaper/core/`：Go 核心，仓库唯一可执行产物来源。`cmd/nodepaper/` 是 CLI 入口；`internal/` 是业务包（app、artifact、build、buildctx、buildlock、cli、config、diagnostic、doctor、export、fonts、fragment、latexlog、output、process、profile、project、validate）；`tests/` 是单元、集成测试与夹具、语料。`go.mod` 在此目录（模块名 `nodepaper`，工具链由 go.mod 钉住），所有 `go` 命令在该目录下执行。
* `nodepaper/desktop/`：桌面端。`src/` React 前端；`src-tauri/` 极薄 Tauri 壳；`binaries/` 内嵌的 nodepaper-core 二进制。
* `nodepaper/shared/`：core 与 desktop 共享的契约与类型（预留）。
* `docs/`：`development/` 开发规范、`guides/` 用户与专题指南、`assets/` 图片与展示物料。
* `scripts/`：PowerShell 构建、测试与发布脚本；`scripts/build/Build-Paper.ps1` 是 LaTeX 编译入口。
* `profiles/`：论文 profile（当前仅 `cumcm/`，LaTeX 模板 + `profile.json`）。
* `packaging/`：安装包与工具链清单（Inno Setup、windows-x64 工具链钉版）。
* `CONTRIBUTING.md` 与 `CHANGELOG.md`：贡献指南与重大更改日志。
* `licenses/` 与 `THIRD_PARTY_NOTICES.md`：第三方许可，随依赖变更同步。
* `skills/`：项目可复用 Skill。
* `.github/workflows/`：`ci`（gofmt/vet/test/race/govulncheck，windows-latest）、`export-linux`、`miktex-e2e`、`release-build`。
* 新增文件应按上述结构放置；迁移、重命名或删除现有目录前必须取得工程师同意。
* 接口契约发生变化时，应在同一提交中同步更新实现、类型、测试和对应接口文档。

## 3. 工作区与已有改动保护

* 修改前检查 `git status` 和相关差异，将已有未提交改动视为工程师的重要工作。
* 不覆盖、删除、回滚或格式化与当前任务无关的已有改动。
* 文件已有改动时，在当前版本上做最小增量编辑。
* 撤销自己的错误时只撤销自己引入的改动，不回滚整个文件。
* 未经工程师明确同意，禁止执行：
  * `git reset --hard`
  * `git clean -fd` 或 `git clean -fdx`
  * `git restore .`
  * `git checkout -- .`
  * 强制推送
  * 修改或重写 Git 历史
* 修改后检查 `git diff`，确认没有意外改动。

## 4. 询问边界

以下情况必须暂停并询问工程师：

* 不可逆或高风险操作，以及删除、覆盖或迁移重要数据。
* 修改外部接口、持久化格式、公开协议或明显的产品行为。
* 新增重要基础设施、框架、数据库、构建工具或长期依赖。
* 推送、发布、部署、产生费用或访问及暴露敏感信息。
* 无法安全保留工程师已有改动。
* 多种方案会导致明显不同结果，且项目中没有选择依据。

以下情况可以简短声明假设后继续，并保持改动可逆：

* 局部实现细节和小范围函数拆分。
* 可由现有测试快速验证的选择。
* 不影响公开接口和架构的代码组织。
* 差异很小且其中一种明显符合现有风格的方案。

## 5. 工具与资料

* 查询库、框架和 API 时，优先查阅对应版本的官方文档。
* context7 可用时，可用于快速定位文档和示例；不可用或信息不足时，依次查看官方文档、官方源码及类型定义、官方示例及发行说明，再参考高质量社区资料。
* 不因某个特定工具不可用而停止任务。
* 读取文档时优先使用 MarkItDown，例如 `markitdown input.pdf > output.md`；不可用时使用能保真读取内容的现有工具。
* 页面验证优先使用项目已有测试或验证脚本；没有可用脚本时提供可复现的手动步骤，并如实说明未完成自动化验证的范围。

## 6. Windows 兼容性

* 项目以 Windows 为第一目标平台（CI 与发布均在 windows-latest 上运行）；开发机可能是 Windows 或 macOS，注意路径分隔符、大小写敏感性与 Shell 语法差异。
* 默认使用 UTF-8，并保留项目现有行尾风格。
* 路径可能包含空格时正确引用参数，不假设文件系统区分大小写。
* 仅修改文件名大小写时，应使用中间文件名过渡。
* 注意 PowerShell、CMD、Git Bash 和 WSL 的路径及环境变量语法差异；未经确认不混用 Windows 与 WSL 路径。
* 跨平台脚本不依赖仅在 Bash 中有效的语法。

## 7. 依赖管理

* Go 依赖统一在 `nodepaper/core` 模块内管理，工具链版本由 `go.mod` 钉住；不引入第二个模块根。
* 修改依赖前识别项目当前包管理器和锁文件；存在锁文件时使用与其匹配的工具。
* 不混用 `npm`、`pnpm` 和 `yarn`，不因普通代码修改升级依赖或重写锁文件。
* 仅在依赖实际变化时修改锁文件，不顺便升级无关依赖。
* 不主动全局安装依赖，优先使用项目本地工具或 `npx` 等临时执行方式。
* 引入新依赖前，评估现有依赖或标准库能否满足需求，以及体积、维护状态、许可证、安全和平台兼容性。

## 8. 测试与验证

任何可能改变程序行为、构建结果、依赖关系、配置、数据格式或用户界面的改动都必须有相应验证。仅修改说明文字、拼写或不参与执行的注释时，可以不运行完整测试，但应检查修改范围。

详细规则见 [验证规范](docs/development/verification.md)。

## 9. Git 工作流

* 仅在独立、可验证且适合保留的语义单元完成后创建本地提交。
* 提交前必须能够准确区分本次改动与已有改动，并完成最小验证。
* 不提交无关文件、敏感文件、构建产物、临时文件或未验证内容。
* Git 提交遵循 Conventional Commits，使用简洁中文描述，不包含任何暗示由 AI 生成的字样。
* 推送远程仓库必须取得工程师明确同意。
* 涉及 GitHub 构建、推送、PR、发行时遵循 [GitHub 规范](docs/development/github.md)。

详细规则见 [Git 工作流](docs/development/git-workflow.md)。

## 10. 项目记录

* 项目已有 `TODO.md` 或 `MEMORY.md` 时，按本规范维护。
* 工程师明确要求、任务需要跨会话维护，或项目确有长期协作需求时才主动创建；不为一次性任务机械创建。
* `TODO.md` 保持简洁，完成项标记完成而不删除。
* `MEMORY.md` 只记录长期有效的架构决策、技术约束和流程，不记录临时任务、密钥、账号、私人路径、个人信息或敏感部署信息。
* 正式架构决策应进入 ADR 或设计文档，不用 `MEMORY.md` 代替。

## 11. 敏感文件

* 密钥、Token、密码、私钥、账号、内部地址和本地专属配置不得提交，应按项目现状纳入 `.gitignore`。
* 不擅自提交 `.env`、缓存目录、构建产物或大型无关文件。
* `CLAUDE.md`、`AGENTS.md` 是否纳入版本控制取决于是否包含私人环境或敏感策略。

## 12. 终端与文档

* 终端输出使用稳定、可复制、可日志化的语义提示，禁止闪烁刷新和滥用 `*`、`-`、`=` 作为分隔线。
* 仅在交互式终端支持颜色时启用颜色；重定向、CI、非 TTY 或设置 `NO_COLOR` 时关闭。
* 颜色不能作为唯一区分方式，应同时提供 `[成功]`、`[警告]`、`[错误]` 等文字标识。
* 绿色表示成功，黄色表示警告，红色表示错误，青色表示交互，蓝色表示高亮和链接，灰色表示次要信息。
* 报告类 Markdown 应结构清晰、图文并茂但不堆砌装饰；普通技术说明以清晰为先。
* LaTeX 使用 XeLaTeX 编译，中文内容注意字体、编码和跨平台兼容性。
* 技术文档、报告、Markdown 与 PDF 交付遵循 [写作规范](docs/development/writing.md)；需要 LaTeX 专项流程时使用项目 `writing-standard` Skill。

## 13. 子代理

* 仅在任务可并行、边界清晰且文件冲突较小时使用子代理，最多 6 个。
* 适合并行调研、多模块审查或可独立拆分的文档、测试和实现；小范围单文件修改或高耦合任务不拆分。
* 主代理负责统一整合、验证和汇报，不机械拼接结论。
* 子代理的分工、编排、边界与安全详见 [子代理使用规范](docs/development/subagents.md)。

## 14. 专题规范

* [前端开发](docs/development/frontend.md)
* [Rust 开发](docs/development/rust.md)
* [HarmonyOS 开发](docs/development/harmonyos.md)
* [前后端热加载](docs/development/hot-reload.md)
* [Python 开发](docs/development/python.md)
* [CMake](docs/development/cmake.md)
* [Git 工作流](docs/development/git-workflow.md)
* [GitHub 规范](docs/development/github.md)
* [验证](docs/development/verification.md)
* [写作](docs/development/writing.md)
* [代码注释](docs/development/code-comments.md)
* [并发与线程](docs/development/concurrency.md)
* [接口文档](docs/development/api-documentation.md)
* [子代理使用](docs/development/subagents.md)

命名优先遵循项目现有规范。JavaScript、TypeScript 使用 `camelCase` 与 `PascalCase`；Python 使用 `snake_case`、`PascalCase` 与 `UPPER_SNAKE_CASE`；C、C++ 无现成约定时，不在单次任务中自行建立新命名体系。

## 15. 最终汇报

完成任务后简洁汇报改动、实际运行的验证、提交信息和未解决风险。没有提交或没有运行测试时，明确说明原因，不得虚假声称已验证。
