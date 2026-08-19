# NodePaper 发布回归语料

本目录保存可公开审查的真实长文档回归工程。它们只证明两个已知复杂案例在 NodePaper 上没有退化，不代表任意论文都能正确构建。

## 边界

- `real-world/A163` 与 `real-world/C063` 只保留 canonical 工程实际需要的白名单文件。
- `real-world/A163` 是 8 个显式有序 Markdown Source 组成的多文件工程；`real-world/C063` 是单个 Markdown Source，并从正文直接 `\input` 两张已声明的 LaTeX 表格。
- 原始论文 PDF、MinerU JSON/中间文件、旧私有包、构建产物、日志、缓存、工具脚本、观察记录和结构变体不进入 Git 或公开语料包。
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

打包脚本按 `corpus-manifest.json` 的文件白名单生成独立的 `nodepaper-<version>-test-corpus.zip` 与 `.sha256`。程序 ZIP 和 Setup 的发布白名单不包含 `tests/corpus`。

人工 PDF 抽查至少覆盖首页、目录、公式密集页、宽表/图片页、参考文献和附录；结果记录在发布检查记录中，不把生成 PDF 提交到此目录。

## 隐藏验证

隐藏验证工程不在本仓库。公开文件 `hidden-validation-manifest.example.json` 只定义匿名记录格式；真实映射、内容和日志保存在治理仓库的私有材料区。beta/rc 至少记录匿名项目 ID、冻结哈希、环境、pass/fail 和缺陷编号，不记录文件名、正文或绝对路径。

未知工程暴露缺陷后，先把问题缩减为虚构 Fixture 并加入回归；只有获得明确授权时才增加真实公开语料，同时视需要轮换或补充隐藏集。
