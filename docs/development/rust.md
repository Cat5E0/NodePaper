# Rust 与 Tauri 开发规范

## 基本原则

- 使用 Rust 惯例和 `rustfmt` 默认格式，优先编写清晰、可测试、职责单一的代码。
- Tauri 仅承担桌面边界与适配职责，领域逻辑不得依赖 Tauri。
- 不引入本规范未要求的新依赖；依赖应按实际需求最小化。

## 目录与依赖方向

Rust 代码位于 `app/src-tauri/`，按需创建以下目录：

```text
app/src-tauri/
├── src/
│   ├── commands/
│   ├── domain/
│   ├── services/
│   ├── state/
│   ├── dto/
│   ├── lib.rs
│   └── main.rs
└── tests/
```

- `main.rs` 与 `lib.rs` 保持精简，只负责启动、注册和模块组装。
- `commands` 负责 Tauri IPC 边界，可调用 `domain`、`services` 和 `state`。
- `domain` 保存领域模型与核心规则，不依赖 Tauri、窗口或 IPC 类型。
- `services` 封装文件、系统、网络等外部能力；`state` 管理共享运行时状态；`dto` 定义 IPC 数据结构。
- 仅在实际需要时创建目录，不为一次性逻辑提前抽象。

## 命名与代码组织

- 模块、文件、变量和函数使用 `snake_case`。
- 类型、Trait、枚举和枚举变体使用 `PascalCase`；常量使用 `SCREAMING_SNAKE_CASE`。
- Tauri command 保持精简，只负责输入校验、状态获取、调用领域或服务逻辑以及结果转换。
- IPC DTO 与领域类型分离，避免把内部实现细节直接暴露给前端。

## 错误处理

- 可失败操作返回 `Result`，调用方必须显式处理错误。
- 生产路径不得随意使用 `unwrap`、`expect`、`panic!`、`todo!` 或 `unimplemented!`。
- 内部错误应结构化并保留可定位的上下文；IPC 返回稳定、可序列化的错误结构。
- 面向前端的错误不得泄露密钥、绝对路径、内部堆栈或其他敏感信息。
- 对 lint 的 `allow` 必须定向到最小范围，并写明原因；禁止文件级或全局无说明放宽。

## 异步、并发与共享状态

- 仅短时、非阻塞任务使用同步 command；I/O 或长耗时任务使用异步处理。
- 阻塞 I/O 和 CPU 密集任务必须与异步执行器隔离，避免阻塞 Tauri 运行时。
- 不得跨 `await` 持有锁；锁范围应尽可能短。
- 长任务应根据业务需要提供超时或取消机制，并避免产生无法回收的后台任务。
- 共享状态通过 `tauri::State` 和必要的 `Arc` 管理，不使用无约束的全局可变状态。
- 事件适合状态变更通知；需要顺序、背压或可靠消费的流程优先使用 channel。

## 安全边界

- Tauri capabilities 与 permissions 遵循最小授权，只开放实际使用的命令和资源。
- 文件路径在使用前必须规范化并验证位于允许范围内，不能只依赖字符串前缀判断。
- 禁止拼接字符串执行 shell 命令；必须调用外部程序时使用参数化 API，并校验程序与参数。
- 密钥、令牌、用户隐私和敏感路径不得写入日志或返回给前端。
- 原则上禁止 `unsafe`；确有必要时限制在最小作用域，并使用 `SAFETY:` 注释说明不变量与依据。

## 依赖管理

- Cargo feature 按需启用，不启用无关默认 feature。
- 保留并提交 `Cargo.lock`，保证应用构建可复现。
- 禁止使用通配版本；新增依赖需说明用途，修改时不得顺带升级无关依赖。

## 测试

- 领域规则和纯逻辑应优先编写单元测试，覆盖正常路径、边界和错误路径。
- command 测试聚焦输入校验、权限边界和 DTO 转换，不重复测试领域实现。
- Rust 集成测试放在 `app/src-tauri/tests/`；项目根目录 `tests/` 用于跨栈和验收测试。
- 外部系统能力应通过清晰边界隔离，使领域测试不依赖真实窗口、网络或用户环境。

## 文件规模

- Rust 文件目标不超过 300 行。
- 超过 400 行必须按职责拆分，或说明不能拆分的原因。
- 超过 600 行原则上禁止；生成文件、静态表和协议定义除外。
- 单个函数目标不超过 80 行；出现多阶段处理或多类错误时应提取职责明确的函数。

## 提交前验证

按项目实际支持的能力执行：

```powershell
cargo fmt --check
cargo clippy --all-targets --all-features -- -D warnings
cargo test
```

- 仅当项目完整支持全 feature 组合时，才对 Clippy 或测试启用 `--all-features`；否则运行项目定义的受支持 feature 集并说明范围。
- 验证命令应在 `app/src-tauri/` 执行，所有警告和测试失败必须在提交前处理。

## 官方依据

- [Tauri：从前端调用 Rust](https://v2.tauri.app/develop/calling-rust/)
- [Tauri：Capabilities](https://v2.tauri.app/security/capabilities/)
- [Tauri：Permissions](https://v2.tauri.app/security/permissions/)
