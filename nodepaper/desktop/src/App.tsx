// NodePaper 桌面端 —— 应用根组件，持有全部状态与交互。
// 布局：菜单栏顶栏 + 工作区（书架列 | 目录列 | 阅读区），书架/目录列各自可折叠。
import { useCallback, useEffect, useMemo, useState } from "react";
import type { UnlistenFn } from "@tauri-apps/api/event";
import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow";
import { Masthead } from "./components/Masthead";
import { Brush } from "./components/Brush";
import { Shelf } from "./components/Shelf";
import { Stage } from "./components/Stage";
import { EditStage } from "./components/EditStage";
import type { CompileState } from "./components/EditStage";
import type { ScrollState } from "./components/Stage";
import { Toc } from "./components/Toc";
import { DropVeil } from "./components/DropVeil";
import { DoctorLayer } from "./components/DoctorLayer";
import { PromptLayer } from "./components/PromptLayer";
import { useInstallStore } from "./lib/useInstallStore";
import { message } from "@tauri-apps/plugin-dialog";
import { PenLine } from "lucide-react";
import { InfoLayer } from "./components/InfoLayer";
import { ContextMenuProvider } from "./components/ContextMenu";
import type { CtxItem } from "./components/ContextMenu";
import { renderMarkdown } from "./lib/markdown";
import { invoke } from "@tauri-apps/api/core";
import { ask } from "@tauri-apps/plugin-dialog";
import { SAMPLE } from "./lib/sample";
import { buildShelfTree } from "./lib/shelf";
import {
  pickDirectory,
  listMarkdown,
  readMarkdown,
  createMarkdown,
  renameMarkdown,
  deleteMarkdown,
  MD_RE,
} from "./lib/fileSystem";
import type { FileEntry } from "./lib/fileSystem";

const THEME_KEY = "md-read-theme";
const SIZES = [0.875, 0.9375, 1, 1.0625, 1.125, 1.1875];
// 编辑模式源码栏字号阶梯（rem），默认 0.84 档
const EDIT_SIZES = [0.72, 0.78, 0.84, 0.9, 0.98, 1.06];

interface ShelfData {
  /** 书架根目录绝对路径（刷新/新建/删除用） */
  dir: string;
  title: string;
  tree: import("./lib/shelf").ShelfNode;
}

export default function App() {
  const [md, setMd] = useState<string>(SAMPLE);
  const { html, toc } = useMemo(() => renderMarkdown(md), [md]);

  const [theme, setTheme] = useState<string>(() => {
    try {
      return localStorage.getItem(THEME_KEY) || "sujian";
    } catch {
      return "sujian";
    }
  });
  const [sizeIdx, setSizeIdx] = useState(3);
  const [editSizeIdx, setEditSizeIdx] = useState(2);

  const [shelf, setShelf] = useState<ShelfData | null>(null);
  const [activePath, setActivePath] = useState<string | null>(null);
  const [shelfCollapsed, setShelfCollapsed] = useState(true);
  const [tocCollapsed, setTocCollapsed] = useState(false);
  /* 两态工作模式：阅读 / 编辑（双栏：源码 + 预览·LaTeX·日志） */
  const [mode, setMode] = useState<"read" | "edit">("read");
  const [compile, setCompile] = useState<CompileState | null>(null);

  const [scrollState, setScrollState] = useState<ScrollState>({
    scrolled: false,
    activeId: null,
  });

  const [dropOpen, setDropOpen] = useState(false);
  const [infoKind, setInfoKind] = useState<null | "shortcuts" | "about">(null);
  const [doctorOpen, setDoctorOpen] = useState(false);
  /* 全局下载状态：后端任务与浮层生命周期解耦，关闭浮层下载不中断 */
  const { install } = useInstallStore();

  /* 主题 / 字号 —— 写到 :root */
  useEffect(() => {
    document.documentElement.setAttribute("data-theme", theme);
  }, [theme]);

  useEffect(() => {
    document.documentElement.style.setProperty(
      "--np-reader-size",
      SIZES[sizeIdx] + "rem"
    );
  }, [sizeIdx]);

  /* 源码栏字号 —— 写到 :root */
  useEffect(() => {
    document.documentElement.style.setProperty(
      "--np-edit-size",
      EDIT_SIZES[editSizeIdx] + "rem"
    );
  }, [editSizeIdx]);

  /* 载入文件 */
  const loadEntry = useCallback(async (entry: FileEntry) => {
    try {
      const text = await readMarkdown(entry.path);
      setActivePath(entry.path);
      setMd(text);
    } catch (e) {
      console.error("读取文件失败", e);
    }
  }, []);

  /* 打开文件夹 → 书架（并展开书架列）；dir 存入 state 供新建/删除后刷新 */
  const refreshShelf = useCallback(async (dir: string) => {
    const items = await listMarkdown(dir);
    items.sort((a, b) => a.rel.localeCompare(b.rel, "zh-Hans"));
    const tree = buildShelfTree(items);
    const rootName = dir.split(/[\\/]/).pop() || "书架";
    setShelf({ dir, title: rootName, tree });
    return items;
  }, []);

  const handleOpen = useCallback(async () => {
    const dir = await pickDirectory();
    if (!dir) return;
    setShelfCollapsed(false);
    try {
      const items = await refreshShelf(dir);
      if (items.length) loadEntry(items[0]);
    } catch (e) {
      console.error("打开文件夹失败", e);
    }
  }, [loadEntry, refreshShelf]);

  /* 命名浮层：新建（预填空）与重命名（预填当前名）共用
     promptErr 存后端校验错误（重名/非法字符），浮层保持打开供修改 */
  const [prompt, setPrompt] = useState<null | { kind: "create"; dir: string } | { kind: "rename"; entry: FileEntry }>(null);
  const [promptErr, setPromptErr] = useState<string | null>(null);

  const openCreatePrompt = useCallback(() => {
    const dir = shelf?.dir;
    if (!dir) return;
    setPromptErr(null);
    setPrompt({ kind: "create", dir });
  }, [shelf]);

  const openRenamePrompt = useCallback((entry: FileEntry) => {
    setPromptErr(null);
    setPrompt({ kind: "rename", entry });
  }, []);

  const handlePromptConfirm = useCallback(
    async (name: string) => {
      if (!prompt) return;
      setPromptErr(null);
      try {
        if (prompt.kind === "create") {
          const entry = await createMarkdown(prompt.dir, name);
          setPrompt(null);
          await refreshShelf(prompt.dir);
          loadEntry(entry);
          setMode("edit");
        } else {
          const entry = await renameMarkdown(prompt.entry.path, name);
          setPrompt(null);
          if (shelf) await refreshShelf(shelf.dir);
          // 重命名的是当前打开的文件：跟随新路径重开（内容不变）
          if (activePath === prompt.entry.path) loadEntry(entry);
        }
      } catch (e) {
        setPromptErr(String(e));
      }
    },
    [prompt, shelf, activePath, refreshShelf, loadEntry]
  );

  /* 删除文件：原生确认框防误删；删的是当前文件则回示例卷首 */
  const handleDelete = useCallback(
    async (entry: FileEntry) => {
      const ok = await ask(`确定删除「${entry.rel}」？此操作不可恢复。`, {
        title: "删除笔记",
        kind: "warning",
      }).catch(() => false); // 浏览器预览无 dialog 插件时不删
      if (!ok) return;
      try {
        await deleteMarkdown(entry.path);
        if (shelf) await refreshShelf(shelf.dir);
        if (activePath === entry.path) {
          setActivePath(null);
          setMd(SAMPLE);
        }
      } catch (e) {
        console.error("删除失败", e);
      }
    },
    [shelf, activePath, refreshShelf]
  );

  /* 字号增减 */
  const sizeUp = useCallback(
    () => setSizeIdx((i) => Math.min(SIZES.length - 1, i + 1)),
    []
  );
  const sizeDown = useCallback(() => setSizeIdx((i) => Math.max(0, i - 1)), []);
  const editSizeUp = useCallback(
    () => setEditSizeIdx((i) => Math.min(EDIT_SIZES.length - 1, i + 1)),
    []
  );
  const editSizeDown = useCallback(
    () => setEditSizeIdx((i) => Math.max(0, i - 1)),
    []
  );

  /* 编译：调用 Rust 侧 compile_latex（对齐核心 citeproc 主链的 pandoc 直连） */
  const runCompile = useCallback(async (mdText: string) => {
    setCompile({ status: "running" });
    try {
      const r = await invoke<{ tex: string; log: string; tool: string }>(
        "compile_latex",
        { md: mdText }
      );
      setCompile({ status: "ok", tex: r.tex, log: r.log, tool: r.tool });
    } catch (e) {
      setCompile({ status: "err", message: String(e) });
    }
  }, []);

  /* 阅读区右键菜单工厂：组装当前可用的菜单项 */
  const buildReaderItems = useCallback((): CtxItem[] => {
    const sel = window.getSelection?.()?.toString() ?? "";
    const hasSel = !!sel.trim();
    return [
      {
        label: "复制选中",
        kbd: "Ctrl+C",
        disabled: !hasSel,
        onClick: () => {
          navigator.clipboard?.writeText(sel).catch(() => {});
        },
      },
      { type: "sep" },
      { label: "打开文件夹…", onClick: handleOpen },
      { type: "sep" },
      { label: "放大字号", onClick: sizeUp },
      { label: "缩小字号", onClick: sizeDown },
      { label: "目录", check: !tocCollapsed, onClick: () => setTocCollapsed((v) => !v) },
      { label: "编辑模式", check: mode === "edit", onClick: () => setMode((m) => (m === "edit" ? "read" : "edit")) },
    ];
  }, [handleOpen, sizeUp, sizeDown, tocCollapsed, mode]);

  /* 拖放（Tauri webview 原生事件，浏览器 dragdrop 在此被禁用） */
  useEffect(() => {
    const win = getCurrentWebviewWindow();
    let unlisten: UnlistenFn | undefined;
    win.onDragDropEvent((e) => {
      const p = e.payload;
      if (p.type === "enter" || p.type === "over") {
        setDropOpen(true);
      } else if (p.type === "leave") {
        setDropOpen(false);
      } else if (p.type === "drop") {
        setDropOpen(false);
        const target = p.paths.find((f) => MD_RE.test(f) || /\.txt$/i.test(f));
        if (target) {
          readMarkdown(target).then(setMd).catch((err) => console.error(err));
        }
      }
    }).then((fn) => {
      unlisten = fn;
    });
    return () => {
      unlisten?.();
    };
  }, []);

  /* 快捷键 */
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const el = document.activeElement;
      const typing = !!el && /TEXTAREA|INPUT/.test(el.tagName);
      const k = e.key.toLowerCase();
      if (k === "escape") {
        setInfoKind(null);
        return;
      }
      if (typing || e.ctrlKey || e.metaKey || e.altKey) return;
      if (k === "o") {
        e.preventDefault();
        handleOpen();
      } else if (k === "t") {
        e.preventDefault();
        setTocCollapsed((v) => !v);
      } else if (k === "e") {
        e.preventDefault();
        setMode((m) => (m === "edit" ? "read" : "edit"));
      }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [handleOpen]);

  return (
    <ContextMenuProvider>
      <Masthead
        scrolled={scrollState.scrolled}
        theme={theme}
        shelfCollapsed={shelfCollapsed}
        tocCollapsed={tocCollapsed}
        editMode={mode === "edit"}
        onHome={() => setMd(SAMPLE)}
        onOpen={handleOpen}
        onToggleShelf={() => setShelfCollapsed((v) => !v)}
        onToggleToc={() => setTocCollapsed((v) => !v)}
        onToggleEdit={() => setMode((m) => (m === "edit" ? "read" : "edit"))}
        onSizeUp={sizeUp}
        onSizeDown={sizeDown}
        onSelectTheme={(key) => {
          setTheme(key);
          try {
            localStorage.setItem(THEME_KEY, key);
          } catch {
            /* 隐私模式忽略 */
          }
        }}
        onShowShortcuts={() => setInfoKind("shortcuts")}
        onShowAbout={() => setInfoKind("about")}
        onDoctor={() => setDoctorOpen(true)}
      />

      <div className="workspace">
        <Shelf
          collapsed={shelfCollapsed}
          title={shelf?.title ?? "书架"}
          tree={shelf?.tree ?? null}
          activePath={activePath}
          onOpenFile={loadEntry}
          onOpenFolder={handleOpen}
          onCreate={openCreatePrompt}
          onDelete={handleDelete}
          onRename={openRenamePrompt}
        />

        <Toc collapsed={tocCollapsed} items={toc} activeId={scrollState.activeId} />

        <div className="stage-wrap">
          {mode === "read" && <Brush />}
          {mode === "edit" ? (
            <EditStage
              md={md}
              html={html}
              compileState={compile}
              editSizeIdx={editSizeIdx}
              editSizeCount={EDIT_SIZES.length}
              onEditSizeUp={editSizeUp}
              onEditSizeDown={editSizeDown}
              onEdit={setMd}
              onCompile={() => runCompile(md)}
              onScrollState={setScrollState}
              buildContextMenu={buildReaderItems}
            />
          ) : md.trim() ? (
            <Stage
              html={html}
              onScrollState={setScrollState}
              buildContextMenu={buildReaderItems}
            />
          ) : (
            /* 空文档引导：新建笔记/清空内容后的阅读态，指向编辑而非空白页 */
            <div className="stage empty-stage" role="status">
              <div className="empty-hint">
                <PenLine size={22} />
                <p>这篇笔记还是空的</p>
                <button
                  type="button"
                  className="btn primary"
                  onClick={() => setMode("edit")}
                >
                  进入编辑模式
                </button>
                <span className="empty-kbd">
                  或按 <kbd>E</kbd>
                </span>
              </div>
            </div>
          )}
        </div>
      </div>

      <PromptLayer
        open={prompt !== null}
        title={prompt?.kind === "rename" ? "重命名笔记" : "新建笔记"}
        initial={
          prompt?.kind === "rename"
            ? prompt.entry.name.replace(/\.(md|markdown)$/i, "")
            : "未命名"
        }
        error={promptErr}
        onCancel={() => {
          setPrompt(null);
          setPromptErr(null);
        }}
        onConfirm={handlePromptConfirm}
      />

      <DoctorLayer
        open={doctorOpen}
        onClose={() => setDoctorOpen(false)}
        install={install}
        onInstallError={() => {}}
      />

      {/* 下载完成但浮层已关：系统级通知透出结果，避免静默失败 */}
      <InstallDoneToast install={install} />

      <DropVeil open={dropOpen} />

      <InfoLayer
        open={infoKind === "shortcuts"}
        title="快捷键"
        onClose={() => setInfoKind(null)}
      >
        <dl className="kbd-list">
          <dt>O</dt>
          <dd>打开文件夹</dd>
          <dt>T</dt>
          <dd>显示 / 隐藏目录</dd>
          <dt>E</dt>
          <dd>进入 / 退出编辑模式</dd>
          <dt>Ctrl+Enter</dt>
          <dd>编辑模式内编译 LaTeX</dd>
          <dt>Esc</dt>
          <dd>关闭浮层、菜单</dd>
          <dt>右键</dt>
          <dd>上下文菜单</dd>
        </dl>
      </InfoLayer>

      <InfoLayer
        open={infoKind === "about"}
        title="关于 NodePaper"
        onClose={() => setInfoKind(null)}
      >
        <div className="about-body">
          <p className="about-line">
            <span className="seal">NodePaper</span>
            <span className="muted">Paper &amp; Ink · 0.1.0</span>
          </p>
          <p className="muted">把 Markdown 论文工程构建为 PDF 的工具，这里是它的桌面书房。</p>
          <p className="muted">
            打开文件夹浏览书架，粘贴文本即时预览，拖入 .md 文件直接展开。
          </p>
        </div>
      </InfoLayer>
    </ContextMenuProvider>
  );
}

/* 下载完成通知：浮层关闭时经系统 message 透出（开着则由浮层内刷新呈现）。
   独立组件：只在状态从 running 翻转到终态时触发一次。 */
function InstallDoneToast({
  install,
}: {
  install: import("./lib/tools").InstallState | null;
}) {
  const [notified, setNotified] = useState<string | null>(null);
  useEffect(() => {
    if (!install || install.status === "running") return;
    // 只在浮层关闭时系统级通知（开着时浮层内已有反馈）
    const tag = install.key + ":" + install.status + ":" + install.pct;
    if (notified === tag) return;
    setNotified(tag);
    if (!install.error) return;
    message(`${install.key} 下载失败：${install.error}`, {
      title: "工具安装",
      kind: "error",
    }).catch(() => {});
    // 成功不弹窗：重开诊断浮层即可见 ✓
  }, [install, notified]);
  return null;
}
