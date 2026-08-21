// 环境诊断与外部工具自取（pandoc / TinyTeX）。
//
// 安装位置用应用数据目录（app_data_dir）而非应用安装目录：
// macOS 应用包内写入会破坏 Gatekeeper 签名，Windows Program Files 需提权；
// app_data_dir 用户可写、跨平台一致。查找顺序 env → app_data/bin → PATH，
// 对编译链等价于「随应用走」。
// 解压统一走系统 bsdtar（macOS / Windows 10+ 内置，支持 zip / tar.xz / tar.gz），
// 不引入压缩 crate；下载走 reqwest 流式，按 1% 步进节流发 np:tool-install 事件。
// Windows TinyTeX 官方只发安装器（Inno Setup），用 /VERYSILENT /DIR 静默装入数据目录。

use std::fs;
use std::io::Write;
use std::path::{Path, PathBuf};
use std::process::Command;

use serde::Serialize;
use tauri::{AppHandle, Emitter, Manager};

use crate::run_capture;

/// 前端进度事件通道
pub const EVENT_INSTALL: &str = "np:tool-install";

// ---------------------------------------------------------------- 下载清单

#[derive(Debug, Clone, Copy, PartialEq)]
enum Layout {
    /// 压缩包内含单个可执行（zip），定位后拷入 bin/
    ZipBinary(&'static str),
    /// 压缩包展开为目录树（tar.xz），整体就位为 tinytex/
    TarTree(&'static str),
    /// Windows 安装器（Inno Setup），静默参数指定安装目录
    InstallerExe,
}

struct ToolSpec {
    key: &'static str,
    label: &'static str,
    url: &'static str,
    /// 发布资产字节数（GitHub Release API），用于进度与完整性校验
    size: u64,
    layout: Layout,
}

/// 可自取工具清单（按平台）。版本与资产名对齐 2026-08 的上游 Release。
fn manifest() -> Vec<ToolSpec> {
    if cfg!(target_os = "windows") {
        vec![
            ToolSpec {
                key: "pandoc",
                label: "Pandoc",
                url: "https://github.com/jgm/pandoc/releases/download/3.9/pandoc-3.9-windows-x86_64.zip",
                size: 40_860_638,
                layout: Layout::ZipBinary("pandoc.exe"),
            },
            ToolSpec {
                key: "tinytex",
                label: "TinyTeX（XeLaTeX）",
                url: "https://github.com/rstudio/tinytex-releases/releases/download/v2026.08/TinyTeX-1-windows-v2026.08.exe",
                size: 73_670_669,
                layout: Layout::InstallerExe,
            },
        ]
    } else {
        vec![
            ToolSpec {
                key: "pandoc",
                label: "Pandoc",
                url: "https://github.com/jgm/pandoc/releases/download/3.9/pandoc-3.9-arm64-macOS.zip",
                size: 42_595_715,
                layout: Layout::ZipBinary("pandoc"),
            },
            ToolSpec {
                key: "tinytex",
                label: "TinyTeX（XeLaTeX）",
                url: "https://github.com/rstudio/tinytex-releases/releases/download/v2026.08/TinyTeX-1-darwin-v2026.08.tar.xz",
                size: 67_051_368,
                layout: Layout::TarTree("TinyTeX"),
            },
        ]
    }
}

// ---------------------------------------------------------------- 工具定位

fn exe_name(base: &str) -> String {
    if cfg!(windows) {
        format!("{}.exe", base)
    } else {
        base.to_string()
    }
}

/// which 语义：PATH 逐段探测，避免依赖系统 which
pub fn find_on_path(name: &str) -> Option<PathBuf> {
    let path_var = std::env::var_os("PATH")?;
    std::env::split_paths(&path_var)
        .map(|dir| dir.join(name))
        .find(|p| p.is_file())
}

pub fn find_pandoc(data: Option<&Path>) -> Option<PathBuf> {
    if let Ok(env_pandoc) = std::env::var("NODEPAPER_PANDOC") {
        let p = PathBuf::from(env_pandoc);
        if p.is_file() {
            return Some(p);
        }
    }
    if let Some(d) = data {
        let p = d.join("bin").join(exe_name("pandoc"));
        if p.is_file() {
            return Some(p);
        }
    }
    find_on_path(&exe_name("pandoc"))
}

/// pandoc-crossref 可选：app_data/bin 手动放置或 PATH 均可被发现
pub fn find_crossref(data: Option<&Path>) -> Option<PathBuf> {
    let name = exe_name("pandoc-crossref");
    if let Some(d) = data {
        let p = d.join("bin").join(&name);
        if p.is_file() {
            return Some(p);
        }
    }
    find_on_path(&name)
}

/// TinyTeX 发行版的 bin 子目录平台三元组（darwin 为 universal）
const TEX_TRIPLES: [&str; 3] = ["universal-darwin", "x86_64-darwin", "windows"];

pub fn find_xelatex(data: Option<&Path>) -> Option<PathBuf> {
    let name = exe_name("xelatex");
    if let Some(d) = data {
        for t in TEX_TRIPLES {
            let p = d.join("tinytex").join("bin").join(t).join(&name);
            if p.is_file() {
                return Some(p);
            }
        }
    }
    find_on_path(&name)
}

// ---------------------------------------------------------------- 诊断

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ToolStatus {
    pub key: String,
    pub label: String,
    pub found: bool,
    pub path: Option<String>,
    pub version: Option<String>,
    pub downloadable: bool,
    pub size_mb: Option<f64>,
    pub hint: Option<String>,
}

fn probe_version(bin: &Path, args: &[&str]) -> Option<String> {
    run_capture(bin, args)
        .lines()
        .next()
        .map(|s| s.trim().to_string())
        .filter(|s| !s.is_empty())
}

fn found(key: &str, label: &str, p: &Path) -> ToolStatus {
    ToolStatus {
        key: key.to_string(),
        label: label.to_string(),
        found: true,
        path: Some(p.to_string_lossy().to_string()),
        version: probe_version(p, &["--version"]),
        downloadable: false,
        size_mb: None,
        hint: None,
    }
}

fn missing(spec: &ToolSpec) -> ToolStatus {
    ToolStatus {
        key: spec.key.to_string(),
        label: spec.label.to_string(),
        found: false,
        path: None,
        version: None,
        downloadable: true,
        size_mb: Some(spec.size as f64 / 1024.0 / 1024.0),
        hint: None,
    }
}

/// 诊断编译链四件套：pandoc（必需）/ pandoc-crossref（可选，仅手动）/
/// TinyTeX·XeLaTeX（PDF 编译用）/ nodepaper-core（内嵌）。
#[tauri::command]
pub fn diagnose_tools(app: AppHandle) -> Result<Vec<ToolStatus>, String> {
    let data = app.path().app_data_dir().ok();
    let d = data.as_deref();
    let man = manifest();
    let mut out = Vec::new();

    let pandoc_spec = man.iter().find(|s| s.key == "pandoc").unwrap();
    out.push(match find_pandoc(d) {
        Some(p) => found("pandoc", "Pandoc", &p),
        None => missing(pandoc_spec),
    });

    out.push(match find_crossref(d) {
        Some(p) => found("crossref", "pandoc-crossref", &p),
        None => ToolStatus {
            key: "crossref".into(),
            label: "pandoc-crossref".into(),
            found: false,
            downloadable: false,
            hint: Some(
                "可选（交叉引用编号）。发布包为 .tar.xz / .7z，请手动安装到应用数据 bin/ 或 PATH"
                    .into(),
            ),
            ..missing(pandoc_spec)
        },
    });

    let tex_spec = man.iter().find(|s| s.key == "tinytex").unwrap();
    out.push(match find_xelatex(d) {
        Some(p) => found("tinytex", "TinyTeX（XeLaTeX）", &p),
        None => missing(tex_spec),
    });

    let core = crate::find_core_binary(app.path().resource_dir().ok().as_deref());
    out.push(match core {
        Some(p) => found("core", "nodepaper-core", &p),
        None => ToolStatus {
            key: "core".into(),
            label: "nodepaper-core".into(),
            found: false,
            downloadable: false,
            hint: Some(
                "build/export 链依赖 powershell.exe 仅 Windows 可达；桌面编译已用 pandoc 直连"
                    .into(),
            ),
            ..missing(pandoc_spec)
        },
    });

    Ok(out)
}

// ---------------------------------------------------------------- 下载与就位

#[derive(Clone, Serialize)]
#[serde(rename_all = "camelCase")]
struct InstallProgress {
    key: String,
    received: u64,
    total: u64,
    pct: u32,
}

/// 下载并安装外部工具。确认弹窗由前端原生 ask 完成；此处为
/// 下载（流式 + 进度事件）→ 大小校验 → 解压就位。返回安装路径。
#[tauri::command]
pub async fn install_tool(app: AppHandle, key: String) -> Result<String, String> {
    let spec = manifest()
        .into_iter()
        .find(|s| s.key == key)
        .ok_or_else(|| format!("未知工具：{}", key))?;
    let data = app
        .path()
        .app_data_dir()
        .map_err(|e| format!("无法定位应用数据目录：{}", e))?;
    fs::create_dir_all(&data).map_err(|e| format!("创建数据目录失败：{}", e))?;
    let tmp = data.join("tmp");
    fs::create_dir_all(&tmp).map_err(|e| format!("创建临时目录失败：{}", e))?;
    let fname = spec.url.rsplit('/').next().unwrap_or("download.bin");
    let archive = tmp.join(fname);

    // 流式下载；GitHub Release 会 302 到 CDN，reqwest 默认跟随
    let client = reqwest::Client::new();
    let resp = client
        .get(spec.url)
        .send()
        .await
        .map_err(|e| format!("下载失败：{}", e))?;
    if !resp.status().is_success() {
        return Err(format!("下载失败：HTTP {}", resp.status()));
    }
    let total = resp.content_length().unwrap_or(spec.size);
    let mut file = fs::File::create(&archive).map_err(|e| format!("写临时文件失败：{}", e))?;
    let mut received: u64 = 0;
    let mut last_pct = u32::MAX;
    let mut stream = resp.bytes_stream();
    use futures_util::StreamExt;
    while let Some(chunk) = stream.next().await {
        let chunk = chunk.map_err(|e| format!("下载中断：{}", e))?;
        file.write_all(&chunk)
            .map_err(|e| format!("写盘失败：{}", e))?;
        received += chunk.len() as u64;
        let pct = if total > 0 {
            ((received * 100) / total) as u32
        } else {
            0
        };
        if pct != last_pct {
            last_pct = pct;
            let _ = app.emit(
                EVENT_INSTALL,
                InstallProgress {
                    key: key.clone(),
                    received,
                    total,
                    pct,
                },
            );
        }
    }
    drop(file);

    // 完整性：字节数对不上清单即失败，防止 CDN 半包/劫持页
    let got = fs::metadata(&archive).map(|m| m.len()).unwrap_or(0);
    if spec.size > 0 && got != spec.size {
        let _ = fs::remove_file(&archive);
        return Err(format!(
            "大小校验失败：期望 {} 字节，实得 {}，请重试",
            spec.size, got
        ));
    }

    let dest = place_files(&archive, spec.layout, &data)?;
    let _ = fs::remove_file(&archive);
    let _ = fs::remove_dir_all(tmp.join("staging"));
    Ok(dest.to_string_lossy().to_string())
}

/// 解压就位（同步纯函数，便于本地 fixture 测试，不触网络）
fn place_files(archive: &Path, layout: Layout, data: &Path) -> Result<PathBuf, String> {
    let staging = data.join("tmp").join("staging");
    let _ = fs::remove_dir_all(&staging);
    fs::create_dir_all(&staging).map_err(|e| format!("创建解压目录失败：{}", e))?;
    let out = Command::new("tar")
        .args([
            "-xf",
            archive.to_str().unwrap_or_default(),
            "-C",
            staging.to_str().unwrap_or_default(),
        ])
        .output()
        .map_err(|e| format!("启动解压失败（系统需内置 tar）：{}", e))?;
    if !out.status.success() {
        return Err(format!(
            "解压失败：{}",
            String::from_utf8_lossy(&out.stderr).trim()
        ));
    }
    match layout {
        Layout::ZipBinary(find) => {
            let bin_dir = data.join("bin");
            fs::create_dir_all(&bin_dir).map_err(|e| format!("创建 bin 目录失败：{}", e))?;
            let src = find_file(&staging, find, 0).ok_or("压缩包内未找到可执行文件")?;
            let dest = bin_dir.join(find);
            fs::copy(&src, &dest).map_err(|e| format!("就位失败：{}", e))?;
            #[cfg(unix)]
            {
                use std::os::unix::fs::PermissionsExt;
                let _ = fs::set_permissions(&dest, fs::Permissions::from_mode(0o755));
            }
            Ok(dest)
        }
        Layout::TarTree(top) => {
            let src = find_dir(&staging, top, 0).ok_or("压缩包内未找到工具目录")?;
            let dest = data.join(top.to_lowercase());
            // 覆盖旧安装（用户已在确认弹窗知情）；rename 失败（跨设备）回退复制
            let _ = fs::remove_dir_all(&dest);
            if fs::rename(&src, &dest).is_err() {
                copy_dir(&src, &dest)?;
            }
            Ok(dest)
        }
        Layout::InstallerExe => {
            // TinyTeX Windows 官方只发 Inno Setup 安装器：静默装入数据目录
            let dest = data.join("tinytex");
            let out = Command::new(archive)
                .args([
                    "/VERYSILENT",
                    "/NORESTART",
                    "/SUPPRESSMSGBOXES",
                    &format!("/DIR={}", dest.display()),
                ])
                .output()
                .map_err(|e| format!("启动安装器失败：{}", e))?;
            if !out.status.success() {
                return Err(format!(
                    "安装器执行失败：{}",
                    String::from_utf8_lossy(&out.stderr).trim()
                ));
            }
            Ok(dest)
        }
    }
}

fn find_file(dir: &Path, name: &str, depth: usize) -> Option<PathBuf> {
    if depth > 8 {
        return None;
    }
    for entry in fs::read_dir(dir).ok()?.flatten() {
        let p = entry.path();
        if p.is_file() && p.file_name()?.to_string_lossy() == name {
            return Some(p);
        }
        if p.is_dir() {
            if let Some(hit) = find_file(&p, name, depth + 1) {
                return Some(hit);
            }
        }
    }
    None
}

fn find_dir(dir: &Path, name: &str, depth: usize) -> Option<PathBuf> {
    if depth > 8 {
        return None;
    }
    for entry in fs::read_dir(dir).ok()?.flatten() {
        let p = entry.path();
        if p.is_dir() {
            if p.file_name()?.to_string_lossy() == name {
                return Some(p);
            }
            if let Some(hit) = find_dir(&p, name, depth + 1) {
                return Some(hit);
            }
        }
    }
    None
}

fn copy_dir(src: &Path, dest: &Path) -> Result<(), String> {
    fs::create_dir_all(dest).map_err(|e| e.to_string())?;
    for entry in fs::read_dir(src).map_err(|e| e.to_string())? {
        let entry = entry.map_err(|e| e.to_string())?;
        let p = entry.path();
        let to = dest.join(entry.file_name());
        if p.is_dir() {
            copy_dir(&p, &to)?;
        } else {
            fs::copy(&p, &to).map_err(|e| e.to_string())?;
            #[cfg(unix)]
            {
                use std::os::unix::fs::PermissionsExt;
                if let Ok(mode) = p.metadata().map(|m| m.permissions().mode()) {
                    let _ = fs::set_permissions(&to, fs::Permissions::from_mode(mode));
                }
            }
        }
    }
    Ok(())
}

// ---------------------------------------------------------------- 测试（不触网络）

#[cfg(test)]
mod tests {
    use super::*;

    fn env_lock() -> std::sync::MutexGuard<'static, ()> {
        crate::ENV_LOCK.lock().unwrap_or_else(|e| e.into_inner())
    }

    /// app_data/bin 中的 pandoc 应被优先于 PATH 发现（打包态布局）。
    /// 持 env 锁并清 NODEPAPER_PANDOC，防止外部注入劫持断言。
    #[test]
    fn pandoc_resolves_from_data_bin() {
        let _guard = env_lock();
        std::env::remove_var("NODEPAPER_PANDOC");
        let data = std::env::temp_dir().join("np-tools-find-test");
        let _ = fs::remove_dir_all(&data);
        let bin = data.join("bin");
        fs::create_dir_all(&bin).unwrap();
        let fake = bin.join(exe_name("pandoc"));
        fs::write(&fake, "stub").unwrap();
        assert_eq!(find_pandoc(Some(&data)), Some(fake));
        let _ = fs::remove_dir_all(&data);
    }

    /// zip → 定位可执行 → 拷入 bin/（bsdtar 按扩展名生成 zip，双平台可用）
    #[test]
    fn zip_binary_places_into_bin() {
        let data = std::env::temp_dir().join("np-tools-zip-test");
        let _ = fs::remove_dir_all(&data);
        let work = data.join("work");
        let pkg = work.join("pandoc-3.9").join("bin");
        fs::create_dir_all(&pkg).unwrap();
        fs::write(pkg.join("pandoc"), "bin").unwrap();
        fs::write(pkg.join("pandoc.pdf"), "noise").unwrap();
        let arch = data.join("pandoc-test.zip");
        let st = Command::new("tar")
            .args([
                "-acf",
                arch.to_str().unwrap(),
                "-C",
                work.to_str().unwrap(),
                "pandoc-3.9",
            ])
            .output()
            .unwrap();
        assert!(st.status.success(), "造 zip 夹具失败");
        let dest = place_files(&arch, Layout::ZipBinary("pandoc"), &data).expect("就位应成功");
        assert_eq!(dest, data.join("bin").join("pandoc"));
        assert!(dest.is_file());
        let _ = fs::remove_dir_all(&data);
    }

    /// tar.gz 目录树 → TinyTeX 就位为 tinytex/（覆盖旧目录）
    #[test]
    fn tar_tree_places_tinytex() {
        let data = std::env::temp_dir().join("np-tools-tree-test");
        let _ = fs::remove_dir_all(&data);
        let work = data.join("work");
        let bin = work.join("TinyTeX").join("bin").join("universal-darwin");
        fs::create_dir_all(&bin).unwrap();
        fs::write(bin.join("xelatex"), "tex").unwrap();
        let arch = data.join("tinytex-test.tar.gz");
        let st = Command::new("tar")
            .args([
                "-czf",
                arch.to_str().unwrap(),
                "-C",
                work.to_str().unwrap(),
                "TinyTeX",
            ])
            .output()
            .unwrap();
        assert!(st.status.success(), "造 tar.gz 夹具失败");
        // 预置旧 tinytex，就位应覆盖
        fs::create_dir_all(data.join("tinytex").join("old")).unwrap();
        let dest = place_files(&arch, Layout::TarTree("TinyTeX"), &data).expect("就位应成功");
        assert_eq!(dest, data.join("tinytex"));
        assert!(dest.join("bin/universal-darwin/xelatex").is_file());
        assert!(!dest.join("old").exists(), "旧安装应被移除");
        let _ = fs::remove_dir_all(&data);
    }

    /// 诊断版本探测：假 pandoc 脚本输出版本行
    #[test]
    fn probe_version_reads_first_line() {
        let tmp = std::env::temp_dir().join("np-tools-ver-test");
        let _ = fs::remove_dir_all(&tmp);
        fs::create_dir_all(&tmp).unwrap();
        let fake = tmp.join("pandoc");
        fs::write(&fake, "#!/bin/sh\necho 'pandoc 3.9.0'\n").unwrap();
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            fs::set_permissions(&fake, fs::Permissions::from_mode(0o755)).unwrap();
        }
        assert_eq!(
            probe_version(&fake, &["--version"]),
            Some("pandoc 3.9.0".to_string())
        );
        let _ = fs::remove_dir_all(&tmp);
    }

    /// 清单健全性：URL 平台匹配、体积量级合理（防清单改错无感）
    #[test]
    fn manifest_sanity() {
        let man = manifest();
        assert_eq!(man.len(), 2);
        for s in &man {
            assert!(s.url.starts_with("https://github.com/"), "{}", s.url);
            assert!(s.size > 30_000_000, "{} 体积异常：{}", s.key, s.size);
        }
    }
}
