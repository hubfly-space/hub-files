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
    <nav className="flex items-center text-sm overflow-x-auto no-scrollbar bg-secondary/30 p-1 rounded-xl border border-border/40 max-w-full">
      <button
        onClick={() => onNavigate('/')}
        className="flex items-center gap-2 px-2.5 py-1.5 hover:bg-background rounded-lg transition-all hover:text-primary hover:shadow-sm"
      >
        <Home className="w-4 h-4" />
        {parts.length === 0 && <span className="font-bold text-xs uppercase tracking-widest">Root</span>}
      </button>

      {parts.length > 0 && <ChevronRight className="w-3.5 h-3.5 text-muted-foreground/30 shrink-0 mx-0.5" />}

      <div className="flex items-center gap-0.5">
        {parts.map((part, i) => {
          const isLast = i === parts.length - 1;
          const currentPath = '/' + parts.slice(0, i + 1).join('/');

          return (
            <React.Fragment key={i}>
              <button
                onClick={() => onNavigate(currentPath)}
                disabled={isLast}
                className={cn(
                  "px-2.5 py-1.5 rounded-lg transition-all truncate shrink-0 text-xs font-semibold",
                  isLast 
                    ? "text-primary bg-background shadow-sm cursor-default" 
                    : "hover:bg-background hover:text-primary text-muted-foreground"
                )}
              >
                {part}
              </button>
              {!isLast && <ChevronRight className="w-3.5 h-3.5 text-muted-foreground/30 shrink-0 mx-0.5" />}
            </React.Fragment>
          );
        })}
      </div>
    </nav>
  );
};
