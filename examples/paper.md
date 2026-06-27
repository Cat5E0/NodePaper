---
title_zh: 基于Lyapunov泛函方法的时滞神经网络稳定性分析
title_en: Stability Analysis of Time-Delay Neural Networks Based on Lyapunov Functional Method
author_zh: 陈汝恒
author_en: Chen Ruheng
college: 数学与信息学院、软件学院
major: 信息与计算科学
student_id: "202225810103"
supervisor: 史晨阳
supervisor_title: 讲师
date: 2026-06-09
abstract_zh: |
  本文给出一个用于验证模板转换链路的示例摘要。摘要可以包含行内数学公式 $x(t)$，也可以包含多段文字。

  第二段摘要用于验证 Pandoc 片段转换和中文排版是否正常。
keywords_zh:
  - 神经网络
  - 指数稳定性
  - Lyapunov-Krasovskii泛函
  - 时变时滞
  - 线性矩阵不等式
abstract_en: |
  This sample abstract verifies the Markdown to SCAU LaTeX conversion pipeline. It may contain inline math such as $x(t)$ and multiple paragraphs.
keywords_en:
  - Neural networks
  - Exponential stability
  - Lyapunov-Krasovskii functional
  - Time-varying delay
  - Linear matrix inequality
acknowledgements: |
  感谢老师和同学在论文写作与排版测试过程中提供的帮助。
references_tex: |
  \bibitem{fridman2014}
  Fridman, E. (2014). \textit{Introduction to Time-Delay Systems}. Birkhäuser.

  \bibitem{park2015}
  Park, P., Ko, J. W., and Jeong, C. (2015). Reciprocally convex approach to stability of systems with time-varying delays. \textit{Automatica}, 47(1), 235--238.
---

# 绪论 {#sec:intro}

本文示例用于验证 Markdown 到 SCAU LaTeX 模板的转换。章节交叉引用可写为 @sec:intro，图片引用可写为 @fig:logo，公式引用可写为 @eq:model。

![华南农业大学标识示例](image/SCAU-LOGO.jpg){#fig:logo width=45%}

时滞神经网络的一类离散化模型可写为：

$$
x(t+1)=Ax(t)+Bu(t).
$$ {#eq:model}

由 @eq:model 可知，系统状态的演化同时受到当前状态和输入项影响。

::: theorem
若存在正定矩阵 $P$ 使得 $A^TPA-P<0$，则无输入线性系统是渐近稳定的。
:::

## 研究背景与意义 {#sec:background}

如 @sec:background 所示，Markdown 标题中的 `{#sec:...}` 会被保留为 LaTeX 标签，并由 `pandoc-crossref` 处理引用。

# 预备知识 {#sec:prelim}

下面给出一个简单表格，验证表格标签与引用。

| 方法 | 特点 |
|:--|:--|
| Jensen 不等式 | 结构简单 |
| Wirtinger 不等式 | 保守性较低 |

: 常见积分不等式对比 {#tbl:ineq}

见 @tbl:ineq。

下面给出一段带语言标记的代码块，用于验证 PDF 中的语法高亮。

```python
def stable(delay: float) -> bool:
    return delay < 1.0
```

# 总结与展望 {#sec:conclusion}

本文档是项目自带的烟雾测试输入，用于确认封面、摘要、正文、交叉引用、参考文献和致谢可以进入 SCAU 模板。

