# 更改日志

本项目尚未发布正式版本，以下为自首次提交以来的重大更改。版本号遵循 `X.Y.Z-dev.N+g<commit>` → `-beta.N` → `-rc.N` → `X.Y.Z` 生命周期；正式发布后按版本分节。

## [未发布]

### 新增

- **CUMCM 排版体系**：CUMCM 垂直切片与排版契约（基于官方 2026 格式事实），可导航 PDF 书签大纲，参考文献与附录独立起页。
- **排版配置项**：`linespread` 全文档行距、`abstractLinespread` 摘要区行距、`mathFont`（cm/newtx）数学字体路线、表格居中与题注分隔样式。
- **导出通道**：`nodepaper export` 产出可编辑 LaTeX 工程（Overleaf 可直接编译），`--bib` 支持 inline / natbib(bibtex) / biblatex 三种参考文献模式，`--to` 以 `.zip` 结尾时自动导出压缩包；无引用论文不再输出任何参考文献机制。
- **安装与入门**：`nodepaper setup` 安装与引导流程；Inno Setup 安装器（钉版工具链、双击可装、降级需确认）与便携 ZIP 双通道（解压目录 PATH 注册、升级换目录、卸载只摘 PATH）。
- **工程化**：版本发布生命周期强制校验；`doctor` 按能力报告环境（缺 TeX 不再直接失败）；测试语料库（真实赛题 A163/C063 回归 corpus 与 fixtures）；Windows 图标与 logo。
- **Tauri 桌面端**：引入 Tauri + React 桌面壳（`nodepaper/desktop/`），Go 核心迁至 `nodepaper/core/`，形成 core / desktop / shared 三层布局。

### 变更

- **破坏性**：XeLaTeX 改为直接驱动并逐轮解析日志收敛判定，**移除 latexmk 依赖**；overfull box 由阻断改为警告。
- 默认交付路径从"本地编译 PDF"转向 `export` 可编辑工程；README 默认语言改为简体中文。
- 项目起点为 SCAU 论文模板工具链，后演进为通用 profile 体系（`profiles/`，当前仅 `cumcm`）。

### 修复

- CJK 字体族绑定（sans/mono 缺字、内联代码与正文丢字形）、摘要中文首行缩进、CJK 回退字体的 pdf 路径与 none 计数器。
- 含波浪号（`~`）路径被 TeX 截断、中文路径与含空格路径的构建失败。
- LaTeX 箭头与邮箱地址被误判为引用键、声明未插入的 LaTeX fragment 缺失告警、命令行失败原因丢失。
