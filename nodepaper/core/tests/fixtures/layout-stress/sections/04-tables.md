# 表格压力 {#sec:layout-table}

NP-LAYOUT-TABLE-01。普通八列表格用于验证 Pandoc 长表格输出：

| 序号 | 场景 | 指标甲 | 指标乙 | 指标丙 | 指标丁 | 状态 | 说明 |
|---:|---|---:|---:|---:|---:|---|---|
| 1 | 合成场景一 | 12 | 18 | 24 | 30 | 正常 | 可公开虚构数据 |
| 2 | 合成场景二 | 15 | 21 | 27 | 33 | 正常 | 长单元格需要在列宽内换行而不能越过正文边界 |
| 3 | 合成场景三 | 17 | 23 | 29 | 35 | 正常 | 布局表格标记二 |

: 普通宽表格压力样例 {#tbl:layout-wide}

NP-LAYOUT-TABLE-AUTO。自然宽度表格不应因 Markdown 源码长度变化而被拉伸到页宽：

| 符号 | 含义 |
| :---: | :----------------: |
| $x$ | 决策变量 |
| $c$ | 单位成本 |

: 自然宽度表格样例 {#tbl:layout-auto width=auto}

NP-LAYOUT-TABLE-PERCENT。百分比宽度与显式比例共同控制表格和列宽：

| 短列 | 需要更多空间的说明列 |
| :---: | :----------------------------: |
| A | 该列用于验证受控换行和总宽度 |

: 百分比宽度表格样例 {#tbl:layout-percent width=80% ratios="1:4"}

受控复杂长表格 Fragment：

\input{tables/complex-result.tex}

见\cref{tab:fragment-long}，其首行标记为 NP-LAYOUT-LONGTABLE-FIRST，末行标记为 NP-LAYOUT-LONGTABLE-LAST。
