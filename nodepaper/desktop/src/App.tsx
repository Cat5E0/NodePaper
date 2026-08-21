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
import type { ScrollState } from "./components/Stage";
import { Toc } from "./components/Toc";
import { PasteLayer } from "./components/PasteLayer";
import { DropVeil } from "./components/DropVeil";
import { InfoLayer } from "./components/InfoLayer";
import { ContextMenuProvider } from "./components/ContextMenu";
import type { CtxItem } from "./components/ContextMenu";
import { renderMarkdown } from "./lib/markdown";
import { SAMPLE } from "./lib/sample";
import { buildShelfTree } from "./lib/shelf";
import {
  pickDirectory,
  listMarkdown,
  readMarkdown,
  MD_RE,
} from "./lib/fileSystem";
import type { FileEntry } from "./lib/fileSystem";

const THEME_KEY = "md-read-theme";
const SIZES = [0.875, 0.9375, 1, 1.0625, 1.125, 1.1875];
// 编辑模式源码栏字号阶梯（rem），默认 0.84 档
const EDIT_SIZES = [0.72, 0.78, 0.84, 0.9, 0.98, 1.06];

interface ShelfData {
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
  const [editMode, setEditMode] = useState(false);

  const [scrollState, setScrollState] = useState<ScrollState>({
    scrolled: false,
    activeId: null,
  });

  const [pasteOpen, setPasteOpen] = useState(false);
  const [pasteValue, setPasteValue] = useState("");
  const [dropOpen, setDropOpen] = useState(false);
  const [infoKind, setInfoKind] = useState<null | "shortcuts" | "about">(null);

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

  /* 打开文件夹 → 书架（并展开书架列） */
  const handleOpen = useCallback(async () => {
    const dir = await pickDirectory();
    if (!dir) return;
    const items = await listMarkdown(dir);
    items.sort((a, b) => a.rel.localeCompare(b.rel, "zh-Hans"));
    const tree = buildShelfTree(items);
    const rootName = dir.split(/[\\/]/).pop() || "书架";
    setShelf({ title: rootName, tree });
    setShelfCollapsed(false);
    if (items.length) loadEntry(items[0]);
  }, [loadEntry]);

  /* 粘贴确认 */
  const handlePasteConfirm = useCallback(() => {
    if (pasteValue.trim()) {
      setMd(pasteValue);
      setPasteOpen(false);
    }
  }, [pasteValue]);

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
      { label: "粘贴文本…", onClick: () => setPasteOpen(true) },
      { label: "打开文件夹…", onClick: handleOpen },
      { type: "sep" },
      { label: "放大字号", onClick: sizeUp },
      { label: "缩小字号", onClick: sizeDown },
      { label: "目录", check: !tocCollapsed, onClick: () => setTocCollapsed((v) => !v) },
      { label: "编辑模式", check: editMode, onClick: () => setEditMode((v) => !v) },
    ];
  }, [handleOpen, sizeUp, sizeDown, tocCollapsed, editMode]);

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
        setPasteOpen(false);
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
        setEditMode((v) => !v);
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
        editMode={editMode}
        onHome={() => setMd(SAMPLE)}
        onOpen={handleOpen}
        onToggleShelf={() => setShelfCollapsed((v) => !v)}
        onToggleToc={() => setTocCollapsed((v) => !v)}
        onToggleEdit={() => setEditMode((v) => !v)}
        onPaste={() => setPasteOpen(true)}
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
      />

      <div className="workspace">
        <Shelf
          collapsed={shelfCollapsed}
          title={shelf?.title ?? "书架"}
          tree={shelf?.tree ?? null}
          activePath={activePath}
          onOpenFile={loadEntry}
          onOpenFolder={handleOpen}
        />

        <Toc collapsed={tocCollapsed} items={toc} activeId={scrollState.activeId} />

        <div className="stage-wrap">
          {!editMode && <Brush />}
          {editMode ? (
            <EditStage
              md={md}
              html={html}
              editSizeIdx={editSizeIdx}
              editSizeCount={EDIT_SIZES.length}
              onEditSizeUp={editSizeUp}
              onEditSizeDown={editSizeDown}
              onEdit={setMd}
              onScrollState={setScrollState}
              buildContextMenu={buildReaderItems}
            />
          ) : (
            <Stage
              html={html}
              onScrollState={setScrollState}
              buildContextMenu={buildReaderItems}
            />
          )}
        </div>
      </div>

      <PasteLayer
        open={pasteOpen}
        value={pasteValue}
        onChange={setPasteValue}
        onCancel={() => setPasteOpen(false)}
        onConfirm={handlePasteConfirm}
      />

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
