# CMake 规范

## 基本原则

- 使用现代 CMake，并以 target 为中心组织构建配置。
- 优先使用 `target_compile_features`、`target_compile_options`、`target_include_directories` 和 `target_link_libraries`。
- 为依赖关系明确声明 `PRIVATE`、`PUBLIC` 或 `INTERFACE`，避免目录级全局配置。
- 不使用全局 `include_directories`、`link_libraries` 或 `add_definitions` 影响无关目标。

## 编译器警告

项目自有目标必须启用严格警告：

```cmake
if(MSVC)
  target_compile_options(project_target PRIVATE /W4)
else()
  target_compile_options(project_target PRIVATE -Wall -Wextra -Wpedantic)
endif()
```

- MSVC 使用 `/W4`。
- GCC 和 Clang 使用 `-Wall -Wextra -Wpedantic`。
- 不在公共接口上传播警告选项。

## 警告视为错误

- 通过项目选项或 CI 控制 warnings-as-errors，不在所有构建环境中无条件启用。
- 本地开发默认可关闭，CI 应启用以阻止新增警告。
- MSVC 使用 `/WX`，GCC 和 Clang 使用 `-Werror`。

示例：

```cmake
option(PROJECT_WARNINGS_AS_ERRORS "Treat project warnings as errors" OFF)

if(PROJECT_WARNINGS_AS_ERRORS)
  if(MSVC)
    target_compile_options(project_target PRIVATE /WX)
  else()
    target_compile_options(project_target PRIVATE -Werror)
  endif()
endif()
```

## 第三方依赖

- 第三方目标不得继承项目警告选项或 warnings-as-errors。
- 警告配置只应用于项目自有目标。
- 外部头文件按工具链能力使用 `SYSTEM` 标记，避免第三方警告污染项目构建。
- 引入第三方子目录时，优先关闭其非必要测试、示例和开发者警告。
