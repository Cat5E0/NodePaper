# NodePaper 设计 Token 规范

面向 NodePaper 桌面端（`nodepaper/desktop/`）的设计 token 权威源。读者为实现桌面端界面的前端开发者；落地方式为 CSS 自定义属性，可直接拷贝本文的落地代码块到 `src/styles/`。

当前状态：规范已定稿，尚未替换前端代码中素笺遗留的扁平 token（见第 9 节迁移说明）。

## 1. 设计立场

前端源码迁移自素笺（su-jian），其 token 为单层扁平结构：`:root` 中原始色值、语义命名、布局尺寸混排，主题切换需逐主题重写整套变量。本规范在不推翻素笺视觉气质的前提下，为 NodePaper 重建三层 token 体系，并做三处身份转向：

| 维度 | 素笺（原） | NodePaper（本规范） | 理由 |
|------|-----------|---------------------|------|
| 强调色 | 青瓷绿 `#4F6354` | 墨水蓝 `#2E5C88` | 呼应学术出版与蓝黑墨水传统，匹配论文构建工具身份；logo 为黑白墨线，不预设彩色品牌色 |
| UI 字体 | 全 UI 衬线 | UI 部件 sans，阅读区衬线 | 构建工具的菜单、按钮、表单需高辨识；正文阅读保留纸感。工坊用 sans，书房用 serif |
| 状态色 | 无 | 新增构建反馈色（成功/警告/错误/运行中） | 桌面端核心场景是构建 PDF，必须有状态语义色 |

视觉基调保持素笺验证过的"纸墨"方向：暖灰纸面、墨色文字、低饱和线条、克制阴影。

## 2. 命名与层级

所有 token 统一前缀 `--np-`，与素笺遗留变量（如 `--paper`、`--celadon`）并存不冲突，便于渐进迁移。三层结构：

```text
Primitive（原始值）   --np-gray-4、--np-blue-500、--np-space-4
       ↓
Semantic（语义别名）  --np-surface-page、--np-ink-body、--np-accent
       ↓
Component（组件映射） --np-btn-bg、--np-menu-radius
```

规则：

- Primitive 只定义阶，不出现用途词；每主题下取值稳定或随主题整体翻转。
- Semantic 只做用途命名，引用 primitive，主题差异在此层体现（引用重指，而非重写色值）。
- Component 只在组件需要偏离语义默认时定义，保持最小数量。
- 组件样式代码禁止裸写 hex、rgba、rem 间距；例外见第 9 节。

## 3. Primitive · 色彩

### 3.1 中性灰阶（纸墨轴，暖灰）

单一明度轴，亮主题从低端取表面、暗主题从高端取表面。全栈 hex 如下：

| Token | 值 | 角色（亮主题） |
|-------|-----|---------------|
| `--np-gray-0` | `#FCFBF9` | 表面高光 |
| `--np-gray-1` | `#F7F5F1` | 抬升面 |
| `--np-gray-2` | `#EFECE6` | 主纸面 |
| `--np-gray-3` | `#E3DFD7` | 弱分隔线 |
| `--np-gray-4` | `#D3CEC3` | 分隔线 |
| `--np-gray-5` | `#B4ADA0` | 装饰性弱文本 |
| `--np-gray-6` | `#8C8578` | 次要文本（仅装饰/禁用，见 9.2） |
| `--np-gray-7` | `#5C574D` | 弱化正文文本 |
| `--np-gray-8` | `#37332C` | 正文文本 |
| `--np-gray-9` | `#211E19` | 强文本 |

### 3.2 墨水蓝（强调色）

| Token | 值 | 用途 |
|-------|-----|------|
| `--np-blue-50` | `#EEF3F8` | 强调淡染底 |
| `--np-blue-100` | `#D9E4EF` | 选中底、hover 深一档 |
| `--np-blue-200` | `#B3C9DE` | 边框级强调 |
| `--np-blue-300` | `#9FC0DE` | 暗主题强调文字 |
| `--np-blue-400` | `#4C7BA6` | 信息色（亮主题） |
| `--np-blue-500` | `#2E5C88` | 主强调 |
| `--np-blue-600` | `#24496D` | 强调 hover |
| `--np-blue-700` | `#1B3754` | 强调最暗 |

### 3.3 反馈色（构建状态）

| Token | 值 | 语义 | 对比度（on paper） |
|-------|-----|------|-------------------|
| `--np-ok-600` | `#356B47` | 构建成功 | 5.32 |
| `--np-warn-600` | `#85611F` | LaTeX 警告 | 4.78 |
| `--np-err-600` | `#B03A3A` | 构建失败 | 5.07 |
| `--np-ok-300` | `#7FAF8E` | 暗主题成功 | — |
| `--np-warn-300` | `#D0A85C` | 暗主题警告 | — |
| `--np-err-300` | `#D07676` | 暗主题错误 | — |

运行中（running）复用强调色 `--np-accent`，不单设色相。对比度按 WCAG 2.1 相对亮度公式实测（正文 `#EFECE6` 底），完整数据见附录 A。

## 4. Primitive · 字体、字号、间距

### 4.1 字体族

| Token | 栈 | 用途 |
|-------|-----|------|
| `--np-font-sans` | `-apple-system, BlinkMacSystemFont, "PingFang SC", "Segoe UI", sans-serif` | UI 部件（菜单、按钮、表单、状态） |
| `--np-font-serif-zh` | `"Noto Serif SC", "Songti SC", "SimSun", serif` | 中文正文 |
| `--np-font-serif-en` | `"Newsreader", "Iowan Old Style", Georgia, serif` | 英文正文 |
| `--np-font-kai` | `"LXGW WenKai", "Kaiti SC", "STKaiti", var(--np-font-serif-zh)` | 章节标题、品牌章印（签名） |
| `--np-font-display` | `"Cormorant Garamond", "Newsreader", serif` | 英文眉题、斜体装饰 |
| `--np-font-mono` | `"IBM Plex Mono", "SF Mono", ui-monospace, monospace` | 代码、快捷键 |

### 4.2 字号阶梯

基于 18px 根正文的阅读器尺度：

| Token | 值 | 典型用途 |
|-------|-----|---------|
| `--np-text-2xs` | `0.6875rem` (11px) | 快捷键提示 |
| `--np-text-xs` | `0.75rem` (12px) | 辅助标签 |
| `--np-text-sm` | `0.875rem` (14px) | 菜单项、工具按钮 |
| `--np-text-base` | `1rem` (16px) | 窄屏正文 |
| `--np-text-md` | `1.0625rem` (17px) | 阅读区默认正文 |
| `--np-text-lg` | `1.25rem` (20px) | h3、导语强调 |
| `--np-text-xl` | `1.5rem` (24px) | h2 |
| `--np-text-2xl` | `2rem` (32px) | 面板标题 |
| `--np-text-3xl` | `2.5rem` (40px) | h1（配 clamp） |

### 4.3 间距与圆角

间距以 4px 为基：`--np-space-1` 至 `--np-space-9` = 4/8/12/16/20/24/32/48/64 px。

圆角收敛为四档：

| Token | 值 | 用途 |
|-------|-----|------|
| `--np-radius-xs` | `0.25rem` | kbd 芯片、行内代码 |
| `--np-radius-sm` | `0.4rem` | 工具按钮、菜单项 |
| `--np-radius-md` | `0.6rem` | 下拉菜单、右键菜单 |
| `--np-radius-lg` | `0.8rem` | 对话框卡片 |

## 5. Primitive · 阴影、动效、Z 序

### 5.1 阴影

| Token | 值 | 用途 |
|-------|-----|------|
| `--np-shadow-flat` | `inset 0 1px 0 rgba(255,255,255,.6)` | 代码块内嵌高光 |
| `--np-shadow-card` | `0 10px 24px -16px rgba(33,30,26,.25)` | 卡片 |
| `--np-shadow-pop` | `0 18px 40px -20px rgba(33,30,26,.35)` | 菜单浮层 |
| `--np-shadow-modal` | `0 30px 60px -30px rgba(33,30,26,.4)` | 模态对话框 |

### 5.2 动效

| Token | 值 | 用途 |
|-------|-----|------|
| `--np-dur-fast` | `120ms` | 菜单、hover 反馈 |
| `--np-dur-base` | `200ms` | 颜色、透明度过渡 |
| `--np-dur-slow` | `350ms` | 顶栏收起、面板折叠 |
| `--np-ease-out` | `cubic-bezier(.2,.7,.2,1)` | 签名缓动（沿自素笺面板动效） |

`prefers-reduced-motion: reduce` 时全局关闭动画，规则在落地 CSS 中保留。

### 5.3 Z 序

沿素笺现有层级收编为阶梯，落地时不改现有组件的相对顺序：

| Token | 值 | 组件 |
|-------|-----|------|
| `--np-z-base` | `0` | 正文 |
| `--np-z-brush` | `5` | 阅读进度墨笔 |
| `--np-z-masthead` | `20` | 顶栏 |
| `--np-z-menu` | `35` | 主题菜单 |
| `--np-z-overlay` | `40` | 粘贴浮层 |
| `--np-z-dropdown` | `45` | 应用菜单下拉 |
| `--np-z-veil` | `50` | 拖放遮罩 |
| `--np-z-context` | `60` | 右键菜单 |

## 6. Semantic 层

### 6.1 表面、墨色、线条

| Token | 亮主题引用 | 说明 |
|-------|-----------|------|
| `--np-surface-page` | `var(--np-gray-2)` | 主纸面（含纸纹背景） |
| `--np-surface-raise` | `var(--np-gray-1)` | 侧栏等抬升面 |
| `--np-surface-sheen` | `var(--np-gray-0)` | 菜单、模态高光面 |
| `--np-ink-strong` | `var(--np-gray-9)` | 标题、强文本 |
| `--np-ink-body` | `var(--np-gray-8)` | 正文 |
| `--np-ink-soft` | `var(--np-gray-7)` | 弱化正文（对比度 6.09，达标） |
| `--np-ink-mute` | `var(--np-gray-6)` | 仅装饰性文本与禁用态（对比度 3.10，见 9.2） |
| `--np-line` | `var(--np-gray-4)` | 分隔线、边框 |
| `--np-line-soft` | `var(--np-gray-3)` | 弱分隔线 |

### 6.2 强调与交互态

| Token | 亮主题引用 | 说明 |
|-------|-----------|------|
| `--np-accent` | `var(--np-blue-500)` | 主强调（链接、选中、焦点） |
| `--np-accent-hover` | `var(--np-blue-600)` | 强调 hover |
| `--np-accent-soft` | `var(--np-blue-200)` | 强调边框级 |
| `--np-accent-wash` | `var(--np-blue-50)` | 强调淡染底 |
| `--np-on-accent` | `var(--np-gray-0)` | 强调底上的文字 |
| `--np-selection` | `rgba(46,92,136,.18)` | 文本选区 |
| `--np-state-hover` | `rgba(0,0,0,.04)` | 中性 hover 叠加 |
| `--np-state-active` | `rgba(0,0,0,.065)` | 中性激活叠加 |
| `--np-state-hover-invert` | `rgba(255,255,255,.07)` | 暗主题 hover 叠加 |

素笺样式中 `rgba(0,0,0,.04)` 等散落 8 处以上，全部收编为以上三个交互态 token。

### 6.3 反馈语义

| Token | 亮主题引用 | 说明 |
|-------|-----------|------|
| `--np-ok` | `var(--np-ok-600)` | 构建成功 |
| `--np-warn` | `var(--np-warn-600)` | 警告 |
| `--np-err` | `var(--np-err-600)` | 错误 |
| `--np-info` | `var(--np-blue-400)` | 运行中/提示 |
| `--np-focus-ring` | `var(--np-accent)` | 焦点环（2px） |

### 6.4 布局

| Token | 值 | 说明 |
|-------|-----|------|
| `--np-measure` | `62ch` | 正文行宽 |
| `--np-gutter` | `clamp(1.25rem, 4vw, 3.5rem)` | 页缘留白 |
| `--np-shelf-w` | `264px` | 书架列宽 |
| `--np-toc-w` | `248px` | 目录列宽 |

## 7. 明暗主题机制

主题切换通过重指 semantic 的 primitive 引用实现，不重写色值。主题属性沿用现有 `data-theme`，token 以 `[data-np-theme="dark"]` 为暗色锚点（落地时可与 `data-theme="ink-night"` 合并声明）：

```css
[data-np-theme="dark"] {
  /* 表面翻转到灰阶高端（暗端锚定色，属 9.1 例外） */
  --np-surface-page: #191613;
  --np-surface-raise: #241F1A;
  --np-surface-sheen: #2B2520;
  /* 墨色翻转到低端 */
  --np-ink-strong: #E9E2D6;
  --np-ink-body: #CFC5B4;
  --np-ink-soft: #B7AF9F;
  --np-ink-mute: #8C8578;
  /* 线条取暗阶 */
  --np-line: #3B322B;
  --np-line-soft: #2E2722;
  /* 强调翻到蓝阶亮端 */
  --np-accent: var(--np-blue-300);
  --np-accent-hover: var(--np-blue-200);
  --np-accent-soft: rgba(159,192,222,.4);
  --np-accent-wash: rgba(159,192,222,.12);
  --np-selection: rgba(159,192,222,.22);
  /* 反馈色换暗阶版本 */
  --np-ok: var(--np-ok-300);
  --np-warn: var(--np-warn-300);
  --np-err: var(--np-err-300);
  --np-info: var(--np-blue-300);
  /* 交互态换反向叠加 */
  --np-state-hover: var(--np-state-hover-invert);
  --np-state-active: rgba(255,255,255,.1);
}
```

暗色表面（`#191613` 等）与 `#2B2520` 一类抬升色为暖炭调，与亮端暖纸同族；此处 6 个表面/线条 hex 属于灰阶轴的暗端延伸，允许直接书写。暗主题关键对比度实测见附录 A。

## 8. Component 层（示例）

仅在偏离语义默认时定义。当前给出现有组件的映射基准，后续新组件按需追加：

```css
:root {
  /* 按钮 */
  --np-btn-bg: var(--np-surface-raise);
  --np-btn-fg: var(--np-ink-soft);
  --np-btn-radius: var(--np-radius-sm);
  --np-btn-primary-bg: var(--np-accent);
  --np-btn-primary-fg: var(--np-on-accent);
  /* 菜单 */
  --np-menu-bg: var(--np-surface-sheen);
  --np-menu-radius: var(--np-radius-md);
  --np-menu-shadow: var(--np-shadow-pop);
  /* 代码 */
  --np-code-chip-bg: rgba(33,30,26,.06);      /* 亮主题行内代码底 */
  --np-code-block-bg: var(--np-surface-sheen);
  --np-code-block-shadow: var(--np-shadow-flat);
  /* h1 标题（沿素笺 clamp 尺度） */
  --np-h1-size: clamp(2.2rem, 5.5vw, 3.4rem);
}
```

组件级微调值（如素笺阅读排版中的 `0.55rem` 表格内边距）不强行 token 化，允许在组件样式内保留，但新增代码应优先取间距阶梯。

## 9. 使用规则与迁移

### 9.1 禁止与例外

- 组件样式禁止裸写 hex 与 rgba；颜色一律走 token。
- 例外一：纸纹 SVG data-URI 与背景渐变中的锚定色，允许保留字面值。
- 例外二：`hljs` 语法高亮色板为外部配色方案（GitHub Light/Dark），作为整体模块保留，不逐色 token 化。
- 例外三：第 7、8 节声明的主题锚定 hex 与组件微调值。

### 9.2 无障碍约束

- `--np-ink-body`、`--np-ink-soft`、`--np-accent` 用于文本或与文本同级的信息传达时，对所在表面对比度不低于 4.5:1（WCAG AA），实测数据见附录 A。
- `--np-ink-mute`（3.10）不得用于承载必要信息的文本；仅限装饰、占位与禁用态。
- 焦点可见性：交互控件 focus 一律 `outline: 2px solid var(--np-focus-ring)`，颜色不是唯一区分方式。

### 9.3 迁移策略

素笺遗留变量与 `--np-` 体系并存，按组件渐进替换：改到哪个组件，该组件的 `--paper`、`--ink` 等引用即切换为对应 `--np-` 语义 token；全部替换完成后删除遗留变量。主题机制从"每主题重写整套变量"迁至"semantic 重指"，`apple`、`graphite`、`parchment`、`sakura` 四个亮色主题按第 6 节结构重述，暗色 `ink-night` 并入 `dark` 锚点。

## 附录 A · 对比度实测

WCAG 2.1 相对亮度公式计算，亮主题背景 `#EFECE6`（paper），暗主题背景 `#191613`：

| 配对 | 对比度 | 判定 |
|------|--------|------|
| `#211E19` on paper（ink-strong） | 14.09 | AAA |
| `#37332C` on paper（ink-body） | 10.65 | AAA |
| `#5C574D` on paper（ink-soft） | 6.09 | AA |
| `#8C8578` on paper（ink-mute） | 3.10 | 仅装饰 |
| `#2E5C88` on paper（accent） | 5.93 | AA |
| `#24496D` on paper（accent-hover） | 7.92 | AAA |
| `#FCFBF9` on accent（按钮反白） | 6.76 | AA |
| `#356B47` / `#85611F` / `#B03A3A` on paper（反馈） | 5.32 / 4.78 / 5.07 | AA |
| `#E9E2D6` on dark（ink-strong） | 13.85 | AAA |
| `#B7AF9F` on dark（ink-soft） | 8.28 | AAA |
| `#8C8578` on dark（ink-mute） | 4.93 | AA |
| `#9FC0DE` on dark（accent） | 9.50 | AAA |
| `#7FAF8E` / `#D0A85C` / `#D07676` on dark（反馈） | 7.23 / 8.10 / 5.61 | AAA / AAA / AA |
