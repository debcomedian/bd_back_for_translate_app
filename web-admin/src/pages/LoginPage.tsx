import { LoginForm } from "../features/auth/LoginForm";

export function LoginPage() {
  return (
    <div className="login-shell">
      <div className="card login-card stack">
        <div>
          <h1 className="page-title">Панель администратора</h1>
          <p className="page-subtitle">
            Вход для управления словарным банком, импортом и публикацией учебных данных.
          </p>
        </div>
        <LoginForm />
      </div>
    </div>
  );
}
