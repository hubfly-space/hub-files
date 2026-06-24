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
  FilePlusCorner,
  HardDriveDownload,
  HardDriveUpload,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useEffect, useRef, useState } from "react";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { motion, AnimatePresence } from "framer-motion";
import { cn } from "@/lib/utils";

// Debounce utility
// function useDebounce<T extends (...args: any[]) => any>(
//   callback: T,
//   delay: number,
// ): T {
//   let timeoutId: ReturnType<typeof setTimeout>;
//   return ((...args: any[]) => {
//     clearTimeout(timeoutId);
//     timeoutId = setTimeout(() => callback(...args), delay);
//   }) as T;
// }

interface ToolbarProps {
  viewMode: "list" | "grid";
  onViewToggle: () => void;
  onRefresh: () => void;
  onUpload: () => void;
  onNewFolder: () => void;
  onNewFile: () => void;
  search: string;
  onSearchChange: (value: string) => void;
  selectionMode: boolean;
  onSelectionModeToggle: () => void;
  selectedCount: number;
  onBulkDelete: () => void;
  onBulkZip: () => void;
  onClearSelection: () => void;
  onOpenSelected?: () => void;
  canHostMount?: boolean;
  hostMounting?: boolean;
  onHostMount?: () => void;
  onHostUnmount?: () => void;
}

export const Toolbar: React.FC<ToolbarProps> = ({
  viewMode,
  onViewToggle,
  onRefresh,
  onUpload,
  onNewFolder,
  onNewFile,
  search,
  onSearchChange,
  selectionMode,
  onSelectionModeToggle,
  selectedCount,
  onBulkDelete,
  onBulkZip,
  onClearSelection,
  canHostMount = false,
  hostMounting = false,
  onHostMount,
  onHostUnmount,
}) => {
  const [localSearch, setLocalSearch] = useState(search);

  const searchTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const onSearchChangeRef = useRef(onSearchChange);

  useEffect(() => {
    onSearchChangeRef.current = onSearchChange;
  }, [onSearchChange]);

  const handleSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const value = event.target.value;

    setLocalSearch(value);

    if (searchTimeoutRef.current) {
      clearTimeout(searchTimeoutRef.current);
    }

    searchTimeoutRef.current = setTimeout(() => {
      onSearchChangeRef.current(value);
    }, 300);
  };

  useEffect(() => {
    return () => {
      if (searchTimeoutRef.current) {
        clearTimeout(searchTimeoutRef.current);
      }
    };
  }, []);
  // Debounced search handler
  // const debouncedSearch = useCallback(
  //   useDebounce((value: string) => onSearchChange(value), 300),
  //   [onSearchChange]
  // );

  return (
    <TooltipProvider>
      <div className="flex items-center gap-4 py-1">
        <div className="relative flex-1 max-w-md group">
          <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground group-focus-within:text-primary transition-colors" />
          <Input
            value={localSearch}
            onChange={handleSearchChange}
            placeholder="Search files..."
            className="pl-10 h-10 bg-secondary/30 border-transparent focus-visible:bg-background focus-visible:ring-2 focus-visible:ring-primary/20 focus-visible:border-primary/30 rounded-xl transition-all"
          />
        </div>

        <div className="flex items-center gap-1.5 ml-auto">
          {canHostMount && (
            <>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="secondary"
                    size="icon"
                    onClick={onHostMount}
                    disabled={hostMounting}
                    className="h-10 w-10 rounded-xl bg-secondary/50 hover:bg-primary hover:text-primary-foreground transition-all"
                  >
                    <HardDriveDownload
                      className={cn(
                        "w-4.5 h-4.5",
                        hostMounting && "animate-pulse",
                      )}
                    />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Mount on this machine</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="secondary"
                    size="icon"
                    onClick={onHostUnmount}
                    disabled={hostMounting}
                    className="h-10 w-10 rounded-xl bg-secondary/50 hover:bg-destructive hover:text-destructive-foreground transition-all"
                  >
                    <HardDriveUpload
                      className={cn(
                        "w-4.5 h-4.5",
                        hostMounting && "animate-pulse",
                      )}
                    />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Unmount from this machine</TooltipContent>
              </Tooltip>

              <div className="w-px h-6 bg-border/60 mx-1" />
            </>
          )}

          <AnimatePresence mode="wait">
            {selectedCount > 0 ? (
              <motion.div
                key="actions"
                initial={{ opacity: 0, x: 20 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: 20 }}
                className="flex items-center gap-1.5 bg-primary/5 p-1 rounded-xl border border-primary/10"
              >
                <div className="px-3 py-1 bg-primary rounded-lg text-primary-foreground text-[10px] font-bold uppercase tracking-widest shadow-lg shadow-primary/20">
                  {selectedCount} Selected
                </div>
                <div className="w-px h-4 bg-primary/20 mx-1" />
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={onBulkZip}
                      className="h-8 w-8 rounded-lg hover:bg-primary/10 hover:text-primary transition-colors"
                    >
                      <Package className="w-4 h-4" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Archive Selected</TooltipContent>
                </Tooltip>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={onBulkDelete}
                      className="h-8 w-8 rounded-lg hover:bg-destructive/10 hover:text-destructive transition-colors"
                    >
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Delete Selected</TooltipContent>
                </Tooltip>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={onClearSelection}
                  className="h-8 w-8 rounded-lg"
                >
                  <X className="w-4 h-4" />
                </Button>
              </motion.div>
            ) : (
              <motion.div
                key="tools"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                className="flex items-center gap-1.5"
              >
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant={selectionMode ? "default" : "secondary"}
                      size="icon"
                      onClick={onSelectionModeToggle}
                      className={cn(
                        "h-10 w-10 rounded-xl transition-all shadow-sm",
                        selectionMode ? "shadow-primary/20" : "bg-secondary/50",
                      )}
                    >
                      {selectionMode ? (
                        <CheckSquare className="w-4.5 h-4.5" />
                      ) : (
                        <Square className="w-4.5 h-4.5" />
                      )}
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    {selectionMode ? "Exit Selection" : "Enter Selection Mode"}
                  </TooltipContent>
                </Tooltip>

                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="secondary"
                      size="icon"
                      onClick={onUpload}
                      className="h-10 w-10 rounded-xl bg-secondary/50 hover:bg-primary hover:text-primary-foreground transition-all"
                    >
                      <Upload className="w-4.5 h-4.5" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Upload Files</TooltipContent>
                </Tooltip>

                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="secondary"
                      size="icon"
                      onClick={onNewFolder}
                      className="h-10 w-10 rounded-xl bg-secondary/50 transition-all"
                    >
                      <Plus className="w-4.5 h-4.5" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>New Folder</TooltipContent>
                </Tooltip>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="secondary"
                      size="icon"
                      onClick={onNewFile}
                      className="h-10 w-10 rounded-xl bg-secondary/50 transition-all"
                    >
                      <FilePlusCorner className="w-4.5 h-4.5" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>New File</TooltipContent>
                </Tooltip>

                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="secondary"
                      size="icon"
                      onClick={onRefresh}
                      className="h-10 w-10 rounded-xl bg-secondary/50 transition-all"
                    >
                      <RefreshCw className="w-4.5 h-4.5" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Refresh</TooltipContent>
                </Tooltip>
              </motion.div>
            )}
          </AnimatePresence>

          <div className="w-px h-6 bg-border/60 mx-1" />

          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="secondary"
                size="icon"
                onClick={onViewToggle}
                className="h-10 w-10 rounded-xl bg-secondary/50 transition-all"
              >
                {viewMode === "list" ? (
                  <LayoutGrid className="w-4.5 h-4.5" />
                ) : (
                  <List className="w-4.5 h-4.5" />
                )}
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              Switch to {viewMode === "list" ? "Grid" : "List"} View
            </TooltipContent>
          </Tooltip>
        </div>
      </div>
    </TooltipProvider>
  );
};
