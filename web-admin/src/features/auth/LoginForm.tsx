import { FormEvent, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from './AuthContext';
import { extractApiError } from '../../shared/api/client';

export function LoginForm() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [form, setForm] = useState({ login: '', password: '' });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await login(form.login, form.password);
      navigate('/words', { replace: true });
    } catch (e) {
      setError(extractApiError(e));
    } finally {
      setLoading(false);
    }
  }

  return (
    <form className="stack" onSubmit={handleSubmit}>
      <div>
        <label className="label" htmlFor="login">Логин</label>
        <input id="login" className="input" value={form.login} onChange={(e) => setForm((p) => ({ ...p, login: e.target.value }))} required />
      </div>
      <div>
        <label className="label" htmlFor="password">Пароль</label>
        <input id="password" type="password" className="input" value={form.password} onChange={(e) => setForm((p) => ({ ...p, password: e.target.value }))} required />
      </div>
      {error ? <div className="error">{error}</div> : null}
      <button className="btn btn-primary" disabled={loading} type="submit">
        {loading ? 'Вход...' : 'Войти'}
      </button>
    </form>
  );
}
