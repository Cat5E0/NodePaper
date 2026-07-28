# 模型建立 {#sec:model}

根据第 @sec:problem 节与第 @sec:data 节，定义目标函数：

$$
\min Z =
\sum_{i=1}^{n}\sum_{j=1}^{n} c_{ij}y_{ij}
+ \lambda \sum_{i=1}^{n} s_i
$$ {#eq:objective}

守恒约束为：

$$
x_i + \sum_{j=1}^{n}y_{ji}
- \sum_{j=1}^{n}y_{ij}
- d_i + s_i \ge 0
$$ {#eq:balance}

以上公式将在后续 Source 中被引用。
