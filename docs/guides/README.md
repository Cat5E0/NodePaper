# 用户指南索引

根目录 README 负责最短的安装、配置和构建路径；本目录补充需要解释原理、边界和排错步骤的用户指南。每一页都以当前 CLI、配置契约和可复现构建为准；未提供的设置不会写成已支持能力。

## 已发布指南

- [项目编写、排版与导出](project-authoring.md)：单/多文件组织、`nodepaper.yaml`、摘要首页、Markdown 表格、受控 LaTeX Fragment、构建产物边界、导出与 Overleaf、排错。
- [TikZ / PGF Fragment](tikz-pgf.md)：外部导出、声明、插入、字体/路径限制和排错。

## 阅读路径

1. 从根目录 README 完成安装、初始化和第一次构建。
2. 需要拆分论文、调摘要或排普通表格时，阅读[项目编写、排版与导出](project-authoring.md)。
3. 需要 TikZ、低层 PGF 或外部工具导出的图时，阅读 [TikZ / PGF Fragment](tikz-pgf.md)。

所有指南均以当前实现为界。诸如 `pgfplots` 兼容性、摘要独立段距和 Markdown 表题/表注字体的独立配置目前都不是已提供的功能；如未来实现，会另行在发布说明和对应指南中标明。
