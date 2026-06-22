import React, { useRef } from "react";
import type { FileInfo } from "../api";
import { FileIcon } from "./FileIcon";
import {
  MoreHorizontal,
  Pencil,
  Trash2,
  Box,
  ExternalLink,
  CheckCircle2,
  Circle,
  Calendar,
  HardDrive,
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { motion } from "framer-motion";

interface FileItemProps {
  file: FileInfo;
  viewMode: "list" | "grid";
  isSelected: boolean;
  selectionMode: boolean;
  onNavigate: (name: string) => void;
  onSelect: (name: string, multi: boolean) => void;
  onDelete: (name: string) => void;
  onRename: (name: string) => void;
  onZip: (name: string) => void;
  onExtract: (name: string) => void;
  onMove?: (sourceName: string, targetFolderName: string) => void;
}

export const FileItem: React.FC<FileItemProps> = ({
  file,
  viewMode,
  isSelected,
  selectionMode,
  onNavigate,
  onSelect,
  onDelete,
  onRename,
  onZip,
  onExtract,
  onMove,
}) => {
  const clickTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isDoubleClick = useRef(false);
  const [isDragOver, setIsDragOver] = React.useState(false);

  const formatSize = (bytes: number) => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
  };

  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (isDoubleClick.current) {
      isDoubleClick.current = false;
      return;
    }

    if (clickTimeout.current) {
      clearTimeout(clickTimeout.current);
      clickTimeout.current = null;
    }

    clickTimeout.current = setTimeout(() => {
      clickTimeout.current = null;
      onSelect(file.name, e.ctrlKey || e.metaKey || selectionMode);
    }, 200);
  };

  const handleDoubleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    isDoubleClick.current = true;
    if (clickTimeout.current) {
      clearTimeout(clickTimeout.current);
      clickTimeout.current = null;
    }
    onNavigate(file.name);
  };

  const handleDragStart = (e: React.DragEvent) => {
    e.dataTransfer.setData(
      "application/json",
      JSON.stringify({ name: file.name }),
    );
    e.dataTransfer.effectAllowed = "move";
  };

  const handleDragOver = (e: React.DragEvent) => {
    if (file.isDir) {
      e.preventDefault();
      e.dataTransfer.dropEffect = "move";
      if (!isDragOver) setIsDragOver(true);
    }
  };

  const handleDragLeave = (e: React.DragEvent) => {
    if (file.isDir) {
      e.preventDefault();
      setIsDragOver(false);
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    if (file.isDir) {
      e.preventDefault();
      setIsDragOver(false);
      try {
        const data = JSON.parse(e.dataTransfer.getData("application/json"));
        if (data && data.name && data.name !== file.name && onMove) {
          onMove(data.name, file.name);
        }
      } catch {
        // ignore invalid drag data
      }
    }
  };

  if (viewMode === "grid") {
    return (
      <motion.div
        layout
        initial={{ opacity: 0, scale: 0.9 }}
        animate={{ opacity: 1, scale: 1 }}
        whileHover={{ y: -6, transition: { duration: 0.2 } }}
        onClick={handleClick}
        onDoubleClick={handleDoubleClick}
        draggable
        onDragStartCapture={handleDragStart}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        className={cn(
          "group relative flex flex-col items-center gap-4 p-6 rounded-[2rem] border transition-all cursor-pointer select-none",
          file.name.startsWith(".") && "opacity-60 hover:opacity-100",
          isDragOver && "ring-2 ring-primary bg-primary/10 shadow-lg scale-105",
          isSelected
            ? "bg-primary/5 border-primary/40 shadow-xl shadow-primary/5 ring-1 ring-primary/20"
            : "bg-card border-border/50 hover:border-primary/30 hover:shadow-2xl hover:shadow-primary/10",
        )}
      >
        <div
          className={cn(
            "w-20 h-20 rounded-[1.75rem] transition-all flex items-center justify-center relative",
            isSelected
              ? "bg-primary/10"
              : "bg-secondary/40 group-hover:bg-primary/5",
          )}
        >
          {/* Subtle dots pattern on icon background */}
          <div
            className="absolute inset-0 opacity-[0.1] pointer-events-none"
            style={{
              backgroundImage:
                "radial-gradient(circle, currentColor 1px, transparent 1px)",
              backgroundSize: "12px 12px",
            }}
          />

          <FileIcon
            isDir={file.isDir}
            name={file.name}
            className={cn(
              "w-10 h-10 transition-transform group-hover:scale-110 duration-500 ease-out z-10",
              isSelected ? "text-primary" : "text-foreground/80",
            )}
          />

          {(selectionMode || isSelected) && (
            <div className="absolute -top-2 -right-2 z-20">
              {isSelected ? (
                <div className="bg-primary rounded-xl p-1 shadow-lg shadow-primary/30 animate-in zoom-in duration-200">
                  <CheckCircle2 className="w-5 h-5 text-primary-foreground" />
                </div>
              ) : (
                <div className="bg-background rounded-xl p-1 border-2 border-muted-foreground/20">
                  <Circle className="w-5 h-5 text-transparent" />
                </div>
              )}
            </div>
          )}
        </div>

        <div className="flex flex-col items-center gap-1 w-full z-10">
          <span className="text-sm font-bold truncate w-full text-center px-2 group-hover:text-primary transition-colors">
            {file.name}
          </span>
          {!file.isDir && (
            <span className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest opacity-70">
              {formatSize(file.size)}
            </span>
          )}
        </div>

        <div
          className="absolute top-4 right-4 opacity-0 group-hover:opacity-100 transition-all scale-90 group-hover:scale-100 z-20"
          onClick={(e) => e.stopPropagation()}
        >
          <FileItemActions
            file={file}
            onDelete={onDelete}
            onRename={onRename}
            onZip={onZip}
            onExtract={onExtract}
          />
        </div>
      </motion.div>
    );
  }

  return (
    <motion.div
      layout
      initial={{ opacity: 0, x: -10 }}
      animate={{ opacity: 1, x: 0 }}
      onClick={handleClick}
      onDoubleClick={handleDoubleClick}
      draggable
      onDragStartCapture={handleDragStart}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
      className={cn(
        "group flex items-center justify-between p-4 rounded-2xl transition-all cursor-pointer select-none border",
        file.name.startsWith(".") && "opacity-60 hover:opacity-100",
        isDragOver &&
          "ring-2 ring-primary bg-primary/10 shadow-lg scale-[1.01]",
        isSelected
          ? "bg-primary/5 border-primary/30 shadow-md ring-1 ring-primary/10"
          : "bg-transparent border-transparent hover:bg-secondary/50 hover:border-border/50 hover:shadow-lg hover:shadow-black/5",
      )}
    >
      <div className="flex items-center gap-5 flex-1 min-w-0">
        <div className="relative shrink-0 w-12 h-12 rounded-xl bg-secondary/50 flex items-center justify-center group-hover:bg-primary/5 transition-colors overflow-hidden">
          {/* Subtle dots pattern on icon background */}
          <div
            className="absolute inset-0 opacity-[0.05] pointer-events-none"
            style={{
              backgroundImage:
                "radial-gradient(circle, currentColor 1px, transparent 1px)",
              backgroundSize: "10px 10px",
            }}
          />

          <FileIcon
            isDir={file.isDir}
            name={file.name}
            className={cn(
              "w-6 h-6 transition-transform group-hover:scale-110 z-10",
              isSelected ? "text-primary" : "text-muted-foreground",
            )}
          />
          {(selectionMode || isSelected) && (
            <div className="absolute -top-1.5 -left-1.5 z-20">
              {isSelected ? (
                <div className="bg-primary rounded-lg p-0.5 shadow-md">
                  <CheckCircle2 className="w-4 h-4 text-primary-foreground" />
                </div>
              ) : (
                <div className="bg-background rounded-lg p-0.5 border border-muted-foreground/20">
                  <Circle className="w-4 h-4 text-transparent" />
                </div>
              )}
            </div>
          )}
        </div>
        <div className="flex flex-col min-w-0">
          <span className="text-sm font-bold truncate group-hover:text-primary transition-colors">
            {file.name}
          </span>
          <div className="flex items-center gap-4 text-[10px] text-muted-foreground font-bold uppercase tracking-widest opacity-70">
            {!file.isDir && (
              <span className="flex items-center gap-1.5">
                <HardDrive className="w-3 h-3" />
                {formatSize(file.size)}
              </span>
            )}
            <span className="flex items-center gap-1.5">
              <Calendar className="w-3 h-3" />
              {file.modTime}
            </span>
          </div>
        </div>
      </div>

      <div
        className="flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity"
        onClick={(e) => e.stopPropagation()}
      >
        <FileItemActions
          file={file}
          onDelete={onDelete}
          onRename={onRename}
          onZip={onZip}
          onExtract={onExtract}
        />
      </div>
    </motion.div>
  );
};

interface ActionProps {
  file: FileInfo;
  onDelete: (name: string) => void;
  onRename: (name: string) => void;
  onZip: (name: string) => void;
  onExtract: (name: string) => void;
}

const FileItemActions: React.FC<ActionProps> = ({
  file,
  onDelete,
  onRename,
  onZip,
  onExtract,
}) => {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className="h-7 w-7 rounded-md">
          <MoreHorizontal className="w-4 h-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-44">
        <DropdownMenuItem onClick={() => onRename(file.name)}>
          <Pencil className="mr-2 w-4 h-4" />
          Rename
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => onZip(file.name)}>
          <Box className="mr-2 w-4 h-4" />
          Archive
        </DropdownMenuItem>
        {file.name.endsWith(".zip") && (
          <DropdownMenuItem onClick={() => onExtract(file.name)}>
            <ExternalLink className="mr-2 w-4 h-4" />
            Extract
          </DropdownMenuItem>
        )}
        <DropdownMenuSeparator />
        <DropdownMenuItem
          className="text-destructive focus:text-destructive"
          onClick={() => onDelete(file.name)}
        >
          <Trash2 className="mr-2 w-4 h-4" />
          Delete
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
