// 编辑模式分栏舞台：左栏 Markdown 源码，右栏实时渲染预览（复用 Stage 的 preview 模式）。
import { useEffect, useRef } from "react";
import { Stage } from "./Stage";
import type { ScrollState } from "./Stage";
import type { CtxItem } from "./ContextMenu";

interface EditStageProps {
  md: string;
  html: string;
  onEdit: (v: string) => void;
  onScrollState: (s: ScrollState) => void;
  buildContextMenu?: () => CtxItem[];
}

export function EditStage({ md, html, onEdit, onScrollState, buildContextMenu }: EditStageProps) {
  const areaRef = useRef<HTMLTextAreaElement>(null);

  // 进入编辑模式即聚焦源码栏
  useEffect(() => {
    areaRef.current?.focus();
  }, []);

  return (
    <div className="edit-split">
      <section className="edit-pane" aria-label="Markdown 源码">
        <div className="edit-head">
          <b>源码</b>
          <span className="edit-head-en">Markdown</span>
        </div>
        <textarea
          ref={areaRef}
          className="edit-area"
          value={md}
          onChange={(e) => onEdit(e.target.value)}
          spellCheck={false}
          placeholder={"# 标题\n\n在此书写 Markdown，右侧即时预览……"}
          aria-label="Markdown 源码编辑"
        />
      </section>
      <div className="edit-view">
        <Stage
          html={html}
          mode="preview"
          onScrollState={onScrollState}
          buildContextMenu={buildContextMenu}
        />
      </div>
    </div>
  );
}
