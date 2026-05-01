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
  Circle
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
      <div
        onClick={handleClick}
        onDoubleClick={handleDoubleClick}
        className={cn(
          "group relative flex flex-col items-center gap-3 p-4 rounded-xl border transition-all cursor-pointer select-none",
          isSelected
            ? "bg-secondary border-foreground/20"
            : "border-transparent hover:bg-secondary/50 hover:border-border"
        )}
      >
        <div className={cn(
          "p-3 rounded-lg transition-colors relative",
          isSelected ? "bg-background" : "bg-secondary/70 group-hover:bg-secondary"
        )}>
          <FileIcon isDir={file.isDir} name={file.name} className="w-10 h-10 text-foreground/80" />

          {(selectionMode || isSelected) && (
            <div className="absolute -top-1 -right-1">
              {isSelected ? (
                <CheckCircle2 className="w-5 h-5 text-foreground fill-background" />
              ) : (
                <Circle className="w-5 h-5 text-muted-foreground/40 fill-background" />
              )}
            </div>
          )}
        </div>

        <span className="text-sm font-medium text-center truncate w-full">
          {file.name}
        </span>

        <div className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity" onClick={e => e.stopPropagation()}>
          <FileItemActions
            file={file}
            onDelete={onDelete}
            onRename={onRename}
            onZip={onZip}
            onExtract={onExtract}
          />
        </div>
      </div>
    );
  }

  return (
    <div
      onClick={handleClick}
      onDoubleClick={handleDoubleClick}
      className={cn(
        "group flex items-center justify-between p-2 px-3 rounded-lg transition-all cursor-pointer select-none",
        isSelected
          ? "bg-secondary"
          : "hover:bg-secondary/50"
      )}
    >
      <div className="flex items-center gap-3 flex-1 min-w-0">
        <div className="relative shrink-0">
          <FileIcon isDir={file.isDir} name={file.name} className="w-5 h-5 text-muted-foreground" />
          {selectionMode && (
            <div className="absolute -top-1.5 -left-1.5">
              {isSelected ? (
                <CheckCircle2 className="w-3.5 h-3.5 text-foreground fill-background" />
              ) : (
                <Circle className="w-3.5 h-3.5 text-muted-foreground/30 fill-background" />
              )}
            </div>
          )}
        </div>
        <span className="text-sm truncate">{file.name}</span>
      </div>

      <div className="flex items-center gap-4 text-xs text-muted-foreground shrink-0">
        {!file.isDir && <span>{formatSize(file.size)}</span>}
        <span className="hidden md:inline">{file.modTime}</span>

        <div className="opacity-0 group-hover:opacity-100 transition-opacity" onClick={e => e.stopPropagation()}>
          <FileItemActions
            file={file}
            onDelete={onDelete}
            onRename={onRename}
            onZip={onZip}
            onExtract={onExtract}
          />
        </div>
      </div>
    </div>
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
