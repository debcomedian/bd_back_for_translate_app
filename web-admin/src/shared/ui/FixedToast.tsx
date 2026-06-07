export type ToastKind = "error" | "warning" | "success";

export type AppToast = {
  id: number;
  kind: ToastKind;
  message: string;
};

const TOAST_TITLES: Record<ToastKind, string> = {
  error: "Ошибка",
  warning: "Внимание",
  success: "Готово",
};

export function FixedToast({ toast }: { toast: AppToast | null }) {
  if (!toast) return null;

  return (
    <div className="fixed-toast-wrap" role="status" aria-live="assertive">
      <div key={toast.id} className={`fixed-toast fixed-toast-${toast.kind}`}>
        <div className="fixed-toast-title">{TOAST_TITLES[toast.kind]}</div>
        <div className="fixed-toast-message">{toast.message}</div>
        <div className="fixed-toast-timer" />
      </div>
    </div>
  );
}
