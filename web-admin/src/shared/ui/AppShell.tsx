import { NavLink, Outlet, useNavigate } from 'react-router-dom';
import { useAuth } from '../../features/auth/AuthContext';

export function AppShell() {
  const { logout } = useAuth();
  const navigate = useNavigate();

  function handleLogout() {
    logout();
    navigate('/login', { replace: true });
  }

  return (
    <div className="layout">
      <aside className="sidebar">
        <div className="brand">Rugen Admin V1</div>
        <nav className="nav">
          <NavLink to="/words">Слова</NavLink>
          <NavLink to="/words/new">Создать слово</NavLink>
          <NavLink to="/categories">Категории</NavLink>
          <NavLink to="/import">Импорт CSV</NavLink>
          <NavLink to="/snapshot">Snapshot</NavLink>
        </nav>
        <div className="sidebar-footer">
          <button className="btn btn-ghost" onClick={handleLogout}>Выйти</button>
        </div>
      </aside>
      <main className="content">
        <Outlet />
      </main>
    </div>
  );
}
