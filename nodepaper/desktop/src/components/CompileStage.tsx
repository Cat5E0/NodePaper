// 编译模式舞台：左栏 Markdown 源码（可编辑），右栏三视图切换——
// 渲染（复用 Stage preview）/ LaTeX 源 / 工具链日志。
// 编译走 Rust compile_latex 命令：pandoc 直连核心 citeproc 主链（profile: cumcm）。
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

type OutView = "render" | "tex" | "log";

interface CompileStageProps {
  md: string;
  compileState: CompileState | null;
  onEdit: (v: string) => void;
  onCompile: () => void;
  onScrollState: (s: ScrollState) => void;
  buildContextMenu?: () => CtxItem[];
}

export function CompileStage({
  md,
  html,
  compileState,
  onEdit,
  onCompile,
  onScrollState,
  buildContextMenu,
}: CompileStageProps & { html: string }) {
  const areaRef = useRef<HTMLTextAreaElement>(null);
  const [view, setView] = useState<OutView>("render");

  // 进入编译模式即聚焦源码栏
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

  const tabs: Array<{ key: OutView; label: string }> = [
    { key: "render", label: "渲染" },
    { key: "tex", label: "LaTeX" },
    { key: "log", label: "日志" },
  ];

  return (
    <div className="edit-split compile-split">
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
        </div>
        <textarea
          ref={areaRef}
          className="edit-area"
          value={md}
          onChange={(e) => onEdit(e.target.value)}
          spellCheck={false}
          placeholder={"# 标题\n\n在此书写 Markdown，点击「编译 LaTeX」或按 Ctrl+Enter 转换……"}
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
        {view === "render" ? (
          <div className="compile-render">
            <Stage
              html={html}
              mode="preview"
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
