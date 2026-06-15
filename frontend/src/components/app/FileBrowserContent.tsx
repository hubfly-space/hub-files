import React from "react";

import { AnimatePresence, motion } from "framer-motion";
import { AlertCircle, Loader2, Search } from "lucide-react";

import { FileItem } from "../FileItem";
import { FileViewer } from "../FileViewer";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

type FileEntry = React.ComponentProps<typeof FileItem>["file"];
type ViewMode = React.ComponentProps<typeof FileItem>["viewMode"];

type OpenFile = {
  path: string;
  name: string;
} | null;

type FileBrowserContentProps = {
  openFile: OpenFile;
  files: FileEntry[];
  loading: boolean;
  error?: string | null;
  search: string;
  viewMode: ViewMode;
  selectedItems: Set<string>;
  selectionMode: boolean;
  onCloseFile: () => void;
  onRefresh: () => void;
  onNavigate: (name: string) => void;
  onSelect: (name: string, multi: boolean) => void;
  onDelete: (name: string) => void;
  onRename: (name: string) => void;
  onZip: (name: string) => void;
  onExtract: (name: string) => void;
  onMove: (sourceName: string, targetFolderName: string) => void;
};

export function FileBrowserContent({
  openFile,
  files,
  loading,
  error,
  search,
  viewMode,
  selectedItems,
  selectionMode,
  onCloseFile,
  onRefresh,
  onNavigate,
  onSelect,
  onDelete,
  onRename,
  onZip,
  onExtract,
  onMove,
}: FileBrowserContentProps) {
  if (openFile) {
    return (
      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        className="h-full p-6 max-w-[1600px] mx-auto w-full"
      >
        <FileViewer
          path={openFile.path}
          name={openFile.name}
          onClose={onCloseFile}
        />
      </motion.div>
    );
  }

  return (
    <div className="h-full overflow-y-auto no-scrollbar scroll-smooth">
      <div className="p-6 max-w-[1600px] mx-auto w-full min-h-full flex flex-col">
        <AnimatePresence mode="wait">
          {loading ? (
            <LoadingState />
          ) : error ? (
            <ErrorState error={error} onRefresh={onRefresh} />
          ) : files.length === 0 ? (
            <EmptyState search={search} />
          ) : (
            <motion.div
              key="content"
              layout
              className={cn(
                "flex-1",
                viewMode === "list" ? "space-y-1.5" : "file-grid-layout",
              )}
            >
              {files.map((file) => (
                <FileItem
                  key={file.name}
                  file={file}
                  viewMode={viewMode}
                  isSelected={selectedItems.has(file.name)}
                  selectionMode={selectionMode}
                  onNavigate={onNavigate}
                  onSelect={onSelect}
                  onDelete={onDelete}
                  onRename={onRename}
                  onZip={onZip}
                  onExtract={onExtract}
                  onMove={onMove}
                />
              ))}
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}

function LoadingState() {
  return (
    <motion.div
      key="loading"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      className="flex-1 flex flex-col items-center justify-center gap-4"
    >
      <div className="relative">
        <Loader2 className="w-10 h-10 animate-spin text-primary" />
        <div className="absolute inset-0 blur-xl bg-primary/20 animate-pulse" />
      </div>

      <p className="text-sm font-medium text-muted-foreground animate-pulse">
        Scanning filesystem...
      </p>
    </motion.div>
  );
}

function ErrorState({
  error,
  onRefresh,
}: {
  error: string;
  onRefresh: () => void;
}) {
  return (
    <motion.div
      key="error"
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      exit={{ opacity: 0 }}
      className="flex-1 flex flex-col items-center justify-center gap-6"
    >
      <div className="w-20 h-20 rounded-3xl bg-destructive/10 flex items-center justify-center shadow-inner">
        <AlertCircle className="w-10 h-10 text-destructive" />
      </div>

      <div className="text-center space-y-2">
        <h3 className="text-xl font-bold tracking-tight">Access Denied</h3>
        <p className="text-sm text-muted-foreground max-w-xs mx-auto leading-relaxed">
          {error}
        </p>
      </div>

      <Button variant="secondary" className="rounded-xl px-8" onClick={onRefresh}>
        Try Again
      </Button>
    </motion.div>
  );
}

function EmptyState({ search }: { search: string }) {
  return (
    <motion.div
      key="empty"
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      exit={{ opacity: 0 }}
      className="flex-1 flex flex-col items-center justify-center gap-6 opacity-60"
    >
      <div className="w-20 h-20 rounded-full bg-secondary/50 flex items-center justify-center">
        <Search className="w-10 h-10 text-muted-foreground/40" />
      </div>

      <div className="text-center space-y-2">
        <p className="text-lg font-bold text-muted-foreground">
          {search ? "No matches found" : "Empty Space"}
        </p>

        <p className="text-sm text-muted-foreground/60 max-w-[200px] mx-auto">
          {search
            ? "Try refining your search terms"
            : "Nothing here yet. Drag and drop to upload."}
        </p>
      </div>
    </motion.div>
  );
}
