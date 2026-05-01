import React, { useCallback } from 'react';
import {
  Upload,
  RefreshCw,
  LayoutGrid,
  List,
  Search,
  Plus,
  CheckSquare,
  Square,
  Trash2,
  Package,
  X,
  FolderOpen
} from 'lucide-react';
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { motion, AnimatePresence } from "framer-motion";
import { cn } from '@/lib/utils';

// Debounce utility
function useDebounce<T extends (...args: any[]) => any>(
  callback: T,
  delay: number
): T {
  let timeoutId: ReturnType<typeof setTimeout>;
  return ((...args: any[]) => {
    clearTimeout(timeoutId);
    timeoutId = setTimeout(() => callback(...args), delay);
  }) as T;
}

interface ToolbarProps {
  viewMode: 'list' | 'grid';
  onViewToggle: () => void;
  onRefresh: () => void;
  onUpload: () => void;
  onNewFolder: () => void;
  search: string;
  onSearchChange: (value: string) => void;
  selectionMode: boolean;
  onSelectionModeToggle: () => void;
  selectedCount: number;
  onBulkDelete: () => void;
  onBulkZip: () => void;
  onClearSelection: () => void;
  onOpenSelected?: () => void;
}

export const Toolbar: React.FC<ToolbarProps> = ({
  viewMode, onViewToggle, onRefresh, onUpload, onNewFolder, search, onSearchChange,
  selectionMode, onSelectionModeToggle, selectedCount, onBulkDelete, onBulkZip, onClearSelection
}) => {
  // Debounced search handler
  const debouncedSearch = useCallback(
    useDebounce((value: string) => onSearchChange(value), 300),
    [onSearchChange]
  );

  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    debouncedSearch(e.target.value);
  };

  return (
    <TooltipProvider>
      <div className="flex items-center gap-3 px-6 py-2">
        {selectedCount > 0 ? (
          <motion.div
            initial={{ opacity: 0, x: -10 }}
            animate={{ opacity: 1, x: 0 }}
            className="flex items-center gap-3"
          >
            <span className="text-xs font-medium bg-foreground text-background px-2.5 py-1 rounded-full">
              {selectedCount}
            </span>
            <Button variant="ghost" size="sm" onClick={onClearSelection} className="h-7 text-xs">
              <X className="w-3 h-3 mr-1" /> Clear
            </Button>
            <div className="flex items-center gap-1">
              <Button variant="ghost" size="sm" onClick={onBulkZip} className="h-7 text-xs">
                <Package className="w-3.5 h-3.5 mr-1" /> Zip
              </Button>
              <Button variant="ghost" size="sm" onClick={onBulkDelete} className="h-7 text-xs text-destructive hover:text-destructive">
                <Trash2 className="w-3.5 h-3.5 mr-1" /> Delete
              </Button>
            </div>
          </motion.div>
        ) : (
          <div className="relative flex-1 max-w-sm">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground/60" />
            <Input
              placeholder="Search files..."
              defaultValue={search}
              onChange={handleSearchChange}
              className="pl-9 h-9 bg-secondary/50 border-0 focus-visible:ring-1 focus-visible:ring-foreground/20 rounded-lg"
            />
          </div>
        )}

        <div className="flex items-center gap-1 ml-auto">
          {selectedCount === 0 && (
            <>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant={selectionMode ? "default" : "ghost"}
                    size="icon"
                    onClick={onSelectionModeToggle}
                    className={cn("h-8 w-8 rounded-lg", selectionMode && "shadow-sm")}
                  >
                    {selectionMode ? <CheckSquare className="w-4 h-4" /> : <Square className="w-4 h-4" />}
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{selectionMode ? "Cancel" : "Select"}</TooltipContent>
              </Tooltip>

              <div className="w-px h-4 bg-border mx-1" />

              <Tooltip>
                <TooltipTrigger asChild>
                  <Button variant="ghost" size="icon" onClick={onUpload} className="h-8 w-8 rounded-lg">
                    <Upload className="w-4 h-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Upload</TooltipContent>
              </Tooltip>

              <Tooltip>
                <TooltipTrigger asChild>
                  <Button variant="ghost" size="icon" onClick={onNewFolder} className="h-8 w-8 rounded-lg">
                    <Plus className="w-4 h-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>New Folder</TooltipContent>
              </Tooltip>

              <Tooltip>
                <TooltipTrigger asChild>
                  <Button variant="ghost" size="icon" onClick={onRefresh} className="h-8 w-8 rounded-lg">
                    <RefreshCw className="w-4 h-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Refresh</TooltipContent>
              </Tooltip>
            </>
          )}

          <div className="w-px h-4 bg-border mx-1" />

          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon" onClick={onViewToggle} className="h-8 w-8 rounded-lg">
                {viewMode === 'list' ? <LayoutGrid className="w-4 h-4" /> : <List className="w-4 h-4" />}
              </Button>
            </TooltipTrigger>
            <TooltipContent>{viewMode === 'list' ? 'Grid' : 'List'}</TooltipContent>
          </Tooltip>
        </div>
      </div>
    </TooltipProvider>
  );
};
