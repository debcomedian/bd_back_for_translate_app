import { LoginForm } from "../features/auth/LoginForm";

export function LoginPage() {
  return (
    <div className="login-shell">
      <div className="card login-card stack">
        <div>
          <h1 className="page-title">Админ-панель Rugen</h1>
          <p className="page-subtitle">
            Вход администратора в контур управления направленной словарной
            моделью.
          </p>
        </div>
        <LoginForm />
      </div>
    </div>
  );
}
