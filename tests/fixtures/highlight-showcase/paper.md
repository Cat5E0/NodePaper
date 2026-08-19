---
title: NodePaper 代码高亮样例
problem: A
keywords:
  - 代码高亮
  - 多语言
  - 中文注释
reference-section-title: 参考文献
---

# 摘要

本公开虚构样例只用于比较 NodePaper 的代码高亮配色、中文注释、长行换行和代码框，不承担完整排版压力测试。

# 代码高亮样例

## Python

```python
# 中文注释：计算一组完全虚构的目标值
def objective(values: list[float]) -> float:
    weighted_total = sum((index + 1) * value for index, value in enumerate(values))
    return weighted_total / max(len(values), 1)

sample_values = [1.25, 2.50, 3.75, 5.00]
print(f"synthetic objective = {objective(sample_values):.3f}; long text remains safely breakable")
```

## Matlab

```matlab
% 中文注释：构造公开虚构矩阵
A = [1, 2, 3; 4, 5, 6; 7, 8, 9];
weights = [0.2; 0.3; 0.5];
result = A * weights;
disp(result);
```

## PowerShell

```powershell
# 中文注释：输出公开虚构构建信息
$record = [ordered]@{
    Profile = 'cumcm'
    Highlight = 'reviewed-built-in-style'
    Success = $true
}
$record | ConvertTo-Json -Depth 3
```

## 无语言代码块

```
plain text block
C:\Users\Student\Documents\NodePaper\synthetic-project\dist\paper.pdf
```

以上颜色只来自 Pandoc 内置 Skylighting，不执行示例代码。相关测试方法参考虚构研究 [@smith2023forecast]。

# 参考文献 {-}

::: {#refs}
:::
