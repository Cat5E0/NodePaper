# 表格压力 {#sec:layout-table}

NP-LAYOUT-TABLE-01。普通八列表格用于验证 Pandoc 长表格输出：

| 序号 | 场景 | 指标甲 | 指标乙 | 指标丙 | 指标丁 | 状态 | 说明 |
|---:|---|---:|---:|---:|---:|---|---|
| 1 | 合成场景一 | 12 | 18 | 24 | 30 | 正常 | 可公开虚构数据 |
| 2 | 合成场景二 | 15 | 21 | 27 | 33 | 正常 | 长单元格需要在列宽内换行而不能越过正文边界 |
| 3 | 合成场景三 | 17 | 23 | 29 | 35 | 正常 | 布局表格标记二 |

: 普通宽表格压力样例 {#tbl:layout-wide}

受控复杂长表格 Fragment：

\input{tables/complex-result.tex}

见\cref{tab:fragment-long}，其首行标记为 NP-LAYOUT-LONGTABLE-FIRST，末行标记为 NP-LAYOUT-LONGTABLE-LAST。
