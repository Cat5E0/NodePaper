// 目录列（次左）：当前文章标题大纲，随文章更新。作为常驻侧栏，可折叠。
import type { TocItem } from "../lib/markdown";
import { useContextMenu } from "./ContextMenu";

interface TocProps {
  collapsed: boolean;
  items: TocItem[];
  activeId: string | null;
}

function escapeHtml(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

export function Toc({ collapsed, items, activeId }: TocProps) {
  const { open } = useContextMenu();
  return (
    <aside
      className={"pane toc" + (collapsed ? " collapsed" : "")}
      aria-label="目录"
    >
      <div className="pane-head toc-head">
        <span className="t">
          <b>目</b>contents
        </span>
      </div>
      <nav className="toc-body">
        {items.length ? (
          items.map((t) => (
            <a
              key={t.id}
              href={`#${t.id}`}
              className={`lvl-${t.level - 1}` + (t.id === activeId ? " active" : "")}
              data-target={t.id}
              dangerouslySetInnerHTML={{ __html: escapeHtml(t.text) }}
              onContextMenu={(e) => {
                e.preventDefault();
                open(e.clientX, e.clientY, [
                  {
                    label: "跳转到此节",
                    onClick: () =>
                      document
                        .getElementById(t.id)
                        ?.scrollIntoView({ behavior: "smooth", block: "start" }),
                  },
                ]);
              }}
            />
          ))
        ) : (
          <p className="toc-empty">此卷尚无小节。</p>
        )}
      </nav>
    </aside>
  );
}
