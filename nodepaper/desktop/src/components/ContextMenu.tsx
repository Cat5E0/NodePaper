// 通用右键上下文菜单：Provider 注入 open/close，任意子组件用 useContextMenu() 弹出。
// 浮层定位自动避让视口右下边界；点击外部 / ESC / 选中项 / 新右键 均自动关闭。
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";

export interface CtxItem {
  type?: "item" | "sep";
  label?: string;
  kbd?: string;
  check?: boolean;
  disabled?: boolean;
  onClick?: () => void;
}

interface CtxState {
  x: number;
  y: number;
  items: CtxItem[];
}

interface CtxApi {
  open: (x: number, y: number, items: CtxItem[]) => void;
  close: () => void;
}

const CtxContext = createContext<CtxApi | null>(null);

export function useContextMenu(): CtxApi {
  const ctx = useContext(CtxContext);
  if (!ctx) throw new Error("useContextMenu 必须在 ContextMenuProvider 内使用");
  return ctx;
}

export function ContextMenuProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<CtxState | null>(null);
  // 渲染后测量出的安全位置；未测得前隐藏，避免边缘闪烁
  const [pos, setPos] = useState<{ x: number; y: number } | null>(null);
  const elRef = useRef<HTMLDivElement>(null);

  const open = useCallback((x: number, y: number, items: CtxItem[]) => {
    setPos(null);
    setState({ x, y, items });
  }, []);
  const close = useCallback(() => setState(null), []);

  useEffect(() => {
    if (!state) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") close();
    };
    const onClick = () => close();
    // 新的右键事件（capture）先关闭旧菜单，调用方随后 open 新菜单
    const onCtx = () => close();
    document.addEventListener("keydown", onKey);
    window.addEventListener("click", onClick);
    document.addEventListener("contextmenu", onCtx, true);
    return () => {
      document.removeEventListener("keydown", onKey);
      window.removeEventListener("click", onClick);
      document.removeEventListener("contextmenu", onCtx, true);
    };
  }, [state, close]);

  // 测量尺寸并避让视口右下
  useEffect(() => {
    if (!state) {
      setPos(null);
      return;
    }
    const el = elRef.current;
    if (!el) return;
    const r = el.getBoundingClientRect();
    let x = state.x;
    let y = state.y;
    const margin = 8;
    if (x + r.width > window.innerWidth - margin)
      x = Math.max(margin, window.innerWidth - r.width - margin);
    if (y + r.height > window.innerHeight - margin)
      y = Math.max(margin, window.innerHeight - r.height - margin);
    setPos({ x, y });
  }, [state]);

  return (
    <CtxContext.Provider value={{ open, close }}>
      {children}
      {state && (
        <div
          ref={elRef}
          className="context-menu"
          style={{
            left: pos?.x ?? state.x,
            top: pos?.y ?? state.y,
            visibility: pos ? "visible" : "hidden",
          }}
          onClick={(e) => e.stopPropagation()}
          onContextMenu={(e) => e.stopPropagation()}
        >
          {state.items.map((it, i) =>
            it.type === "sep" ? (
              <div className="ctx-sep" key={i} />
            ) : (
              <button
                key={i}
                className={"ctx-item" + (it.disabled ? " disabled" : "")}
                disabled={it.disabled}
                onClick={() => {
                  if (it.disabled) return;
                  it.onClick?.();
                  close();
                }}
              >
                <span className="ctx-check">{it.check ? "✓" : ""}</span>
                <span className="ctx-label">{it.label}</span>
                {it.kbd && <span className="ctx-kbd">{it.kbd}</span>}
              </button>
            )
          )}
        </div>
      )}
    </CtxContext.Provider>
  );
}
