import React from 'react';
import { 
  Upload, 
  RefreshCw, 
  LayoutGrid, 
  List, 
  Search,
  PlusCircle,
  CheckSquare,
  Square,
  Trash2,
  Package,
  X
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
}

export const Toolbar: React.FC<ToolbarProps> = ({ 
  viewMode, onViewToggle, onRefresh, onUpload, onNewFolder, search, onSearchChange,
  selectionMode, onSelectionModeToggle, selectedCount, onBulkDelete, onBulkZip, onClearSelection
}) => {
  return (
    <TooltipProvider>
      <div className="flex flex-col border-b bg-card/50 backdrop-blur-md sticky top-0 z-30">
        <div className="flex items-center justify-between gap-4 py-3 px-6">
          <div className="relative flex-1 max-w-md">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <Input 
              placeholder="Search files..." 
              value={search}
              onChange={(e) => onSearchChange(e.target.value)}
              className="pl-10 bg-muted/50 border-none h-9 focus-visible:ring-1 focus-visible:ring-primary/30 rounded-full"
            />
          </div>

          <div className="flex items-center gap-1">
            <Tooltip>
              <TooltipTrigger asChild>
                <Button 
                  variant={selectionMode ? "default" : "ghost"} 
                  size="icon" 
                  onClick={onSelectionModeToggle}
                  className={cn("h-9 w-9 rounded-full", selectionMode && "bg-primary shadow-lg shadow-primary/20")}
                >
                  {selectionMode ? <CheckSquare className="w-4 h-4" /> : <Square className="w-4 h-4" />}
                </Button>
              </TooltipTrigger>
              <TooltipContent>{selectionMode ? "Disable Selection" : "Enable Selection"}</TooltipContent>
            </Tooltip>

            <div className="w-px h-4 bg-border mx-1" />

            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon" onClick={onUpload} className="h-9 w-9 rounded-full">
                  <Upload className="w-4 h-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Upload File</TooltipContent>
            </Tooltip>

            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon" onClick={onNewFolder} className="h-9 w-9 rounded-full">
                  <PlusCircle className="w-4 h-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>New Folder</TooltipContent>
            </Tooltip>

            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon" onClick={onRefresh} className="h-9 w-9 rounded-full">
                  <RefreshCw className="w-4 h-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Refresh</TooltipContent>
            </Tooltip>

            <div className="w-px h-4 bg-border mx-1" />

            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon" onClick={onViewToggle} className="h-9 w-9 rounded-full">
                  {viewMode === 'list' ? <LayoutGrid className="w-4 h-4" /> : <List className="w-4 h-4" />}
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                Switch to {viewMode === 'list' ? 'Grid' : 'List'} View
              </TooltipContent>
            </Tooltip>
          </div>
        </div>

        <AnimatePresence>
          {selectedCount > 0 && (
            <motion.div 
              initial={{ height: 0, opacity: 0 }}
              animate={{ height: 'auto', opacity: 1 }}
              exit={{ height: 0, opacity: 0 }}
              className="overflow-hidden bg-primary/5 border-t border-primary/10"
            >
              <div className="flex items-center justify-between px-6 py-2">
                <div className="flex items-center gap-3">
                  <span className="text-xs font-bold bg-primary text-primary-foreground px-2 py-0.5 rounded-full">
                    {selectedCount}
                  </span>
                  <span className="text-xs font-medium text-primary">items selected</span>
                  <Button 
                    variant="ghost" 
                    size="sm" 
                    onClick={onClearSelection}
                    className="h-7 text-xs hover:bg-primary/10 text-primary"
                  >
                    <X className="w-3 h-3 mr-1" /> Clear
                  </Button>
                </div>

                <div className="flex items-center gap-2">
                  <Button 
                    variant="ghost" 
                    size="sm" 
                    onClick={onBulkZip}
                    className="h-8 text-xs font-semibold hover:bg-primary/10"
                  >
                    <Package className="w-3.5 h-3.5 mr-1.5" /> Archive
                  </Button>
                  <Button 
                    variant="ghost" 
                    size="sm" 
                    onClick={onBulkDelete}
                    className="h-8 text-xs font-semibold text-destructive hover:bg-destructive/10 hover:text-destructive"
                  >
                    <Trash2 className="w-3.5 h-3.5 mr-1.5" /> Delete
                  </Button>
                </div>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </TooltipProvider>
  );
};
