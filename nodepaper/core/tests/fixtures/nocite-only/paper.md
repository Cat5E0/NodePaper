---
title: NodePaper nocite-only 测试论文
problem: A
keywords:
  - nocite
  - 文献测试
reference-section-title: 参考文献
nocite: '@wang2024bikesharing @smith2023forecast'
---

# 摘要

本项目正文不含任何行内文献引用，所有参考文献条目仅通过 Front Matter 的 `nocite` 字段列出。它复现 A163 风格的合法写法：作者只想把参考文献表挂出来，正文不点引用。`nodepaper build`（citeproc 路由）能正确把 `nocite` 条目排进参考文献表；导出为 natbib 或 biblatex 工程时，若不补发 `\nocite`，bibtex/biber 会因找不到 `\citation` 命令而失败。本项目用于固定该导出缺陷的回归。

# 正文 {#sec:body}

这里有一段公式：

$$
z = x + y
$$ {#eq:add}

如式 @eq:add 所示，本文只验证公式编号，正文不出现任何方括号加引用键形式的行内文献引用。

# 参考文献 {-}

::: {#refs}
:::
