# 模型求解与结果

本文件跨文件引用式 @eq:objective、式 @eq:balance、表 @tbl:symbols 和图 @fig:demand-trend。

```python
def dispatch_cost(values):
    return sum(values)
```

| 方案 | 总成本（元） | 缺车率 |
|---|---:|---:|
| 调度前 | 12500 | 18.2% |
| 调度后 | 8930 | 6.4% |

: 多文件版结果对比 {#tbl:result}

表 @tbl:result 用于验证表格编号。模型思想参考虚构教材 [@li2022optimization] 和测试报告 [@city2025report]。[^multi]

[^multi]: 该脚注位于多文件项目的中间 Source 中。
