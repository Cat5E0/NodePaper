// 阅读舞台：自身为滚动容器。渲染 Markdown HTML、按 h2 切 section、逐节浮起、滚动墨笔/目录高亮。
// mode="preview" 供编辑模式右侧预览：保留滚动位置、跳过逐节动画，避免每次键入闪烁归顶。
import { useEffect, useRef } from "react";
import type { RefObject } from "react";
import { inView, animate } from "motion";
import { groupIntoSections } from "../lib/markdown";
import { useContextMenu } from "./ContextMenu";
import type { CtxItem } from "./ContextMenu";

export interface ScrollState {
  scrolled: boolean;
  activeId: string | null;
}

interface StageProps {
  html: string;
  onScrollState: (s: ScrollState) => void;
  buildContextMenu?: () => CtxItem[];
  mode?: "read" | "preview";
  /** 外部可注入的滚动容器 ref（编辑模式同步滚动用），与内部 ref 同步赋值 */
  scrollRef?: RefObject<HTMLElement | null>;
}

export function Stage({ html, onScrollState, buildContextMenu, mode = "read", scrollRef }: StageProps) {
  const { open } = useContextMenu();
  const stageRef = useRef<HTMLElement>(null);
  const articleRef = useRef<HTMLDivElement>(null);
  // 合并内部与外部 ref：挂载时同时写入
  const setStageRef = (el: HTMLElement | null) => {
    stageRef.current = el;
    if (scrollRef) scrollRef.current = el;
  };
  // 持有最新回调，避免 html 不变时 effect 因回调引用变化而重建
  const cbRef = useRef(onScrollState);
  cbRef.current = onScrollState;

  useEffect(() => {
    const stage = stageRef.current;
    const art = articleRef.current;
    if (!stage || !art) return;

    const preview = mode === "preview";
    const prevTop = preview ? stage.scrollTop : 0;

    art.innerHTML = html;
    groupIntoSections(art);

    // 逐节浮起（仅阅读模式；预览模式下直接可见）。
    // 减弱动效偏好下直接呈现，不做位移揭示。
    let stopInView: (() => void) | undefined;
    const reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (!preview && !reduceMotion) {
      const sections = art.querySelectorAll("section");
      sections.forEach((s) => {
        s.style.opacity = "0";
        s.style.transform = "translateY(10px)";
      });
      stopInView = inView(
        "section",
        (info: any) => {
          animate(
            info.target,
            // transform 字符串而非 y shorthand：直接走合成器，主线程繁忙时不掉帧
            { opacity: [0, 1], transform: ["translateY(10px)", "translateY(0px)"] },
            { duration: 0.45, easing: [0.2, 0.7, 0.2, 1] } as any
          );
        },
        { root: stage, margin: "0px 0px -10% 0px" }
      );
    }

    const headingEls = [...art.querySelectorAll<HTMLElement>("h2[id], h3[id]")];
    const brush = document.getElementById("brushHead");

    let scrollRaf = 0;
    const update = () => {
      const max = stage.scrollHeight - stage.clientHeight;
      const pct = max > 0 ? stage.scrollTop / max : 0;
      if (brush) {
        brush.style.setProperty("--progress", (pct * 100).toFixed(2) + "%");
        brush.classList.toggle("is-reading", stage.scrollTop > 40);
      }
      // 用 getBoundingClientRect 相对 stage 定位标题，避免 offsetParent 链干扰
      const stageTop = stage.getBoundingClientRect().top;
      const probe = stage.clientHeight * 0.3;
      let activeId: string | null = null;
      for (const el of headingEls) {
        if (el.getBoundingClientRect().top - stageTop <= probe) activeId = el.id;
      }
      cbRef.current({ scrolled: stage.scrollTop > 10, activeId });
    };
    const onScroll = () => {
      if (scrollRaf) return;
      scrollRaf = requestAnimationFrame(() => {
        scrollRaf = 0;
        update();
      });
    };

    stage.addEventListener("scroll", onScroll, { passive: true });
    // 阅读模式回到顶部；预览模式保留原滚动位置
    if (preview) stage.scrollTop = prevTop;
    else stage.scrollTo({ top: 0 });
    update();

    return () => {
      stopInView?.();
      stage.removeEventListener("scroll", onScroll);
      if (scrollRaf) cancelAnimationFrame(scrollRaf);
    };
  }, [html, mode]);

  return (
    <main
      className="stage"
      ref={setStageRef}
      onContextMenu={(e) => {
        if (!buildContextMenu) return;
        e.preventDefault();
        open(e.clientX, e.clientY, buildContextMenu());
      }}
    >
      <article className="reading">
        <div className="article" id="article" ref={articleRef} />
        <footer className="colophon" id="colophon">
          <span className="colophon-cn">卷终</span> &middot; finis
        </footer>
      </article>
    </main>
  );
}
