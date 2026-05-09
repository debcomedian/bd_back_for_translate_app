import { createContext, useContext, useMemo, useState } from 'react';
import type { PropsWithChildren } from 'react';
import { adminLogin } from '../../shared/api/adminAuth';
import { clearAdminToken, getAdminToken, setAdminToken } from '../../shared/lib/storage';

interface AuthContextValue {
  isAuthenticated: boolean;
  login: (login: string, password: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: PropsWithChildren) {
  const [token, setTokenState] = useState<string | null>(() => getAdminToken());

  const value = useMemo<AuthContextValue>(() => ({
    isAuthenticated: Boolean(token),
    async login(loginValue: string, password: string) {
      const response = await adminLogin({ login: loginValue, password });
      setAdminToken(response.token);
      setTokenState(response.token);
    },
    logout() {
      clearAdminToken();
      setTokenState(null);
    },
  }), [token]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return context;
}
