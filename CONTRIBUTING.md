# 贡献指南

感谢关注 NodePaper。项目当前处于 v0.1 测试版阶段，面向 CUMCM 2026 电子版论文场景，正在向桌面端（Tauri + React）扩展。贡献前请先阅读 [README.md](README.md) 与 [AGENTS.md](AGENTS.md)（仓库结构、规范优先级与工作区约定的事实来源）。

## 开发环境

| 依赖 | 说明 |
| --- | --- |
| Go | 工具链版本由 `nodepaper/core/go.mod` 钉住（当前 1.26.5），无需手动匹配版本 |
| PowerShell | 所有构建与测试脚本为 `.ps1`，CI 在 PowerShell 7（pwsh）下运行；Windows 建议使用 PowerShell 7+，macOS/Linux 需安装 `pwsh` |
| Pandoc / pandoc-crossref | 由 `scripts/dev/Bootstrap-Tools.ps1` 按钉版清单安装到用户目录，不污染系统 |
| XeLaTeX | 仅构建 PDF 需要；MiKTeX 或 TeX Live 均可，缺宏包时首次构建自动下载 |
| Node.js + pnpm | 仅桌面端（`nodepaper/desktop/`）需要；锁文件为 `pnpm-lock.yaml` |
| Rust | 仅桌面端 Tauri 壳（`nodepaper/desktop/src-tauri/`）需要 |

项目以 Windows 为第一目标平台（CI 与发布产物均在 windows-latest）。macOS 可以开发核心与前端，但涉及路径分隔符、注册表与安装器的改动必须在 Windows 上验证。

## 构建与测试

所有 `go` 命令在 `nodepaper/core/` 目录下执行：

```powershell
cd nodepaper/core
go build ./cmd/nodepaper      # 构建 CLI
go test -count=1 ./...        # 单元 + 集成
```

或直接使用仓库脚本（自动定位仓库根与核心根）：

```powershell
./scripts/test-unit.ps1                    # gofmt + go test + go vet
./scripts/test-integration.ps1             # 集成测试包
./scripts/test-xelatex-convergence.ps1     # 假 XeLaTeX 收敛测试，无需 TeX
./scripts/test-all.ps1 -Race -IncludeE2E   # 全量，含竞态与真实 E2E
```

真实 E2E 需要完整 Pandoc + XeLaTeX 工具链；无 TeX 环境时先跑其余部分。

## 提交与 PR 约定

* 提交信息遵循 Conventional Commits，使用简洁中文描述（如 `fix(export): 修复无引用时仍输出参考文献机制`）。
* **禁止任何 AI 署名 trailer**（如 `Co-Authored-By: ...`）。维护者是唯一 contributor。
* 仅在独立、可验证的语义单元完成后提交；不提交无关文件、构建产物或未验证内容。
* PR 目标分支为 `main`；CI 五项检查（gofmt / vet / test / race / govulncheck）必须全部通过。
* 涉及排版、导出或安装器行为的改动，请在 PR 中说明实际运行过的验证（脚本名或手动步骤），不接受"应该没问题"。
* 不放宽 CI 规则让检查变绿；不自动接受 Golden 文件差异。

## 版本与发布

版本号遵循 `scripts/version-lifecycle.ps1` 定义的生命周期：`X.Y.Z-dev.N+g<commit>` → `X.Y.Z-beta.N` → `X.Y.Z-rc.N` → `X.Y.Z`。发布由维护者通过 `release-build` workflow 执行；贡献者不要自行打 tag、创建 Release 或修改版本号。

## 更多规范

开发、验证、Git 工作流与写作等专题规范见 [docs/development/](docs/development/)。重大更改记录见 [CHANGELOG.md](CHANGELOG.md)。
