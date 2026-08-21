import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import "./styles/tokens.css";
import "./styles/style.css";

// macOS Overlay 标题栏：交通灯浮在 masthead 上，CSS 靠此类让位并隐藏自绘控制钮。
// 桌面壳内 UA 可靠；浏览器预览退化为无类（自绘钮仍在）。
if (navigator.userAgent.includes("Mac")) {
  document.documentElement.classList.add("os-mac");
}

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
