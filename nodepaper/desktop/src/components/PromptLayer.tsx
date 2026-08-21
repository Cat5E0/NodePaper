// 命名浮层：新建/重命名 Markdown 的命名环节（dialog 插件无文本输入框，用 in-app 卡片）。
// 复用 InfoLayer 的遮罩 + paste-card 卡片样式；Enter 确认 / Esc 取消。
import { useEffect, useRef, useState } from "react";
import { X } from "lucide-react";

interface PromptLayerProps {
  open: boolean;
  title: string;
  /** 输入框预填值（stem，不含 .md） */
  initial: string;
  /** 确认时后端返回的错误（重名/非法字符），浮层保持打开供修改 */
  error: string | null;
  onCancel: () => void;
  onConfirm: (name: string) => void;
}

export function PromptLayer({ open, title, initial, error, onCancel, onConfirm }: PromptLayerProps) {
  const [value, setValue] = useState(initial);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (open) {
      setValue(initial);
      // 下一帧聚焦并全选，便于直接键入覆盖
      requestAnimationFrame(() => {
        inputRef.current?.focus();
        inputRef.current?.select();
      });
    }
  }, [open, initial]);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onCancel();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open, onCancel]);

  if (!open) return null;

  return (
    <div className="paste open" onClick={onCancel}>
      <div className="paste-card prompt-card" onClick={(e) => e.stopPropagation()}>
        <header>
          <h3>{title}</h3>
          <button className="toc-close" aria-label="关闭" onClick={onCancel}>
            <X />
          </button>
        </header>
        <div className="prompt-body">
          <input
            ref={inputRef}
            className="prompt-input"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && value.trim()) onConfirm(value.trim());
            }}
            placeholder="输入笔记名…"
            aria-label="笔记名称"
            spellCheck={false}
          />
          <span className="prompt-ext">.md</span>
        </div>
        {error && <p className="prompt-err">{error}</p>}
        <footer className="prompt-foot">
          <button type="button" className="btn" onClick={onCancel}>
            取消
          </button>
          <button
            type="button"
            className="btn primary"
            disabled={!value.trim()}
            onClick={() => onConfirm(value.trim())}
          >
            确定
          </button>
        </footer>
      </div>
    </div>
  );
}
