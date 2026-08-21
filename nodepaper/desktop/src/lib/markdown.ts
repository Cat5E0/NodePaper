// Markdown 渲染：marked + highlight.js，自定义标题渲染以注入稳定 id / h2 编号 / 收集目录。
// 移植自原项目 src/render.js，行为保持一致。
import { marked } from "marked";
import { markedHighlight } from "marked-highlight";
import hljs from "highlight.js/lib/common";

marked.use(
  markedHighlight({
    langPrefix: "hljs language-",
    highlight(code, lang) {
      // 已知语言按语言高亮，未知语言安全降级为纯文本（仍做 HTML 转义）
      const language = lang && hljs.getLanguage(lang) ? lang : "plaintext";
      return hljs.highlight(code, { language }).value;
    },
  })
);

marked.setOptions({ gfm: true, breaks: false });

export interface TocItem {
  level: number;
  text: string;
  id: string;
}

/* 自定义标题渲染：为 h2/h3 注入稳定 id、h2 编号，并收集目录项 */
let h2Count = 0;
const toc: TocItem[] = [];

function stripText(html: string): string {
  const tmp = document.createElement("div");
  tmp.innerHTML = html;
  return tmp.textContent?.trim() ?? "";
}

const renderer = {
  heading(this: any, token: any): string {
    const text: string = this.parser.parseInline(token.tokens ?? []);
    const depth: number = token.depth;
    if (depth === 2 || depth === 3) {
      const id = `sec-${toc.length}`;
      toc.push({ level: depth, text: stripText(text), id });
      if (depth === 2) {
        h2Count += 1;
        return `<h2 data-num="${String(h2Count).padStart(2, "0")}" id="${id}">${text}</h2>\n`;
      }
      return `<h3 id="${id}">${text}</h3>\n`;
    }
    return `<h${depth}>${text}</h${depth}>\n`;
  },
};
marked.use({ renderer });

/** 渲染 Markdown，返回 { html, toc } */
export function renderMarkdown(md: string): { html: string; toc: TocItem[] } {
  h2Count = 0;
  toc.length = 0;
  const html = marked.parse(md) as string;
  return { html, toc: [...toc] };
}

/**
 * 按 h2 将顶层节点切分为 <section>，服务于逐节浮起与首字下沉。
 * 原地修改 root，与原项目 groupIntoSections 行为一致。
 */
export function groupIntoSections(root: HTMLElement): void {
  const nodes = [...root.childNodes];
  let current = document.createElement("section");
  root.append(current);
  for (const node of nodes) {
    if (node.nodeType === 1 && (node as HTMLElement).tagName === "H2") {
      current = document.createElement("section");
      root.append(current);
    }
    current.append(node);
  }
}
