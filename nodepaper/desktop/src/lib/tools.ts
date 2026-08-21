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

export function diagnoseTools(): Promise<ToolStatus[]> {
  return invoke<ToolStatus[]>("diagnose_tools");
}

/** 下载并安装（确认弹窗由调用方完成）；resolve 返回安装路径 */
export function installTool(key: string): Promise<string> {
  return invoke<string>("install_tool", { key });
}
