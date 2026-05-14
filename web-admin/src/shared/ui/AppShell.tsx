import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { useAuth } from "../../features/auth/AuthContext";

export function AppShell() {
  const { logout } = useAuth();
  const navigate = useNavigate();

  function handleLogout() {
    logout();
    navigate("/login", { replace: true });
  }

  return (
    <div className="layout">
      <aside className="sidebar">
        <div className="brand">Админ-панель Rugen</div>
        <nav className="nav">
          <NavLink to="/directions">Направления</NavLink>
          <NavLink to="/categories">Категории</NavLink>
          <NavLink to="/import">Импорт и пересчёт</NavLink>
          <NavLink to="/bank-quality">Качество банка</NavLink>
          <NavLink to="/snapshot">Снимок контента</NavLink>
        </nav>
        <div className="sidebar-footer">
          <button className="btn btn-ghost" onClick={handleLogout}>
            Выйти
          </button>
        </div>
      </aside>
      <main className="content">
        <Outlet />
      </main>
    </div>
  );
}
