// 环境诊断浮层：列出工具链状态；缺失项可一键下载自取（pandoc / TinyTeX）。
// 下载前原生确认框（含体积与安装位置说明）；进度条由 np:tool-install 事件驱动。
// pandoc-crossref 发布包为 .tar.xz/.7z，不做自取，提供打开发布页入口。
import { useCallback, useEffect, useState } from "react";
import { listen } from "@tauri-apps/api/event";
import { ask } from "@tauri-apps/plugin-dialog";
import { openUrl } from "@tauri-apps/plugin-opener";
import { CheckCircle2, XCircle, Download, ExternalLink, X } from "lucide-react";
import {
  diagnoseTools,
  installTool,
  TOOL_INSTALL_EVENT,
  type ToolStatus,
} from "../lib/tools";

const CROSSREF_RELEASES = "https://github.com/lierdakil/pandoc-crossref/releases";

interface DoctorLayerProps {
  open: boolean;
  onClose: () => void;
}

export function DoctorLayer({ open, onClose }: DoctorLayerProps) {
  const [tools, setTools] = useState<ToolStatus[] | null>(null);
  const [busyKey, setBusyKey] = useState<string | null>(null);
  const [pct, setPct] = useState(0);
  const [err, setErr] = useState<string | null>(null);

  const refresh = useCallback(() => {
    diagnoseTools()
      .then(setTools)
      .catch((e) => setErr(String(e)));
  }, []);

  useEffect(() => {
    if (open) {
      setErr(null);
      refresh();
    }
  }, [open, refresh]);

  // 下载进度事件（安装期间持续监听）
  useEffect(() => {
    if (!open) return;
    const p = listen<{ key: string; pct: number }>(TOOL_INSTALL_EVENT, (e) => {
      setPct(e.payload.pct);
    });
    return () => {
      p.then((u) => u());
    };
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  const handleInstall = async (t: ToolStatus) => {
    const size = t.sizeMb ? `约 ${Math.round(t.sizeMb)} MB，` : "";
    const ok = await ask(
      `将下载 ${t.label}（${size}安装到应用数据目录，完成后编译链自动发现，无需配置）。是否继续？`,
      { title: "安装外部工具", kind: "info" }
    ).catch(() => false); // 浏览器预览无 dialog 插件时视为取消
    if (!ok) return;
    setBusyKey(t.key);
    setPct(0);
    setErr(null);
    try {
      await installTool(t.key);
      refresh();
    } catch (e) {
      setErr(String(e));
    } finally {
      setBusyKey(null);
    }
  };

  if (!open) return null;

  return (
    <div className="paste open" onClick={onClose}>
      <div className="paste-card doctor-card" onClick={(e) => e.stopPropagation()}>
        <header>
          <h3>环境诊断</h3>
          <button className="toc-close" aria-label="关闭" onClick={onClose}>
            <X />
          </button>
        </header>
        <div className="doctor-list">
          {(tools ?? []).map((t) => (
            <div className="doctor-row" key={t.key}>
              <span className={"doctor-ic " + (t.found ? "ok" : "miss")}>
                {t.found ? <CheckCircle2 size={16} /> : <XCircle size={16} />}
              </span>
              <div className="doctor-meta">
                <div className="doctor-name">
                  {t.label}
                  {t.version && <span className="doctor-ver"> {t.version}</span>}
                </div>
                <div className="doctor-sub" title={t.path ?? t.hint ?? ""}>
                  {t.path ?? t.hint ?? "未安装"}
                </div>
                {busyKey === t.key && (
                  <div className="doctor-bar" role="progressbar" aria-valuenow={pct}>
                    <i style={{ width: `${pct}%` }} />
                  </div>
                )}
              </div>
              <div className="doctor-act">
                {!t.found && t.downloadable && busyKey !== t.key && (
                  <button
                    type="button"
                    className="btn primary doctor-btn"
                    onClick={() => handleInstall(t)}
                    disabled={busyKey !== null}
                  >
                    <Download size={13} />
                    下载{t.sizeMb ? `（约 ${Math.round(t.sizeMb)} MB）` : ""}
                  </button>
                )}
                {t.key === "crossref" && !t.found && (
                  <button
                    type="button"
                    className="btn doctor-btn"
                    onClick={() => openUrl(CROSSREF_RELEASES)}
                    title="打开发布页手动安装"
                  >
                    <ExternalLink size={13} />
                    发布页
                  </button>
                )}
              </div>
            </div>
          ))}
          {tools === null && !err && <div className="doctor-sub">诊断中…</div>}
        </div>
        {err && <p className="doctor-err">{err}</p>}
        <p className="doctor-note">
          安装位置为应用数据目录（应用安装目录受系统保护：macOS 写入会破坏签名，Windows 需提权），
          编译链按 应用数据目录 → PATH 顺序自动发现。
        </p>
      </div>
    </div>
  );
}
