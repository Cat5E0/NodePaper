// 书架列（最左）：递归渲染目录/文件树。作为常驻侧栏，可折叠。
import { useState } from "react";
import type { ShelfNode, ShelfDir } from "../lib/shelf";
import { stripExt } from "../lib/shelf";
import type { FileEntry } from "../lib/fileSystem";
import { useContextMenu } from "./ContextMenu";

interface ShelfProps {
  collapsed: boolean;
  title: string;
  tree: ShelfNode | null;
  activePath: string | null;
  onOpenFile: (entry: FileEntry) => void;
  onOpenFolder: () => void;
  onCreate: () => void;
  onDelete: (entry: FileEntry) => void;
}

interface NodeProps {
  node: ShelfNode;
  depth: number;
  activePath: string | null;
  onOpenFile: (entry: FileEntry) => void;
  onDelete: (entry: FileEntry) => void;
}

function NodeView({ node, depth, activePath, onOpenFile, onDelete }: NodeProps) {
  const { open } = useContextMenu();
  return (
    <ul className={depth === 0 ? "tree" : "tree tree-sub"}>
      {node.dirs.map((d) => (
        <DirEntry
          key={d.name}
          dir={d}
          depth={depth}
          activePath={activePath}
          onOpenFile={onOpenFile}
          onDelete={onDelete}
        />
      ))}
      {node.files.map((f) => (
        <li key={f.path} className="tree-file">
          <a
            href="#"
            data-path={f.path}
            className={f.path === activePath ? "active" : ""}
            onClick={(e) => {
              e.preventDefault();
              onOpenFile(f);
            }}
            onContextMenu={(e) => {
              e.preventDefault();
              open(e.clientX, e.clientY, [
                { label: "打开", onClick: () => onOpenFile(f) },
                { type: "sep" },
                { label: "删除…", onClick: () => onDelete(f) },
              ]);
            }}
          >
            {stripExt(f.name)}
          </a>
        </li>
      ))}
    </ul>
  );
}

function DirEntry({
  dir,
  depth,
  activePath,
  onOpenFile,
  onDelete,
}: {
  dir: ShelfDir;
  depth: number;
  activePath: string | null;
  onOpenFile: (entry: FileEntry) => void;
  onDelete: (entry: FileEntry) => void;
}) {
  const [open, setOpen] = useState(true);
  return (
    <li className={"tree-dir" + (open ? " open" : "")}>
      <button type="button" className="tree-fold" onClick={() => setOpen((o) => !o)}>
        <span className="caret" />
        <span className="tree-name">{dir.name}</span>
      </button>
      <NodeView
        node={dir.node}
        depth={depth + 1}
        activePath={activePath}
        onOpenFile={onOpenFile}
        onDelete={onDelete}
      />
    </li>
  );
}

export function Shelf({
  collapsed,
  title,
  tree,
  activePath,
  onOpenFile,
  onOpenFolder,
  onCreate,
  onDelete,
}: ShelfProps) {
  const { open } = useContextMenu();
  return (
    <aside
      className={"pane shelf" + (collapsed ? " collapsed" : "")}
      aria-label="书架"
    >
      <div className="pane-head shelf-head">
        <span className="t">
          <b>书</b>shelf
        </span>
      </div>
      <div className="shelf-title">{title}</div>
      <nav
        className="shelf-body"
        onContextMenu={(e) => {
          // 仅在空白处（非文件项）弹出
          if ((e.target as HTMLElement).closest(".tree-file")) return;
          e.preventDefault();
          open(e.clientX, e.clientY, [
            { label: "新建笔记", onClick: onCreate },
            { type: "sep" },
            { label: "打开文件夹…", onClick: onOpenFolder },
          ]);
        }}
      >
        {tree ? (
          <NodeView
            node={tree}
            depth={0}
            activePath={activePath}
            onOpenFile={onOpenFile}
            onDelete={onDelete}
          />
        ) : (
          <div className="shelf-empty">
            <p>尚未打开文件夹。</p>
            <button className="btn primary" onClick={onOpenFolder}>
              打开文件夹
            </button>
          </div>
        )}
      </nav>
    </aside>
  );
}
