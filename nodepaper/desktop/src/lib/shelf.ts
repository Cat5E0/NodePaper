// 由扁平 FileEntry 构建嵌套书架树。移植自原项目 renderShelf / buildTree 的数据逻辑，
// 渲染交给 React 组件。
import type { FileEntry } from "./fileSystem";

export interface ShelfDir {
  name: string;
  node: ShelfNode;
}
export interface ShelfNode {
  dirs: ShelfDir[];
  files: FileEntry[];
}

const zh = "zh-Hans";

function sortNode(n: ShelfNode): void {
  n.dirs.sort((a, b) => a.name.localeCompare(b.name, zh));
  n.files.sort((a, b) => a.name.localeCompare(b.name, zh));
  n.dirs.forEach((d) => sortNode(d.node));
}

export function buildShelfTree(items: FileEntry[]): ShelfNode {
  const root: ShelfNode = { dirs: [], files: [] };
  for (const it of items) {
    const parts = it.rel.split("/");
    let node = root;
    for (let i = 0; i < parts.length - 1; i++) {
      const d = parts[i];
      let next = node.dirs.find((x) => x.name === d);
      if (!next) {
        next = { name: d, node: { dirs: [], files: [] } };
        node.dirs.push(next);
      }
      node = next.node;
    }
    node.files.push(it);
  }
  sortNode(root);
  return root;
}

export function stripExt(name: string): string {
  return name.replace(/\.(md|markdown)$/i, "");
}
