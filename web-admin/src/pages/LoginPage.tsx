import { LoginForm } from '../features/auth/LoginForm';

export function LoginPage() {
  return (
    <div className="login-shell">
      <div className="card login-card stack">
        <div>
          <h1 className="page-title">Rugen Admin V1</h1>
          <p className="page-subtitle">Вход единственного администратора в контур управления словарным контентом.</p>
        </div>
        <LoginForm />
      </div>
    </div>
  );
}
