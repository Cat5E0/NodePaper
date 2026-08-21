// 信息浮层：用于「快捷键」「关于」。复用粘贴浮层的遮罩 + 卡片样式。
import { useEffect } from "react";
import { X } from "lucide-react";
import type { ReactNode } from "react";

interface InfoLayerProps {
  open: boolean;
  title: string;
  onClose: () => void;
  children: ReactNode;
}

export function InfoLayer({ open, title, onClose, children }: InfoLayerProps) {
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div className="paste open" onClick={onClose}>
      <div className="paste-card info-card" onClick={(e) => e.stopPropagation()}>
        <header>
          <h3>{title}</h3>
          <button className="toc-close" aria-label="关闭" onClick={onClose}>
            <X />
          </button>
        </header>
        <div className="info-body">{children}</div>
      </div>
    </div>
  );
}
