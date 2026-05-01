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
    <nav className="flex items-center space-x-1 text-sm text-muted-foreground font-medium overflow-x-auto no-scrollbar py-2">
      <button 
        onClick={() => onNavigate('/')}
        className="flex items-center hover:text-foreground transition-colors p-1 rounded-md hover:bg-accent"
      >
        <Home className="w-4 h-4" />
      </button>

      {parts.length > 0 && <ChevronRight className="w-4 h-4 text-muted-foreground/50 shrink-0" />}

      {parts.map((part, i) => {
        const isLast = i === parts.length - 1;
        const currentPath = '/' + parts.slice(0, i + 1).join('/');

        return (
          <React.Fragment key={i}>
            <button 
              onClick={() => onNavigate(currentPath)}
              disabled={isLast}
              className={cn(
                "flex items-center hover:text-foreground transition-colors p-1 px-2 rounded-md",
                isLast ? "text-foreground cursor-default font-semibold" : "hover:bg-accent"
              )}
            >
              {part}
            </button>
            {!isLast && <ChevronRight className="w-4 h-4 text-muted-foreground/50 shrink-0" />}
          </React.Fragment>
        );
      })}
      
      <span className="ml-2 w-2 h-2 rounded-full bg-primary animate-pulse shadow-[0_0_8px_rgba(99,102,241,0.5)]" />
    </nav>
  );
};
