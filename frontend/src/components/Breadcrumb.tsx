import React from 'react';
import { ChevronRight, Home } from 'lucide-react';
import { cn } from '@/lib/utils';

interface BreadcrumbProps {
  path: string;
  onNavigate: (path: string) => void;
}

export const Breadcrumb: React.FC<BreadcrumbProps> = ({ path, onNavigate }) => {
  const parts = path.split('/').filter(Boolean);

  return (
    <nav className="flex items-center gap-1 text-sm overflow-x-auto no-scrollbar">
      <button
        onClick={() => onNavigate('/')}
        className="flex items-center hover:bg-secondary p-1.5 rounded-md transition-colors shrink-0"
      >
        <Home className="w-4 h-4" />
      </button>

      {parts.length > 0 && <ChevronRight className="w-3.5 h-3.5 text-muted-foreground/40 shrink-0" />}

      {parts.map((part, i) => {
        const isLast = i === parts.length - 1;
        const currentPath = '/' + parts.slice(0, i + 1).join('/');

        return (
          <React.Fragment key={i}>
            <button
              onClick={() => onNavigate(currentPath)}
              disabled={isLast}
              className={cn(
                "px-2 py-1.5 rounded-md transition-colors truncate shrink-0",
                isLast ? "text-foreground font-medium cursor-default" : "hover:bg-secondary"
              )}
            >
              {part}
            </button>
            {!isLast && <ChevronRight className="w-3.5 h-3.5 text-muted-foreground/40 shrink-0" />}
          </React.Fragment>
        );
      })}
    </nav>
  );
};
