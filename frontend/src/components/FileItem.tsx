import React, { useRef } from 'react';
import type { FileInfo } from '../api';
import { FileIcon } from './FileIcon';
import {
  MoreHorizontal,
  Pencil,
  Trash2,
  Box,
  ExternalLink,
  CheckCircle2,
  Circle,
  Calendar,
  HardDrive
} from 'lucide-react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { cn } from '@/lib/utils';
import { motion } from 'framer-motion';

interface FileItemProps {
  file: FileInfo;
  viewMode: 'list' | 'grid';
  isSelected: boolean;
  selectionMode: boolean;
  onNavigate: (name: string) => void;
  onSelect: (name: string, multi: boolean) => void;
  onDelete: (name: string) => void;
  onRename: (name: string) => void;
  onZip: (name: string) => void;
  onExtract: (name: string) => void;
}

export const FileItem: React.FC<FileItemProps> = ({
  file, viewMode, isSelected, selectionMode,
  onNavigate, onSelect, onDelete, onRename, onZip, onExtract
}) => {
  const clickTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isDoubleClick = useRef(false);

  const formatSize = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
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

  if (viewMode === 'grid') {
    return (
      <motion.div
        layout
        initial={{ opacity: 0, scale: 0.9 }}
        animate={{ opacity: 1, scale: 1 }}
        whileHover={{ y: -4 }}
        onClick={handleClick}
        onDoubleClick={handleDoubleClick}
        className={cn(
          "group relative flex flex-col items-center gap-3 p-5 rounded-2xl border transition-all cursor-pointer select-none",
          isSelected
            ? "bg-primary/5 border-primary/30 shadow-md ring-1 ring-primary/20"
            : "bg-card border-border/50 hover:border-primary/20 hover:shadow-xl hover:shadow-primary/5"
        )}
      >
        <div className={cn(
          "w-14 h-14 rounded-2xl transition-all flex items-center justify-center relative",
          isSelected ? "bg-primary/10" : "bg-secondary/50 group-hover:bg-primary/5"
        )}>
          <FileIcon isDir={file.isDir} name={file.name} className={cn(
            "w-8 h-8 transition-transform group-hover:scale-110 duration-300",
            isSelected ? "text-primary" : "text-foreground/70"
          )} />

          {(selectionMode || isSelected) && (
            <div className="absolute -top-1.5 -right-1.5">
              {isSelected ? (
                <div className="bg-primary rounded-full p-0.5 shadow-lg shadow-primary/30">
                  <CheckCircle2 className="w-5 h-5 text-primary-foreground fill-primary" />
                </div>
              ) : (
                <div className="bg-background rounded-full p-0.5 border-2 border-muted-foreground/20">
                  <Circle className="w-4 h-4 text-transparent" />
                </div>
              )}
            </div>
          )}
        </div>

        <div className="flex flex-col items-center gap-0.5 w-full">
          <span className="text-sm font-bold truncate w-full text-center px-1">
            {file.name}
          </span>
          {!file.isDir && (
            <span className="text-[10px] font-medium text-muted-foreground uppercase tracking-tight">
              {formatSize(file.size)}
            </span>
          )}
        </div>

        <div className="absolute top-3 right-3 opacity-0 group-hover:opacity-100 transition-all scale-90 group-hover:scale-100" onClick={e => e.stopPropagation()}>
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
      className={cn(
        "group flex items-center justify-between p-3 rounded-xl transition-all cursor-pointer select-none border",
        isSelected
          ? "bg-primary/5 border-primary/20 shadow-sm"
          : "bg-transparent border-transparent hover:bg-secondary/40 hover:border-border/50"
      )}
    >
      <div className="flex items-center gap-4 flex-1 min-w-0">
        <div className="relative shrink-0 w-10 h-10 rounded-lg bg-secondary/30 flex items-center justify-center group-hover:bg-secondary/50 transition-colors">
          <FileIcon isDir={file.isDir} name={file.name} className={cn(
            "w-5 h-5 transition-colors",
            isSelected ? "text-primary" : "text-muted-foreground"
          )} />
          {(selectionMode || isSelected) && (
            <div className="absolute -top-1.5 -left-1.5">
              {isSelected ? (
                <CheckCircle2 className="w-4 h-4 text-primary fill-background" />
              ) : (
                <Circle className="w-4 h-4 text-muted-foreground/20 fill-background" />
              )}
            </div>
          )}
        </div>
        <div className="flex flex-col min-w-0">
          <span className="text-sm font-semibold truncate group-hover:text-primary transition-colors">{file.name}</span>
          <div className="flex items-center gap-3 text-[10px] text-muted-foreground font-medium uppercase tracking-wider">
            {!file.isDir && (
              <span className="flex items-center gap-1">
                <HardDrive className="w-3 h-3" />
                {formatSize(file.size)}
              </span>
            )}
            <span className="flex items-center gap-1">
              <Calendar className="w-3 h-3" />
              {file.modTime}
            </span>
          </div>
        </div>
      </div>

      <div className="flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity" onClick={e => e.stopPropagation()}>
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
  file, onDelete, onRename, onZip, onExtract
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
        {file.name.endsWith('.zip') && (
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
