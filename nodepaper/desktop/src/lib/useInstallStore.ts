// 全局下载状态 store：App 挂载即监听进度与完成事件，与诊断浮层生命周期解耦。
// 关闭浮层后下载仍在后端继续；重开浮层能从全局状态恢复进度条；
// 完成事件到达时若浮层开着则自动刷新诊断。
import { useEffect, useState } from "react";
import { listen } from "@tauri-apps/api/event";
import {
  TOOL_INSTALL_EVENT,
  TOOL_INSTALL_DONE_EVENT,
  type InstallState,
  type InstallDone,
  type InstallProgress,
} from "../lib/tools";

export function useInstallStore() {
  const [install, setInstall] = useState<InstallState | null>(null);

  useEffect(() => {
    const p1 = listen<InstallProgress>(TOOL_INSTALL_EVENT, (e) => {
      setInstall({
        key: e.payload.key,
        pct: e.payload.pct,
        status: "running",
      });
    });
    const p2 = listen<InstallDone>(TOOL_INSTALL_DONE_EVENT, (e) => {
      setInstall({
        key: e.payload.key,
        pct: 100,
        status: e.payload.ok ? "ok" : "err",
        error: e.payload.error ?? undefined,
      });
    });
    return () => {
      p1.then((u) => u());
      p2.then((u) => u());
    };
  }, []);

  return { install, setInstall };
}
