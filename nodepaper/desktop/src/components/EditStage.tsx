// 编辑模式分栏舞台：左栏 Markdown 源码，右栏输出视图切换——
// 预览（复用 Stage preview，与源码双向同步滚动）/ LaTeX 源 / 工具链日志。
// 编译走 Rust compile_latex 命令：pandoc 直连核心 citeproc 主链（profile: cumcm），
// 「编译 LaTeX」按钮或 Ctrl+Enter 触发；成功自动切 LaTeX 视图，失败切日志。
// 源码栏字号可在头部局部调节（--np-edit-size）；视图切回或字号变化后按百分比重对齐预览。
import { useEffect, useRef, useState } from "react";
import { Stage } from "./Stage";
import type { ScrollState } from "./Stage";
import type { CtxItem } from "./ContextMenu";

export interface CompileState {
  status: "running" | "ok" | "err";
  tex?: string;
  log?: string;
  tool?: string;
  message?: string;
}

type OutView = "preview" | "tex" | "log";

interface EditStageProps {
  md: string;
  html: string;
  compileState: CompileState | null;
  editSizeIdx: number;
  editSizeCount: number;
  onEditSizeUp: () => void;
  onEditSizeDown: () => void;
  onEdit: (v: string) => void;
  onCompile: () => void;
  onScrollState: (s: ScrollState) => void;
  buildContextMenu?: () => CtxItem[];
}

export function EditStage({
  md,
  html,
  compileState,
  editSizeIdx,
  editSizeCount,
  onEditSizeUp,
  onEditSizeDown,
  onEdit,
  onCompile,
  onScrollState,
  buildContextMenu,
}: EditStageProps) {
  const areaRef = useRef<HTMLTextAreaElement>(null);
  const previewScrollRef = useRef<HTMLElement | null>(null);
  const syncing = useRef(false);
  const [view, setView] = useState<OutView>("preview");

  // 进入编辑模式即聚焦源码栏
  useEffect(() => {
    areaRef.current?.focus();
  }, []);

  // 编译结束自动切到对应视图：成功看 LaTeX，失败看日志
  useEffect(() => {
    if (compileState?.status === "ok") setView("tex");
    else if (compileState?.status === "err") setView("log");
  }, [compileState?.status]);

  // Ctrl/Cmd+Enter 触发编译（textarea 内不占用单键）
  useEffect(() => {
    const area = areaRef.current;
    if (!area) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
        e.preventDefault();
        onCompile();
      }
    };
    area.addEventListener("keydown", onKey);
    return () => area.removeEventListener("keydown", onKey);
  }, [onCompile]);

  // 双向同步滚动（仅预览视图）：按各自可滚动高度的百分比映射；
  // rAF 锁防止同步引发的 scroll 事件回弹死循环。
  // 依赖 view：切回预览时 Stage 重新挂载、滚动容器是新元素，需重绑。
  useEffect(() => {
    if (view !== "preview") return;
    const area = areaRef.current;
    const pv = previewScrollRef.current;
    if (!area || !pv) return;
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
    const onAreaScroll = () => sync(area, pv);
    const onPreviewScroll = () => sync(pv, area);
    area.addEventListener("scroll", onAreaScroll, { passive: true });
    pv.addEventListener("scroll", onPreviewScroll, { passive: true });
    return () => {
      area.removeEventListener("scroll", onAreaScroll);
      pv.removeEventListener("scroll", onPreviewScroll);
      cancelAnimationFrame(raf);
    };
  }, [view]);

  // 视图切回预览（Stage 重挂后 scrollTop 归零）或字号变化引起重排后：
  // 延迟一帧等 Stage 填充完内容，再按左栏当前百分比重对齐预览。
  useEffect(() => {
    if (view !== "preview") return;
    const area = areaRef.current;
    const pv = previewScrollRef.current;
    if (!area || !pv) return;
    const raf = requestAnimationFrame(() => {
      const sMax = area.scrollHeight - area.clientHeight;
      const dMax = pv.scrollHeight - pv.clientHeight;
      if (sMax > 0 && dMax > 0) pv.scrollTop = (area.scrollTop / sMax) * dMax;
    });
    return () => cancelAnimationFrame(raf);
  }, [view, editSizeIdx]);

  const pct = Math.round(((editSizeIdx + 1) / editSizeCount) * 100);

  const tabs: Array<{ key: OutView; label: string }> = [
    { key: "preview", label: "预览" },
    { key: "tex", label: "LaTeX" },
    { key: "log", label: "日志" },
  ];

  return (
    <div className="edit-split">
      <section className="edit-pane" aria-label="Markdown 源码">
        <div className="edit-head">
          <b>源码</b>
          <span className="edit-head-en">Markdown</span>
          <button
            type="button"
            className="btn primary compile-btn"
            onClick={onCompile}
            disabled={compileState?.status === "running"}
          >
            {compileState?.status === "running" ? "编译中…" : "编译 LaTeX"}
          </button>
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
          placeholder={"# 标题\n\n在此书写 Markdown，右侧即时预览；Ctrl+Enter 编译 LaTeX……"}
          aria-label="Markdown 源码编辑"
        />
      </section>
      <div className="compile-view">
        <div className="edit-head compile-out-head">
          <b>输出</b>
          <span className="edit-head-en">
            {compileState?.tool ?? "pandoc · profile: cumcm"}
          </span>
          <div className="compile-tabs" role="tablist" aria-label="输出视图">
            {tabs.map((t) => (
              <button
                key={t.key}
                type="button"
                role="tab"
                className={"compile-tab" + (view === t.key ? " active" : "")}
                aria-selected={view === t.key}
                onClick={() => setView(t.key)}
              >
                {t.label}
              </button>
            ))}
          </div>
        </div>
        {view === "preview" ? (
          <div className="compile-render">
            <Stage
              html={html}
              mode="preview"
              scrollRef={previewScrollRef}
              onScrollState={onScrollState}
              buildContextMenu={buildContextMenu}
            />
          </div>
        ) : view === "log" ? (
          <pre className={"compile-log" + (compileState?.status === "err" ? " is-err" : "")}>
            {compileState?.status === "err"
              ? compileState.message
              : compileState?.log ?? "尚未编译，无日志。"}
          </pre>
        ) : compileState?.status === "ok" ? (
          <pre className="compile-tex">{compileState.tex}</pre>
        ) : (
          <div className="compile-empty">
            尚未编译。点击「编译 LaTeX」或按 Ctrl+Enter。
          </div>
        )}
      </div>
    </div>
  );
}
