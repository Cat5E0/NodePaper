// 文件系统：dialog 选目录 + invoke 自定义 command 列文件/读文本。
// 读文件走 Rust command（见 src-tauri/src/lib.rs），无需 plugin-fs 的运行时 scope 授权。
import { invoke } from "@tauri-apps/api/core";
import { open } from "@tauri-apps/plugin-dialog";

export interface FileEntry {
  /** 绝对路径，用于读取 */
  path: string;
  /** 相对根目录的路径（/ 分隔），用于书架展示与排序 */
  rel: string;
  /** 文件名 */
  name: string;
}

/** 弹出目录选择器，返回选中目录绝对路径或 null */
export async function pickDirectory(): Promise<string | null> {
  const selected = await open({ directory: true, multiple: false });
  return typeof selected === "string" ? selected : null;
}

/** 递归列出目录下所有 .md / .markdown 文件 */
export async function listMarkdown(dir: string): Promise<FileEntry[]> {
  return invoke<FileEntry[]>("list_markdown", { dir });
}

/** 读取文本文件内容 */
export async function readMarkdown(path: string): Promise<string> {
  return invoke<string>("read_markdown", { path });
}

export const MD_RE = /\.(md|markdown)$/i;
