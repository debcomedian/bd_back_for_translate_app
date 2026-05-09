import type { PropsWithChildren, ReactNode } from 'react';

export function PageHeader({ title, subtitle, actions }: PropsWithChildren<{ title: string; subtitle?: string; actions?: ReactNode }>) {
  return (
    <div className="page-header">
      <div>
        <h1 className="page-title">{title}</h1>
        {subtitle ? <p className="page-subtitle">{subtitle}</p> : null}
      </div>
      {actions ? <div className="actions">{actions}</div> : null}
    </div>
  );
}
