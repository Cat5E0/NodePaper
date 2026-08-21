// NodePaper 桌面壳 —— 极薄 Tauri 壳。
// compile_latex：把编辑区 Markdown 写成临时最小工程（nodepaper.yaml + paper.md +
// references.bib），在宿主 shell 环境直接执行 pandoc 完成 md → LaTeX 转换。
//
// 参数集逐项对齐核心 scripts/build/Convert-CumcmProjectToLatex.ps1 的 citeproc
// 主链（单文件情形）：同 --from 扩展、--top-level-division=section、同模板与
// lua 过滤器、citeproc + 国标 CSL、--fail-if-warnings。核心自身经
// powershell.exe 调用该脚本，仅 Windows 可达；macOS 与 pandoc 直连等价链路由
// 本命令承担。pandoc-crossref 为可选外部过滤器，未安装时跳过（编译模式预览
// 不做交叉引用编号）。

use std::fs;
use std::path::{Path, PathBuf};
use std::process::Command;
use std::sync::atomic::{AtomicU64, Ordering};
use std::time::{SystemTime, UNIX_EPOCH};

use serde::Serialize;
use tauri::Manager;

/// .md / .markdown 后缀匹配（书架与阅读共用；不引入 regex 依赖）
fn is_markdown_name(name: &str) -> bool {
    let lower = name.to_ascii_lowercase();
    lower.ends_with(".md") || lower.ends_with(".markdown")
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct CompileOutcome {
    /// 生成的 LaTeX 源码；失败时为空串
    pub tex: String,
    /// 工具链日志（pandoc 版本、命令、stdout/stderr）
    pub log: String,
    /// 实际使用的转换器描述
    pub tool: String,
}

#[tauri::command]
fn compile_latex(app: tauri::AppHandle, md: String) -> Result<CompileOutcome, String> {
    let profile_dir =
        find_profile_dir(app.path().resource_dir().ok().as_deref()).ok_or_else(|| {
            "未找到 profiles/cumcm（可用环境变量 NODEPAPER_PROFILE_DIR 指定）".to_string()
        })?;
    let pandoc = find_pandoc().ok_or_else(|| {
        "未找到 pandoc，请安装 pandoc 3.x 或用 NODEPAPER_PANDOC 指定路径".to_string()
    })?;
    run_compile(
        &md,
        &profile_dir,
        &pandoc,
        find_core_binary(app.path().resource_dir().ok().as_deref()),
    )
    .map_err(|e| e)
}

/// 书架文件条目；字段名与前端 FileEntry 约定一致
#[derive(Debug, Serialize)]
pub struct FileEntry {
    /// 绝对路径，用于读取
    pub path: String,
    /// 相对根目录的路径（/ 分隔），用于书架展示与排序
    pub rel: String,
    /// 文件名
    pub name: String,
}

/// 递归列出目录下所有 .md / .markdown 文件（书架数据源）。
/// 跳过隐藏目录与 node_modules / target / .git 等生成物目录；
/// 目录 symlink 不跟随，防循环。返回按 rel 排序。
#[tauri::command]
fn list_markdown(dir: String) -> Result<Vec<FileEntry>, String> {
    let root = PathBuf::from(&dir);
    if !root.is_dir() {
        return Err(format!("目录不存在或不可读：{}", dir));
    }
    const SKIP_DIRS: [&str; 4] = ["node_modules", "target", ".git", ".venv"];
    let mut out = Vec::new();
    // 迭代式递归（栈），深度上限防御异常嵌套
    let mut stack: Vec<(PathBuf, String, usize)> = vec![(root.clone(), String::new(), 0)];
    while let Some((dir_path, rel_prefix, depth)) = stack.pop() {
        if depth > 16 {
            continue;
        }
        let entries = fs::read_dir(&dir_path)
            .map_err(|e| format!("读取目录失败 {}：{}", dir_path.display(), e))?;
        for entry in entries.flatten() {
            let name = entry.file_name().to_string_lossy().to_string();
            // file_type 不跟随 symlink：目录链接不入选，防循环
            let Ok(ft) = entry.file_type() else { continue };
            if ft.is_dir() {
                if name.starts_with('.') || SKIP_DIRS.contains(&name.as_str()) {
                    continue;
                }
                let rel = if rel_prefix.is_empty() {
                    name.clone()
                } else {
                    format!("{}/{}", rel_prefix, name)
                };
                stack.push((entry.path(), rel, depth + 1));
            } else if is_markdown_name(&name) {
                let rel = if rel_prefix.is_empty() {
                    name.clone()
                } else {
                    format!("{}/{}", rel_prefix, name)
                };
                out.push(FileEntry {
                    path: entry.path().to_string_lossy().to_string(),
                    rel,
                    name,
                });
            }
        }
    }
    out.sort_by(|a, b| a.rel.cmp(&b.rel));
    Ok(out)
}

/// 读取 Markdown 文本（阅读区数据源）。
/// 只收 .md / .markdown；大小上限 10MB，防误读巨型文件卡死渲染。
#[tauri::command]
fn read_markdown(path: String) -> Result<String, String> {
    let p = Path::new(&path);
    let name = p
        .file_name()
        .map(|n| n.to_string_lossy().to_string())
        .unwrap_or_default();
    if !is_markdown_name(&name) {
        return Err(format!("仅支持 .md / .markdown 文件：{}", path));
    }
    let meta = fs::metadata(p).map_err(|e| format!("读取文件失败 {}：{}", path, e))?;
    if !meta.is_file() {
        return Err(format!("不是普通文件：{}", path));
    }
    const MAX_BYTES: u64 = 10 * 1024 * 1024;
    if meta.len() > MAX_BYTES {
        return Err(format!(
            "文件超过 10MB 上限（{} MB）：{}",
            meta.len() / 1024 / 1024,
            path
        ));
    }
    fs::read_to_string(p).map_err(|e| format!("读取失败 {}：{}", path, e))
}

/// 由内到外定位 profiles/cumcm：环境变量 → bundle 资源目录（打包态）→
/// 可执行文件/工作目录祖先（tauri dev 态，仓库根在其上两级）。
/// 返回路径一律 canonicalize：相对路径会随 pandoc 子进程 cwd（临时工程）
/// 改变含义，--template / --lua-filter 将解析失败。
fn find_profile_dir(resource_root: Option<&Path>) -> Option<PathBuf> {
    let absolutize = |p: PathBuf| fs::canonicalize(&p).ok().or(Some(p));
    if let Ok(env_dir) = std::env::var("NODEPAPER_PROFILE_DIR") {
        let p = PathBuf::from(env_dir);
        if p.join("profile.json").exists() {
            return absolutize(p);
        }
    }
    // 打包态：bundle.resources 把 profiles/cumcm 映射到资源目录
    if let Some(root) = resource_root {
        let p = root.join("profiles/cumcm");
        if p.join("profile.json").exists() {
            return absolutize(p);
        }
    }
    // dev 态：仓库根在 cwd（nodepaper/desktop）或 exe 的祖先上
    let target = Path::new("profiles/cumcm/profile.json");
    let mut candidates: Vec<PathBuf> = Vec::new();
    if let Ok(exe) = std::env::current_exe() {
        let mut dir = exe.parent().map(|d| d.to_path_buf());
        while let Some(d) = dir {
            candidates.push(d.to_path_buf());
            dir = d.parent().map(|p| p.to_path_buf());
        }
    }
    if let Ok(cwd) = std::env::current_dir() {
        let mut dir = Some(cwd);
        while let Some(d) = dir {
            candidates.push(d.to_path_buf());
            dir = d.parent().map(|p| p.to_path_buf());
        }
    }
    candidates
        .into_iter()
        .map(|base| base.join(target))
        .find(|p| p.exists())
        .and_then(|p| p.parent().map(|d| d.to_path_buf()))
        .and_then(absolutize)
}

/// 定位内嵌的 nodepaper core 二进制（bundle.resources 的 binaries/）。
/// core 的 build/export 链依赖 powershell.exe 仅 Windows 可达；macOS 桌面
/// 编译模式用 pandoc 直连。core 在位时用于未来的 validate/doctor 集成，
/// 此处先探测并写入日志。
fn find_core_binary(resource_root: Option<&Path>) -> Option<PathBuf> {
    if let Ok(env_core) = std::env::var("NODEPAPER_CORE") {
        let p = PathBuf::from(env_core);
        if p.is_file() {
            return Some(p);
        }
    }
    let root = resource_root?;
    let name = if cfg!(windows) {
        "nodepaper-windows-x64.exe"
    } else {
        "nodepaper-macos-aarch64"
    };
    let p = root.join("binaries").join(name);
    p.is_file().then_some(p)
}

/// 定位 pandoc：环境变量 NODEPAPER_PANDOC → PATH。
fn find_pandoc() -> Option<PathBuf> {
    if let Ok(env_pandoc) = std::env::var("NODEPAPER_PANDOC") {
        let p = PathBuf::from(env_pandoc);
        if p.is_file() {
            return Some(p);
        }
    }
    let name = if cfg!(windows) {
        "pandoc.exe"
    } else {
        "pandoc"
    };
    // which 语义：PATH 逐段探测，避免依赖系统 which
    let path_var = std::env::var_os("PATH")?;
    std::env::split_paths(&path_var)
        .map(|dir| dir.join(name))
        .find(|p| p.is_file())
}

/// 转换链主体：写临时工程 → 组装对齐参数 → 执行 pandoc → 回读 .tex。
fn run_compile(
    md: &str,
    profile: &Path,
    pandoc: &Path,
    core: Option<PathBuf>,
) -> Result<CompileOutcome, String> {
    // 目录名含 pid + 原子序号，避免同毫秒并发（如并行测试）碰撞
    static SEQ: AtomicU64 = AtomicU64::new(0);
    let stamp = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_millis())
        .unwrap_or(0);
    let uniq = format!(
        "nodepaper-compile-{}-{}-{}",
        stamp,
        std::process::id(),
        SEQ.fetch_add(1, Ordering::Relaxed)
    );
    let project = std::env::temp_dir().join(uniq);
    let out_dir = project.join("out");
    fs::create_dir_all(&out_dir).map_err(|e| format!("创建临时目录失败: {}", e))?;

    // 最小工程镜像 nodepaper init 的形状：配置 + 正文 + 参考文献（编译模式正文
    // 未必含引用，空 bib 供 citeproc 链保持与核心一致）
    fs::write(
        project.join("nodepaper.yaml"),
        "version: 1\nprofile: cumcm\nsource: paper.md\noutput:\n  file: out/paper.tex\n",
    )
    .map_err(|e| format!("写入 nodepaper.yaml 失败: {}", e))?;
    fs::write(project.join("paper.md"), md).map_err(|e| format!("写入 paper.md 失败: {}", e))?;
    fs::write(project.join("references.bib"), "")
        .map_err(|e| format!("写入 references.bib 失败: {}", e))?;

    // profile.json 只在存在时读取版本钉版信息，用于日志提示（mac 不做硬校验，
    // 与核心在 Windows 上的强校验不同——桌面编译模式面向预览）
    let mut log = String::new();
    let pandoc_version = run_capture(pandoc, &["--version"]);
    log.push_str(&format!(
        "Pandoc: {}\n",
        pandoc_version.lines().next().unwrap_or("未知版本")
    ));
    let crossref = find_on_path(if cfg!(windows) {
        "pandoc-crossref.exe"
    } else {
        "pandoc-crossref"
    });
    match &crossref {
        Some(p) => {
            let v = run_capture(p, &["--version"]);
            log.push_str(&format!(
                "pandoc-crossref: {}\n",
                v.lines().next().unwrap_or("未知版本")
            ));
        }
        None => log.push_str("pandoc-crossref: 未安装，跳过交叉引用编号（不影响正文转换）\n"),
    }
    match &core {
        Some(p) => {
            let v = run_capture(p, &["--version"]);
            log.push_str(&format!(
                "nodepaper-core: {} ({})\n",
                v.lines().next().unwrap_or("未知版本").trim(),
                p.display()
            ));
        }
        None => log.push_str(
            "nodepaper-core: 未内嵌（build/export 链仅 Windows 可达，编译模式已用 pandoc 直连）\n",
        ),
    }

    // 资源搜索路径：临时工程 + profile（分隔符平台差异：Windows ; / unix :）
    let sep = if cfg!(windows) { ";" } else { ":" };
    let mut args: Vec<String> = vec![
        "paper.md".into(),
        "--from".into(),
        "markdown+yaml_metadata_block+tex_math_dollars+raw_tex+link_attributes+implicit_figures+pipe_tables+fenced_divs".into(),
        "--to".into(),
        "latex".into(),
        "--standalone".into(),
        "--top-level-division=section".into(),
        "--template".into(),
        profile.join("template.tex").to_string_lossy().into(),
        "--metadata-file".into(),
        profile.join("crossref.yaml").to_string_lossy().into(),
        "--lua-filter".into(),
        profile.join("filters/extract-abstract.lua").to_string_lossy().into(),
        "--lua-filter".into(),
        profile.join("filters/layout.lua").to_string_lossy().into(),
    ];
    if let Some(cr) = &crossref {
        args.push("--filter".into());
        args.push(cr.to_string_lossy().into());
    }
    args.extend([
        "--citeproc".to_string(),
        "--bibliography".to_string(),
        "references.bib".to_string(),
        "--csl".to_string(),
        profile
            .join("csl/china-national-standard-gb-t-7714-2015-numeric.csl")
            .to_string_lossy()
            .into(),
        "--syntax-highlighting=tango".to_string(),
        "--metadata".to_string(),
        "link-citations=true".to_string(),
        "--fail-if-warnings".to_string(),
        "--resource-path".to_string(),
        format!(
            "{}{}{}",
            project.to_string_lossy(),
            sep,
            profile.to_string_lossy()
        ),
        "--output".to_string(),
        "out/paper.tex".to_string(),
    ]);

    log.push_str(&format!(
        "Pandoc command: {} {}\n",
        pandoc.display(),
        args.join(" ")
    ));

    let output = Command::new(pandoc)
        .args(&args)
        .current_dir(&project)
        .output()
        .map_err(|e| format!("执行 pandoc 失败: {}", e))?;
    let stdout = String::from_utf8_lossy(&output.stdout);
    let stderr = String::from_utf8_lossy(&output.stderr);
    log.push_str(&stdout);
    if !stderr.trim().is_empty() {
        log.push_str(&stderr);
    }

    let tex_path = out_dir.join("paper.tex");
    if !output.status.success() || !tex_path.exists() {
        let _ = fs::remove_dir_all(&project);
        return Err(format!(
            "转换失败（exit {}）：\n{}",
            output.status.code().unwrap_or(-1),
            stderr.trim()
        ));
    }
    let tex = fs::read_to_string(&tex_path).map_err(|e| format!("回读 paper.tex 失败: {}", e))?;
    let _ = fs::remove_dir_all(&project);

    Ok(CompileOutcome {
        tex,
        log,
        tool: format!("pandoc（对齐核心 citeproc 主链，profile: cumcm）"),
    })
}

fn find_on_path(name: &str) -> Option<PathBuf> {
    let path_var = std::env::var_os("PATH")?;
    std::env::split_paths(&path_var)
        .map(|dir| dir.join(name))
        .find(|p| p.is_file())
}

fn run_capture(bin: &Path, args: &[&str]) -> String {
    Command::new(bin)
        .args(args)
        .output()
        .map(|o| String::from_utf8_lossy(&o.stdout).to_string())
        .unwrap_or_default()
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_opener::init())
        .invoke_handler(tauri::generate_handler![
            compile_latex,
            list_markdown,
            read_markdown
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::{Mutex, MutexGuard};

    /// 环境变量锁：find_profile_dir 依赖进程级 env，并行测试会互相污染。
    /// 涉及 NODEPAPER_* env 的测试段必须持有此锁。
    static ENV_LOCK: Mutex<()> = Mutex::new(());

    fn env_lock() -> MutexGuard<'static, ()> {
        ENV_LOCK.lock().unwrap_or_else(|e| e.into_inner())
    }

    /// 真实链路测试：需要 pandoc。未安装时跳过（CI 无 pandoc 环境不红）。
    /// NODEPAPER_PANDOC / NODEPAPER_PROFILE_DIR 可注入测试工具路径。
    #[test]
    fn compile_sample_md_to_latex() {
        let _guard = env_lock();
        let profile = match find_profile_dir(None) {
            Some(p) => p,
            None => {
                eprintln!("skip: 未找到 profiles/cumcm");
                return;
            }
        };
        let pandoc = match find_pandoc() {
            Some(p) => p,
            None => {
                eprintln!("skip: 未找到 pandoc");
                return;
            }
        };
        let md = "# 摘要之前的标题\n\n::: abstract\n这是一段摘要，用于验证 extract-abstract 过滤器。\n:::\n\n## 第一节点\n\n正文一段，含 **加粗** 与 `code`。\n\n$$E = mc^2$$\n";
        let out = run_compile(md, &profile, &pandoc, None).expect("转换应成功");
        assert!(out.tex.contains("\\documentclass"), "应生成完整 LaTeX 文档");
        assert!(out.tex.contains("第一节点"), "正文应进入 .tex");
        assert!(out.log.contains("Pandoc"), "日志应含 pandoc 版本");
        assert!(
            out.log.contains("nodepaper-core"),
            "日志应报告 core 探测结果: {}",
            out.log
        );
    }

    /// 资源目录解析：模拟打包态 layout（resource_root 下有 profiles/cumcm）
    #[test]
    fn profile_resolves_from_resource_root() {
        let _guard = env_lock();
        std::env::remove_var("NODEPAPER_PROFILE_DIR");
        let tmp = std::env::temp_dir().join("np-resource-root-test");
        let dir = tmp.join("profiles/cumcm");
        fs::create_dir_all(&dir).expect("建目录");
        fs::write(dir.join("profile.json"), "{}").expect("写标记");
        let found = find_profile_dir(Some(&tmp));
        assert!(found.is_some(), "应从资源目录命中");
        let found = found.unwrap();
        assert!(found.is_absolute(), "返回路径必须是绝对路径: {:?}", found);
        assert_eq!(found.canonicalize().ok(), fs::canonicalize(&dir).ok());
        let _ = fs::remove_dir_all(&tmp);
    }

    /// 文件命令：嵌套列举（跳过隐藏/生成物目录）+ 扩展名校验 + 内容读回
    #[test]
    fn file_commands_walk_and_read() {
        let tmp = std::env::temp_dir().join("np-file-cmd-test");
        let _ = fs::remove_dir_all(&tmp);
        fs::create_dir_all(tmp.join("chapters")).unwrap();
        fs::create_dir_all(tmp.join(".git")).unwrap();
        fs::create_dir_all(tmp.join("node_modules")).unwrap();
        fs::write(tmp.join("paper.md"), "# 根文档\n").unwrap();
        fs::write(tmp.join("chapters").join("c1.markdown"), "## 第一章\n").unwrap();
        fs::write(tmp.join(".git").join("hidden.md"), "不应出现\n").unwrap();
        fs::write(tmp.join("node_modules").join("dep.md"), "不应出现\n").unwrap();
        fs::write(tmp.join("cover.txt"), "非 md\n").unwrap();

        let items = list_markdown(tmp.to_string_lossy().to_string()).expect("列举应成功");
        let rels: Vec<&str> = items.iter().map(|i| i.rel.as_str()).collect();
        assert_eq!(
            rels,
            vec!["chapters/c1.markdown", "paper.md"],
            "应递归且跳过生成物目录"
        );
        assert!(items[0].path.contains("chapters"), "path 应为绝对可读路径");

        let text = read_markdown(items[1].path.clone()).expect("读回应成功");
        assert_eq!(text, "# 根文档\n");

        let err = read_markdown(tmp.join("cover.txt").to_string_lossy().to_string())
            .expect_err("非 md 应拒绝");
        assert!(err.contains("仅支持"), "错误应可读：{}", err);

        let err = list_markdown(tmp.join("不不存在").to_string_lossy().to_string())
            .expect_err("不存在目录应报错");
        assert!(err.contains("目录不存在"), "错误应可读：{}", err);

        let _ = fs::remove_dir_all(&tmp);
    }

    #[test]
    fn fail_path_reports_stderr() {
        let _guard = env_lock();
        let profile = match find_profile_dir(None) {
            Some(p) => p,
            None => {
                eprintln!("skip: 未找到 profiles/cumcm");
                return;
            }
        };
        let pandoc = match find_pandoc() {
            Some(p) => p,
            None => {
                eprintln!("skip: 未找到 pandoc");
                return;
            }
        };
        // citeproc 对未收录引用报 warning，--fail-if-warnings 将其升级为硬错误；
        // 这是用户真实会碰到的失败（引了 bib 里没有的文献），核心链同样失败
        let err = run_compile(
            "# t\n\n引用不存在条目 [@missing-ref]\n",
            &profile,
            &pandoc,
            None,
        )
        .expect_err("未收录引用应失败");
        assert!(err.contains("转换失败"), "错误信息应可读：{}", err);
    }
}
