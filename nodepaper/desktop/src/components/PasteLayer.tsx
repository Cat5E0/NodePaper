// 粘贴浮层
import { useEffect, useRef } from "react";
import { X } from "lucide-react";

interface PasteLayerProps {
  open: boolean;
  value: string;
  onChange: (v: string) => void;
  onCancel: () => void;
  onConfirm: () => void;
}

export function PasteLayer({ open, value, onChange, onCancel, onConfirm }: PasteLayerProps) {
  const areaRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    if (open) {
      onChange("");
      const t = setTimeout(() => areaRef.current?.focus(), 50);
      return () => clearTimeout(t);
    }
  }, [open, onChange]);

  if (!open) return null;

  return (
    <div className="paste open" id="pasteLayer">
      <div className="paste-card">
        <header>
          <h3>粘贴 Markdown</h3>
          <button className="toc-close" aria-label="关闭" onClick={onCancel}>
            <X />
          </button>
        </header>
        <textarea
          ref={areaRef}
          placeholder={'# 标题\n\n将 Markdown 文本粘贴于此，或直接拖入 .md 文件……'}
          spellCheck={false}
          value={value}
          onChange={(e) => onChange(e.target.value)}
        />
        <footer>
          <button className="btn" onClick={onCancel}>
            取消
          </button>
          <button className="btn primary" onClick={onConfirm}>
            阅读
          </button>
        </footer>
      </div>
    </div>
  );
}
