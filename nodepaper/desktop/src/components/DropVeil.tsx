// 拖放遮罩
export function DropVeil({ open }: { open: boolean }) {
  return (
    <div className={"drop-veil" + (open ? " open" : "")} id="dropVeil">
      <div className="msg">松手，落纸成卷</div>
    </div>
  );
}
