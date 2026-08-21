# 前后端热加载规范

## 统一开发入口

- 桌面开发统一使用项目定义的 `npm run tauri dev`，由 Tauri CLI 同时管理前端开发服务器和 Rust 应用进程。
- `tauri.conf.json` 的 `build.beforeDevCommand` 启动 Vite，`build.devUrl` 必须与 Vite 的开发地址和端口一致。
- 不额外编写重复的文件监听或进程守护脚本；确有性能问题时，优先使用 `.taurignore` 排除生成文件、构建产物等无关路径。
- Vite 设置 `clearScreen: false`，避免前端刷新覆盖 Rust 编译错误。

## 前端 HMR

- React、TypeScript 和 CSS 修改由 Vite HMR 局部更新，避免无必要的整窗刷新。
- 模块初始化必须支持完整页面刷新，不得依赖 HMR 保留状态才能正确运行。
- HMR 后出现状态、订阅或样式异常时，应先执行完整刷新；不得用重复注册监听器等方式掩盖生命周期错误。
- 前端只能将可丢失的界面临时状态留在内存中；需要跨应用重启保留的数据必须持久化到明确的数据源。

## Rust 自动重建与重启

- `tauri dev` 监视 `app/src-tauri/` 及其工作区依赖 crate；Rust 变更后自动重新编译并重启应用。
- Rust 侧属于“自动重建并重启”，不是进程内热替换；重启后 `tauri::State`、线程、任务、锁和内存缓存均视为失效。
- 启动流程必须幂等；持久状态写入明确的文件或数据库，临时状态不得假定能够跨重启保留。
- 前端必须处理 IPC 短暂中断和窗口重载，启动后重新建立必要的查询、订阅与状态同步。
- 修改构建脚本、权限、capabilities、插件或依赖后，如果自动监听未生效，应完整停止并重新执行开发命令。

## 验证

开发环境至少验证以下路径：

1. 修改一个 React 组件或 CSS Module，确认界面由 Vite HMR 更新且无重复订阅。
2. 修改一个 Rust command，确认终端完成重新编译、应用自动重启且新逻辑生效。
3. 在 Rust 重启后执行一次前后端交互，确认前端能恢复 IPC 调用和必要状态。
4. 完整停止并重新启动开发命令，确认结果不依赖上一次 HMR 或进程内状态。

## 官方依据

- [Tauri：开发](https://v2.tauri.app/zh-cn/develop/)
- [Tauri：Vite 前端配置](https://v2.tauri.app/start/frontend/vite/)
