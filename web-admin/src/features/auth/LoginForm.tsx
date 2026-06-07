import { FormEvent, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "./AuthContext";
import { extractApiError } from "../../shared/api/client";
import { FixedToast, type AppToast } from "../../shared/ui/FixedToast";

export function LoginForm() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [form, setForm] = useState({ login: "", password: "" });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [toast, setToast] = useState<AppToast | null>(null);

  useEffect(() => {
    if (!toast) return undefined;
    const timer = window.setTimeout(() => {
      setToast((current) => (current?.id === toast.id ? null : current));
    }, 3000);
    return () => window.clearTimeout(timer);
  }, [toast]);

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await login(form.login.trim(), form.password);
      navigate("/directions", { replace: true });
    } catch (e) {
      const message = extractApiError(e);
      setError(message);
      setToast({ id: Date.now(), kind: "error", message });
    } finally {
      setLoading(false);
    }
  }

  return (
    <form className="stack" onSubmit={handleSubmit}>
      <FixedToast toast={toast} />
      {error ? (
        <div className="visually-hidden" role="alert">
          {error}
        </div>
      ) : null}
      <div>
        <label className="label" htmlFor="login">
          Логин
        </label>
        <input
          id="login"
          className="input"
          autoComplete="username"
          value={form.login}
          onChange={(e) => setForm((p) => ({ ...p, login: e.target.value }))}
          required
        />
      </div>
      <div>
        <label className="label" htmlFor="password">
          Пароль
        </label>
        <input
          id="password"
          type="password"
          className="input"
          autoComplete="current-password"
          value={form.password}
          onChange={(e) => setForm((p) => ({ ...p, password: e.target.value }))}
          required
        />
      </div>
      <button className="btn btn-primary" disabled={loading} type="submit">
        {loading ? "Проверка..." : "Войти"}
      </button>
    </form>
  );
}
