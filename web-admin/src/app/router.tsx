import { createBrowserRouter } from 'react-router-dom';
import { RequireAuth } from '../features/auth/RequireAuth';
import { AppShell } from '../shared/ui/AppShell';
import { CategoriesPage } from '../pages/CategoriesPage';
import { ImportPage } from '../pages/ImportPage';
import { LoginPage } from '../pages/LoginPage';
import { SnapshotPage } from '../pages/SnapshotPage';
import { WordCreatePage } from '../pages/WordCreatePage';
import { WordEditPage } from '../pages/WordEditPage';
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
          { path: '/words', element: <WordsPage /> },
          { path: '/words/new', element: <WordCreatePage /> },
          { path: '/words/:id/edit', element: <WordEditPage /> },
          { path: '/categories', element: <CategoriesPage /> },
          { path: '/import', element: <ImportPage /> },
          { path: '/snapshot', element: <SnapshotPage /> },
        ],
      },
    ],
  },
]);
