// 编辑模式分栏舞台：左栏 Markdown 源码，右栏实时渲染预览（复用 Stage 的 preview 模式）。
// 两栏按滚动百分比双向同步；rAF 锁防止同步引发的 scroll 事件回弹死循环。
// 源码栏字号可在头部局部调节（--np-edit-size）；字号变化后按当前百分比重对齐右栏。
import { useEffect, useRef } from "react";
import { Stage } from "./Stage";
import type { ScrollState } from "./Stage";
import type { CtxItem } from "./ContextMenu";

interface EditStageProps {
  md: string;
  html: string;
  editSizeIdx: number;
  editSizeCount: number;
  onEditSizeUp: () => void;
  onEditSizeDown: () => void;
  onEdit: (v: string) => void;
  onScrollState: (s: ScrollState) => void;
  buildContextMenu?: () => CtxItem[];
}

export function EditStage({
  md,
  html,
  editSizeIdx,
  editSizeCount,
  onEditSizeUp,
  onEditSizeDown,
  onEdit,
  onScrollState,
  buildContextMenu,
}: EditStageProps) {
  const areaRef = useRef<HTMLTextAreaElement>(null);
  const previewScrollRef = useRef<HTMLElement | null>(null);
  const syncing = useRef(false);

  // 进入编辑模式即聚焦源码栏
  useEffect(() => {
    areaRef.current?.focus();
  }, []);

  // 双向同步滚动：按各自可滚动高度的百分比映射
  useEffect(() => {
    const area = areaRef.current;
    const view = previewScrollRef.current;
    if (!area || !view) return;
    let raf = 0;
    const sync = (src: HTMLElement, dst: HTMLElement) => {
      if (syncing.current) return;
      syncing.current = true;
      const sMax = src.scrollHeight - src.clientHeight;
      const dMax = dst.scrollHeight - dst.clientHeight;
      if (sMax > 0 && dMax >= 0) dst.scrollTop = (src.scrollTop / sMax) * dMax;
      cancelAnimationFrame(raf);
      raf = requestAnimationFrame(() => {
        syncing.current = false;
      });
    };
    const onAreaScroll = () => sync(area, view);
    const onViewScroll = () => sync(view, area);
    area.addEventListener("scroll", onAreaScroll, { passive: true });
    view.addEventListener("scroll", onViewScroll, { passive: true });
    return () => {
      area.removeEventListener("scroll", onAreaScroll);
      view.removeEventListener("scroll", onViewScroll);
      cancelAnimationFrame(raf);
    };
  }, []);

  // 源码字号变化：左栏重排后按当前百分比把右栏拉回对齐
  useEffect(() => {
    const area = areaRef.current;
    const view = previewScrollRef.current;
    if (!area || !view) return;
    const sMax = area.scrollHeight - area.clientHeight;
    const dMax = view.scrollHeight - view.clientHeight;
    if (sMax > 0 && dMax > 0) view.scrollTop = (area.scrollTop / sMax) * dMax;
  }, [editSizeIdx]);

  const pct = Math.round(((editSizeIdx + 1) / editSizeCount) * 100);

  return (
    <div className="edit-split">
      <section className="edit-pane" aria-label="Markdown 源码">
        <div className="edit-head">
          <b>源码</b>
          <span className="edit-head-en">Markdown</span>
          <div className="size-ctrl edit-size" role="group" aria-label="源码字号">
            <button type="button" aria-label="缩小源码字号" onClick={onEditSizeDown}>
              −
            </button>
            <span>{pct}%</span>
            <button type="button" aria-label="放大源码字号" onClick={onEditSizeUp}>
              +
            </button>
          </div>
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
          scrollRef={previewScrollRef}
          onScrollState={onScrollState}
          buildContextMenu={buildContextMenu}
        />
      </div>
    </div>
  );
}
