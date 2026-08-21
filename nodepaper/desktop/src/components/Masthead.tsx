// 顶栏 = 应用菜单栏：品牌（左）+ 文件/视图/主题/帮助下拉（中）+ 窗口控制（右）。
// header 作为窗口拖拽区（data-tauri-drag-region）；交互元素可正常点击。
// 展开任一菜单后，鼠标横向悬停切换到相邻顶层菜单（hover-switch），符合桌面应用直觉。
import { useEffect, useState } from "react";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { Minus, Square, Copy, X } from "lucide-react";

export const THEMES = [
  { key: "sujian", name: "纸墨", desc: "暖纸与墨线" },
  { key: "apple", name: "Apple", desc: "清亮极简" },
  { key: "ink-night", name: "墨夜", desc: "夜读如墨" },
  { key: "graphite", name: "石墨", desc: "冷灰极简" },
  { key: "parchment", name: "羊皮卷", desc: "暖褐旧卷" },
  { key: "sakura", name: "樱", desc: "樱粉轻柔" },
] as const;

interface MastheadProps {
  scrolled: boolean;
  theme: string;
  shelfCollapsed: boolean;
  tocCollapsed: boolean;
  focus: boolean;
  onHome: () => void;
  onOpen: () => void;
  onToggleShelf: () => void;
  onToggleToc: () => void;
  onToggleFocus: () => void;
  onPaste: () => void;
  onSizeUp: () => void;
  onSizeDown: () => void;
  onSelectTheme: (key: string) => void;
  onShowShortcuts: () => void;
  onShowAbout: () => void;
}

type MenuKey = "file" | "view" | "theme" | "help" | null;

function Entry({
  label,
  kbd,
  check,
  disabled,
  onClick,
  onClose,
}: {
  label: string;
  kbd?: string;
  check?: boolean;
  disabled?: boolean;
  onClick?: () => void;
  onClose: () => void;
}) {
  return (
    <button
      type="button"
      className={"menu-entry" + (disabled ? " disabled" : "")}
      disabled={disabled}
      onClick={() => {
        if (disabled) return;
        onClick?.();
        onClose();
      }}
    >
      <span className="entry-check">{check ? "✓" : ""}</span>
      <span className="entry-label">{label}</span>
      {kbd && <span className="entry-kbd">{kbd}</span>}
    </button>
  );
}

export function Masthead(props: MastheadProps) {
  const {
    scrolled,
    theme,
    shelfCollapsed,
    tocCollapsed,
    focus,
    onHome,
    onOpen,
    onToggleShelf,
    onToggleToc,
    onToggleFocus,
    onPaste,
    onSizeUp,
    onSizeDown,
    onSelectTheme,
    onShowShortcuts,
    onShowAbout,
  } = props;

  const [openMenu, setOpenMenu] = useState<MenuKey>(null);
  const [maximized, setMaximized] = useState(false);

  useEffect(() => {
    const win = getCurrentWindow();
    win.isMaximized().then(setMaximized);
    const unlistenP = win.onResized(() => {
      win.isMaximized().then(setMaximized);
    });
    return () => {
      unlistenP.then((u) => u());
    };
  }, []);

  // 外部点击 / ESC 关闭菜单
  useEffect(() => {
    if (!openMenu) return;
    const onDocClick = (e: MouseEvent) => {
      if (!(e.target as HTMLElement).closest(".menubar")) setOpenMenu(null);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpenMenu(null);
    };
    document.addEventListener("click", onDocClick);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("click", onDocClick);
      document.removeEventListener("keydown", onKey);
    };
  }, [openMenu]);

  const close = () => setOpenMenu(null);

  // hover-switch：菜单展开时，悬停另一顶层项即切换
  const hoverTo = (k: MenuKey) => {
    if (openMenu) setOpenMenu(k);
  };

  return (
    <header
      className={"masthead" + (scrolled ? " is-scrolled" : "")}
      id="masthead"
      data-tauri-drag-region
    >
      <a
        className="brand"
        href="#"
        id="brandHome"
        onClick={(e) => {
          e.preventDefault();
          onHome();
        }}
      >
        <span className="seal">NodePaper</span>
        <span className="divider">/</span>
        <span className="en">Paper&nbsp;&amp;&nbsp;Ink</span>
      </a>

      <nav className="menubar" aria-label="主菜单">
        {/* 文件 */}
        <div className="menu" onMouseEnter={() => hoverTo("file")}>
          <button
            type="button"
            className={"menu-bar-btn" + (openMenu === "file" ? " open" : "")}
            onClick={() => setOpenMenu(openMenu === "file" ? null : "file")}
          >
            文件<span className="acc">F</span>
          </button>
          {openMenu === "file" && (
            <div className="menu-dropdown" role="menu">
              <Entry label="打开文件夹…" kbd="O" onClick={onOpen} onClose={close} />
              <Entry label="粘贴文本…" kbd="P" onClick={onPaste} onClose={close} />
              <div className="menu-sep" />
              <Entry label="回到卷首" onClick={onHome} onClose={close} />
            </div>
          )}
        </div>

        {/* 视图 */}
        <div className="menu" onMouseEnter={() => hoverTo("view")}>
          <button
            type="button"
            className={"menu-bar-btn" + (openMenu === "view" ? " open" : "")}
            onClick={() => setOpenMenu(openMenu === "view" ? null : "view")}
          >
            视图<span className="acc">V</span>
          </button>
          {openMenu === "view" && (
            <div className="menu-dropdown" role="menu">
              <Entry
                label="书架"
                check={!shelfCollapsed}
                onClick={onToggleShelf}
                onClose={close}
              />
              <Entry
                label="目录"
                kbd="T"
                check={!tocCollapsed}
                onClick={onToggleToc}
                onClose={close}
              />
              <Entry
                label="专注模式"
                kbd="F"
                check={focus}
                onClick={onToggleFocus}
                onClose={close}
              />
              <div className="menu-sep" />
              <Entry label="放大字号" kbd="+" onClick={onSizeUp} onClose={close} />
              <Entry label="缩小字号" kbd="−" onClick={onSizeDown} onClose={close} />
            </div>
          )}
        </div>

        {/* 主题 */}
        <div className="menu" onMouseEnter={() => hoverTo("theme")}>
          <button
            type="button"
            className={"menu-bar-btn" + (openMenu === "theme" ? " open" : "")}
            onClick={() => setOpenMenu(openMenu === "theme" ? null : "theme")}
          >
            主题<span className="acc">T</span>
          </button>
          {openMenu === "theme" && (
            <div className="menu-dropdown theme-dropdown" role="menu">
              {THEMES.map((t) => {
                const active = t.key === theme;
                return (
                  <button
                    key={t.key}
                    type="button"
                    className={"menu-entry theme-opt" + (active ? " active" : "")}
                    role="menuitemradio"
                    aria-checked={active ? "true" : "false"}
                    onClick={() => {
                      onSelectTheme(t.key);
                      close();
                    }}
                  >
                    <span className="entry-check">{active ? "✓" : ""}</span>
                    <span className="swatch" data-swatch={t.key} />
                    <span className="theme-name">
                      {t.name}
                      <small>{t.desc}</small>
                    </span>
                  </button>
                );
              })}
            </div>
          )}
        </div>

        {/* 帮助 */}
        <div className="menu" onMouseEnter={() => hoverTo("help")}>
          <button
            type="button"
            className={"menu-bar-btn" + (openMenu === "help" ? " open" : "")}
            onClick={() => setOpenMenu(openMenu === "help" ? null : "help")}
          >
            帮助<span className="acc">H</span>
          </button>
          {openMenu === "help" && (
            <div className="menu-dropdown" role="menu">
              <Entry label="快捷键…" onClick={onShowShortcuts} onClose={close} />
              <Entry label="关于 NodePaper" onClick={onShowAbout} onClose={close} />
            </div>
          )}
        </div>
      </nav>

      <div className="win-ctrl">
        <button title="最小化" aria-label="最小化" onClick={() => getCurrentWindow().minimize()}>
          <Minus />
        </button>
        <button
          title={maximized ? "还原" : "最大化"}
          aria-label={maximized ? "还原" : "最大化"}
          onClick={() => getCurrentWindow().toggleMaximize()}
        >
          {maximized ? <Copy /> : <Square />}
        </button>
        <button
          title="关闭"
          aria-label="关闭"
          className="close"
          onClick={() => getCurrentWindow().close()}
        >
          <X />
        </button>
      </div>
    </header>
  );
}
