import React from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { 
  X, 
  CheckCircle2, 
  AlertCircle, 
  Loader2, 
  FileUp,
  ChevronDown,
  ChevronUp
} from 'lucide-react';
import { Button } from './ui/button';
import { cn } from '@/lib/utils';

export interface UploadStatus {
  id: string;
  name: string;
  progress: number;
  status: 'uploading' | 'completed' | 'error';
  error?: string;
}

interface UploadProgressProps {
  uploads: UploadStatus[];
  onClear: (id: string) => void;
  onClearAll: () => void;
}

export const UploadProgress: React.FC<UploadProgressProps> = ({ uploads, onClear, onClearAll }) => {
  const [isExpanded, setIsExpanded] = React.useState(true);
  
  const activeCount = uploads.filter(u => u.status === 'uploading').length;
  const completedCount = uploads.filter(u => u.status === 'completed').length;
  const errorCount = uploads.filter(u => u.status === 'error').length;

  if (uploads.length === 0) return null;

  return (
    <motion.div
      initial={{ y: 100, opacity: 0 }}
      animate={{ y: 0, opacity: 1 }}
      exit={{ y: 100, opacity: 0 }}
      className="fixed bottom-6 right-6 z-50 w-80 max-w-[calc(100vw-3rem)]"
    >
      <div className="bg-card border border-border/50 shadow-2xl rounded-2xl overflow-hidden flex flex-col">
        {/* Header */}
        <div 
          className="bg-secondary/30 px-4 py-3 flex items-center justify-between cursor-pointer select-none"
          onClick={() => setIsExpanded(!isExpanded)}
        >
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-primary/10 flex items-center justify-center text-primary">
              <FileUp className="w-4.5 h-4.5" />
            </div>
            <div>
              <h3 className="text-xs font-bold uppercase tracking-wider">
                {activeCount > 0 ? `Uploading ${activeCount} files...` : 'Uploads finished'}
              </h3>
              <p className="text-[10px] text-muted-foreground font-medium">
                {completedCount} completed • {errorCount} errors
              </p>
            </div>
          </div>
          <div className="flex items-center gap-1">
            <Button 
              variant="ghost" 
              size="icon" 
              className="h-7 w-7 rounded-md"
              onClick={(e) => {
                e.stopPropagation();
                onClearAll();
              }}
            >
              <X className="w-4 h-4" />
            </Button>
            <Button variant="ghost" size="icon" className="h-7 w-7 rounded-md">
              {isExpanded ? <ChevronDown className="w-4 h-4" /> : <ChevronUp className="w-4 h-4" />}
            </Button>
          </div>
        </div>

        {/* Body */}
        <AnimatePresence>
          {isExpanded && (
            <motion.div
              initial={{ height: 0 }}
              animate={{ height: 'auto' }}
              exit={{ height: 0 }}
              className="overflow-hidden"
            >
              <div className="max-h-80 overflow-y-auto no-scrollbar p-3 space-y-2 bg-card">
                {uploads.map((upload) => (
                  <div 
                    key={upload.id} 
                    className="p-3 rounded-xl bg-secondary/20 border border-border/40 space-y-2 group relative"
                  >
                    <div className="flex items-center justify-between gap-3">
                      <div className="flex items-center gap-2 min-w-0">
                        {upload.status === 'uploading' && (
                          <Loader2 className="w-3.5 h-3.5 animate-spin text-primary shrink-0" />
                        )}
                        {upload.status === 'completed' && (
                          <CheckCircle2 className="w-3.5 h-3.5 text-success shrink-0" />
                        )}
                        {upload.status === 'error' && (
                          <AlertCircle className="w-3.5 h-3.5 text-destructive shrink-0" />
                        )}
                        <span className="text-xs font-semibold truncate">
                          {upload.name}
                        </span>
                      </div>
                      <span className="text-[10px] font-bold text-muted-foreground">
                        {upload.progress}%
                      </span>
                    </div>

                    <div className="h-1.5 w-full bg-border/50 rounded-full overflow-hidden">
                      <motion.div
                        initial={{ width: 0 }}
                        animate={{ width: `${upload.progress}%` }}
                        className={cn(
                          "h-full rounded-full transition-all",
                          upload.status === 'uploading' && "bg-primary",
                          upload.status === 'completed' && "bg-success",
                          upload.status === 'error' && "bg-destructive"
                        )}
                      />
                    </div>

                    {upload.error && (
                      <p className="text-[10px] text-destructive font-medium leading-tight">
                        {upload.error}
                      </p>
                    )}

                    <button 
                      onClick={() => onClear(upload.id)}
                      className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-background rounded-md"
                    >
                      <X className="w-3 h-3 text-muted-foreground" />
                    </button>
                  </div>
                ))}
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </motion.div>
  );
};
