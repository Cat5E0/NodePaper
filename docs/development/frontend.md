# 前端开发规范

## 基本原则

- 优先遵循项目已有配置和代码风格，不擅自替换既有规范。
- JavaScript 和 TypeScript 代码遵循 Airbnb JavaScript Style Guide，并使用项目已有的 ESLint、Prettier 配置。
- 保持组件边界清晰、状态流向可追踪；不为一次性需求或未来扩展提前设计复杂抽象。

## 技术选型

- 动画使用 Motion。
- 路由使用 React Router。
- UI 组件使用 Ant Design。
- 样式使用 Ant Design Design Token 和 CSS Modules，不使用 Tailwind CSS。
- 图标使用 Lucide React，不引入 Ant Design Icons。
- 自定义标题栏等桌面专属组件自行封装。
- 状态管理使用 Redux Toolkit。
- 前端自动化测试使用 Vitest、React Testing Library、`@testing-library/user-event`、`@testing-library/jest-dom` 和 jsdom，不使用 Playwright。

## 状态管理

- 组件局部状态优先使用 `useState` 或 `useReducer`。
- 仅将跨页面、跨组件的工作流状态纳入 Redux Toolkit。
- Rust 后端是业务数据的事实来源，前端避免维护与其冲突的状态副本。
- 暂不引入 RTK Query；出现远程 HTTP API 后再评估。
- 不使用传统手写 Redux、Redux Saga 或 Redux Observable。

## 目录与职责

前端代码按以下职责组织；仅在实际需要时创建对应目录：

```text
app/src/
├── components/
│   └── common/
├── features/
│   └── <feature>/
│       ├── components/
│       ├── hooks/
│       ├── services/
│       ├── store/
│       └── types.ts
├── layouts/
├── routes/
├── services/
├── store/
├── styles/
└── utils/
```

- `routes` 中的页面只负责页面组装，不堆积业务实现。
- UI、异步调用、数据转换和状态管理应按职责放入组件、Hooks、服务或 Store。
- 依赖方向保持为 feature 指向 `components/common` 或基础库层；共用层不得反向依赖 feature。
- 组件在跨两个以上功能稳定复用后，才提升到 `components/common`；不得仅因外观相似提前抽象。
- 不包装每个 Ant Design 组件；仅在需要统一行为、主题或业务语义时封装。

## 样式复用

- 全局样式仅包含重置、字体、主题和应用级规则。
- 组件样式使用同名 CSS Module，并与组件就近存放。
- 颜色、间距、圆角和阴影优先使用 Ant Design Design Token。
- 禁止用全局选择器影响其他模块。
- 重复的视觉结构优先提取共用组件，不堆积到全局样式文件。

## 文件规模

- React 组件文件目标不超过 200 行。
- 超过 250 行必须拆分，或说明不能拆分的原因。
- 超过 400 行原则上禁止；生成文件、静态数据和类型声明除外。
- 单个组件或函数目标不超过 100 行。
- 一个文件只保留一个主要组件；小型私有子组件可以例外。
- 同一文件混合 UI、异步调用、数据转换和状态管理时，必须按职责拆分。
- ESLint 的 `max-lines` 和 `max-lines-per-function` 设为 `warning`，用于提示拆分，不作为机械阻断条件。

## 命名

- 变量和函数使用 `camelCase`。
- 组件和类使用 `PascalCase`。
- 项目已有不同命名约定时，以项目现有约定为准。

## 调试信息总线

- 根据项目复杂度和调试需求决定是否启用，不作为所有前端项目的强制初始化步骤。
- 涉及跨组件事件、复杂状态流、桌面容器通信或难以直接定位的问题时，可设置轻量调试信息总线，将必要信息发送到浏览器调试端。
- 简单页面或现有日志机制已足够时，直接复用现有能力。
- 调试逻辑不得污染业务逻辑，发布前应移除临时输出或按项目约定关闭。

## 自动化测试

- Vitest 作为测试运行器，jsdom 提供浏览器 DOM 环境。
- React Testing Library 以用户可观察的行为查询和操作界面；`user-event` 模拟点击、输入和键盘交互，`jest-dom` 提供语义化 DOM 断言。
- 测试文件与被测模块就近存放，命名为 `*.test.ts` 或 `*.test.tsx`。
- 组件测试覆盖渲染、用户交互和可见状态变化；Hooks、工具函数和状态逻辑按公开行为进行单元测试。
- Tauri IPC 使用 `vi.mock` 隔离；真实前后端联调不属于前端单元测试范围。
- 不测试组件内部实现细节，不滥用快照；仅在稳定且审查价值明确时使用快照。
- 本地开发使用 `npm run test` 进入监听模式，CI 使用 `npm run test:run` 单次执行；对应脚本分别约定为 `vitest` 和 `vitest run`。

## 桌面窗口与交互

- 禁用浏览器默认右键菜单，按产品需求提供自定义交互。
- 隐藏视觉滚动条，但保留滚轮、触控板和键盘滚动能力。
- 自定义无边框标题栏必须支持窗口拖拽、双击最大化或还原，以及最小化、最大化或还原、关闭操作。
- 窗口控制按钮必须具有可访问名称并支持键盘操作，且不得位于窗口拖拽区域内。
- 完成交互改动后，验证窗口控制和滚动行为的关键路径。

## 浏览器验证

涉及页面行为时，按以下顺序验证：

1. 优先运行项目已有的测试、检查或验证脚本。
2. 没有可用自动化验证时，提供可复现的手动验证步骤，并明确未自动验证的范围。

验证应覆盖本次改动的关键路径，不为局部修改额外引入重型测试体系。
