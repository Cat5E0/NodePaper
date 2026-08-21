// 墨笔进度条：左侧随阅读进度生长的墨笔。
// progress / reading 由 Stage 的滚动效应直接命令式写入 #brushHead（高频，绕开 React），
// 故此组件不接 props。
export function Brush() {
  return (
    <div className="brush" aria-hidden="true">
      <div className="ink" />
      <div className="head" id="brushHead" />
    </div>
  );
}
