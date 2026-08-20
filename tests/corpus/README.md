# NodePaper 发布回归语料

本目录保存可公开审查的真实长文档回归工程。它们只证明两个已知复杂案例在 NodePaper 上没有退化，不代表任意论文都能正确构建。

## 边界

- `real-world/A163` 与 `real-world/C063` 只保留 canonical 工程实际需要的白名单文件。
- `real-world/A163` 是 8 个显式有序 Markdown Source 组成的多文件工程，并直接 `\input` 自己的 4 张定制表格；`real-world/C063` 是单个 Markdown Source，并直接 `\input` 自己的 2 张表格。两个工程配置、资源与 Fragment 完全独立。
- 原始论文 PDF、MinerU JSON/中间文件、旧私有包、构建产物、日志、缓存、工具脚本、观察记录和结构变体不进入 Git 或公开语料包。这里的构建产物包括 `.nodepaper/`（中间 TeX、日志、锁）和 `dist/`（最终 PDF）；它们应只出现在临时副本中。
- 正文中的附录程序是排版文本；NodePaper 不执行它们。
- 原论文及衍生内容的著作权仍属于原权利人。维护者已决定将净化副本用于 NodePaper 的非商业兼容性研究与回归测试；本仓库的 MIT License 不覆盖这些语料内容。
- 发现来源、许可或个人信息问题时，应先从公开包移除对应文件，再讨论恢复。

## 构建与打包

```powershell
nodepaper validate tests/corpus/real-world/A163
nodepaper build tests/corpus/real-world/A163
nodepaper validate tests/corpus/real-world/C063
nodepaper build tests/corpus/real-world/C063

.\scripts\build-test-corpus.ps1 -Version 0.1.0-beta.1
```

打包脚本按各工程白名单分别生成 `nodepaper-<version>-test-corpus-A163.zip`、`nodepaper-<version>-test-corpus-C063.zip` 及各自的 `.sha256`；每个 ZIP 只含一个可独立构建的工程。程序 ZIP 和 Setup 的发布白名单不包含 `tests/corpus`。

人工 PDF 抽查至少覆盖首页、目录、公式密集页、宽表/图片页、参考文献和附录；结果记录在发布检查记录中，不把生成 PDF 提交到此目录。

## 隐藏验证

隐藏验证工程不在本仓库。公开文件 `hidden-validation-manifest.example.json` 只定义匿名记录格式；真实映射、内容和日志保存在治理仓库的私有材料区。beta/rc 至少记录匿名项目 ID、冻结哈希、环境、pass/fail 和缺陷编号，不记录文件名、正文或绝对路径。

**与隐藏包的预期差异（勿当作同步事故）**：隐藏包（A163 v2、C063 v7，冻结于 2026-08-16）是**旧写法快照**——单文件/内联 LaTeX 大表、旧式粗体表题、无 `width`/`ratios`；本目录是**新写法基准**——多文件、表格 Fragment 化、新表题契约。两者验证方向不同（旧形态兼容 vs 新契约不退化），互不替代。

**同步纪律**：每次修改本目录语料内容时，维护者须判定一次“隐藏包是否需要同步重新冻结”；需要时在治理仓库（文档仓库）更新私有 manifest 哈希并记录差异摘要与理由，顺序为：代码仓库提交 → 文档仓库引用 commit 记录 → 私有包重冻结记录。不得静默重打私有包：manifest 哈希与 ZIP 不一致时，该轮隐藏验证结果作废。详见治理仓库 M4-19 任务文档“隐藏包与公开语料的差异及同步纪律”一节。

未知工程暴露缺陷后，先把问题缩减为虚构 Fixture 并加入回归；只有获得明确授权时才增加真实公开语料，同时视需要轮换或补充隐藏集。
