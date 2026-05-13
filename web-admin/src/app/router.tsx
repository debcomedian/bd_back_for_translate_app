import { createBrowserRouter } from 'react-router-dom';
import { RequireAuth } from '../features/auth/RequireAuth';
import { AppShell } from '../shared/ui/AppShell';
import { BankQualityPage } from '../pages/BankQualityPage';
import { CategoriesPage } from '../pages/CategoriesPage';
import { ImportPage } from '../pages/ImportPage';
import { LoginPage } from '../pages/LoginPage';
import { SnapshotPage } from '../pages/SnapshotPage';
import { WordsPage } from '../pages/WordsPage';

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  {
    path: '/',
    element: <RequireAuth />,
    children: [
      {
        element: <AppShell />,
        children: [
          { index: true, element: <WordsPage /> },
          { path: '/directions', element: <WordsPage /> },
          { path: '/words', element: <WordsPage /> },
          { path: '/categories', element: <CategoriesPage /> },
          { path: '/import', element: <ImportPage /> },
          { path: '/bank-quality', element: <BankQualityPage /> },
          { path: '/snapshot', element: <SnapshotPage /> },
        ],
      },
    ],
  },
]);
