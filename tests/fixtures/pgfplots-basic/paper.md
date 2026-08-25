---
title: NodePaper pgfplots 最小测试
problem: A
keywords:
  - pgfplots
  - Fragment
reference-section-title: 参考文献
---

# 摘要

本文验证使用 `pgfplots` 的受控 Fragment 能够进入最终 PDF [@nodepaper2026fixture]。

# 正文

下面的图形来自项目内已声明的 `.tex` Fragment，其中用的是 `pgfplots` 语法而不是基础 PGF：

\input{figures/axis.tex}

坐标轴、刻度标签和折线应能正常显示，中文轴标题不得缺字。

# 参考文献 {-}

::: {#refs}
:::
