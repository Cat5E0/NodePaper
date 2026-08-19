# 用户指南索引

根目录 README 负责最短的安装、配置和构建路径；本目录逐步补充需要解释原理、边界和排错步骤的用户指南。本页是已收集主题的索引，不把尚未展开的条目误写成新的产品承诺。

## 已发布指南

- [TikZ / PGF Fragment](tikz-pgf.md)：外部导出、声明、插入、字体/路径限制和排错。

## 待整合与展开

### 项目配置与首页排版

- `abstractLinespread`：只调整摘要和关键词区域的行距；当关键词落到第二页时，采用小步调整并重新人工检查首页。
- `linespread`、图片宽度、分页和长内容的关系；区分可由作者配置解决的问题与应报告的模板/构建缺陷。
- 单文件和多文件 `source` / `sources` 的选择、唯一 Front Matter、Source 的显式顺序和跨文件引用。

### Markdown 表格与受控 LaTeX Fragment

- 普通表格直接在表题属性中控制总宽度：`width=auto` 按内容自然收缩，`width=80%` 使用正文宽度的 80%，`width=full` 使用完整正文宽度。调整这些宽度不需要 LaTeX Fragment。
- 百分比宽度和 `full` 可加 `ratios="1:4"` 精确指定各列的相对比例；数值数量必须与列数一致。Markdown 分隔行的横杠数有时也会影响 Pandoc 列宽，但短表会丢失这项信息，因此只能视为启发式提示，不能替代 `ratios` 的稳定契约。
- 只有需要缩小局部字号、竖线分组、合并单元格（`\multirow`）或其他 Markdown 无法表达的结构时，才使用受控 LaTeX Fragment：在 `latexFragments` 白名单声明，再在 Markdown 目标位置 `\input{...}`。Fragment 仍是 Project 源码，应提交 Git。
- Fragment 的安全边界、`\label` / `\ref` 的引用规则，以及何时应把真实案例缩减为虚构 Fixture 回归。
- 表题不要在 `: 标题 {#tbl:id}` 或 `\caption{}` 内再手工加粗。当前 CUMCM Profile 的中文正文默认宋体、粗体映射到黑体；手工加粗会使表号仍为宋体而标题切成黑体，形成明显的字体和字重断裂。需要统一改变表题样式时应由 Profile 集中控制。

### 构建目录、产物与提交边界

- `dist/` 是最终 PDF 等可重新生成的输出；`.nodepaper/` 是中间 TeX、日志、锁等 NodePaper 工作状态。它们会在同一个 Project 中同时出现，但都不属于源码、不提交 Git，也不进入发布程序包或公开测试语料包。
- 应提交的内容是 Markdown、`nodepaper.yaml`、图片、参考文献和已声明的 `.tex` / `.pgf` Fragment。测试应在临时副本中构建，避免在 Fixture 或 Corpus 源目录留下生成物。
- 锁测试的 `build.lock` 是唯一的测试输入例外，不代表普通 `.nodepaper/` 目录可以提交。
- `nodepaper export --to paper.zip --zip` 生成可直接上传 Overleaf 的根目录 ZIP；不加 `--zip` 时保持文件夹输出。两种形式都只包含可编辑 LaTeX 交付物，不包含 PDF 或中间文件。

### 图形、资源与展示

- TikZ、低层 PGF 与 `pgfplots` 的支持边界；Matplotlib 导出时的字体一致性与大图性能。
- README Showcase 的高清导出方法：从 PDF 直接按展示页渲染 PNG，保留原始 PDF，不使用聊天截图或二次压缩图；页面选择由维护者决定。
- 图片路径、相对 Project 根目录、尺寸声明与构建前后资源检查。

### 诊断与发布前检查

- `validate` 与 `build` 的分工、常见 Fragment 诊断、最小复现的写法。
- 生成 PDF 的人工检查重点：首页、宽表、公式密集页、参考文献和附录。

## 编写顺序

下一步优先扩写“Markdown 表格与受控 LaTeX Fragment”和“项目配置与首页排版”；两者已经由 C063、A163 与 `layout-stress` 的实际回归样例暴露出明确的用户需求。每项指南发布前都必须以当前 README、配置契约和真实构建结果复核。
