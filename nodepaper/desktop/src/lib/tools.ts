// 环境诊断与工具自取：类型与 invoke 包装。
// Rust 侧见 src-tauri/src/tools.rs（下载走 reqwest 流式，进度事件 np:tool-install）。
import { invoke } from "@tauri-apps/api/core";

export interface ToolStatus {
  key: string;
  label: string;
  found: boolean;
  path: string | null;
  version: string | null;
  downloadable: boolean;
  /** MB；仅缺失且可下载时有值 */
  sizeMb: number | null;
  hint: string | null;
}

export interface InstallProgress {
  key: string;
  received: number;
  total: number;
  pct: number;
}

export const TOOL_INSTALL_EVENT = "np:tool-install";
export const TOOL_INSTALL_DONE_EVENT = "np:tool-install-done";

/** 全局下载状态（App 层持有，浮层关闭不丢失） */
export interface InstallState {
  key: string;
  pct: number;
  /** running 期间后端互斥，重复触发会被拒绝 */
  status: "running" | "ok" | "err";
  error?: string;
}

export interface InstallDone {
  key: string;
  ok: boolean;
  error: string | null;
  path: string | null;
}

export function diagnoseTools(): Promise<ToolStatus[]> {
  return invoke<ToolStatus[]>("diagnose_tools");
}

/** 下载并安装（确认弹窗由调用方完成）；resolve 返回安装路径。
 * 后端同 key 互斥；任务在后端持续到落盘，关闭浮层不影响。 */
export function installTool(key: string): Promise<string> {
  return invoke<string>("install_tool", { key });
}
