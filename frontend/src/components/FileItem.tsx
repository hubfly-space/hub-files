import React from 'react';
import type { FileInfo } from '../api';
import { FileIcon } from './FileIcon';
import { 
  MoreVertical, 
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
import { motion } from "framer-motion";
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
  const formatSize = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  };

  const itemVariants = {
    initial: { opacity: 0, scale: 0.95 },
    animate: { opacity: 1, scale: 1 },
    exit: { opacity: 0, scale: 0.95 }
  };

  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    onSelect(file.name, e.ctrlKey || e.metaKey || selectionMode);
  };

  const handleDoubleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    onNavigate(file.name);
  };

  if (viewMode === 'grid') {
    return (
      <motion.div 
        variants={itemVariants}
        layout
        onClick={handleClick}
        onDoubleClick={handleDoubleClick}
        className={cn(
          "group relative flex flex-col items-center p-4 rounded-2xl border transition-all cursor-pointer",
          isSelected 
            ? "bg-primary/10 border-primary ring-1 ring-primary/20" 
            : "border-transparent hover:border-border hover:bg-accent/50"
        )}
      >
        <div className={cn(
          "mb-3 p-4 rounded-xl transition-colors relative",
          isSelected ? "bg-primary/20" : "bg-secondary/50 group-hover:bg-secondary"
        )}>
          <FileIcon isDir={file.isDir} name={file.name} className={cn(
            "w-12 h-12 transition-colors",
            isSelected ? "text-primary" : "text-primary/80"
          )} />
          
          {(selectionMode || isSelected) && (
            <div className="absolute -top-1 -right-1">
              {isSelected ? (
                <CheckCircle2 className="w-5 h-5 text-primary fill-background" />
              ) : (
                <Circle className="w-5 h-5 text-muted-foreground/50 fill-background" />
              )}
            </div>
          )}
        </div>
        
        <span className="text-sm font-medium text-center truncate w-full px-1">
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
      </motion.div>
    );
  }

  return (
    <motion.div 
      variants={itemVariants}
      layout
      onClick={handleClick}
      onDoubleClick={handleDoubleClick}
      className={cn(
        "group flex items-center justify-between p-2 px-4 rounded-xl transition-all cursor-pointer border",
        isSelected 
          ? "bg-primary/10 border-primary/30 ring-1 ring-primary/10" 
          : "border-transparent hover:bg-accent/50 hover:border-border"
      )}
    >
      <div className="flex items-center gap-4 flex-1 min-w-0">
        <div className="relative shrink-0">
          <FileIcon isDir={file.isDir} name={file.name} className={cn(
            "w-5 h-5 transition-colors",
            isSelected ? "text-primary" : "text-muted-foreground group-hover:text-primary"
          )} />
          {selectionMode && (
            <div className="absolute -top-2 -left-2">
              {isSelected ? (
                <CheckCircle2 className="w-3.5 h-3.5 text-primary fill-background" />
              ) : (
                <Circle className="w-3.5 h-3.5 text-muted-foreground/30 fill-background" />
              )}
            </div>
          )}
        </div>
        <span className={cn(
          "text-sm font-medium truncate",
          isSelected ? "text-primary" : ""
        )}>{file.name}</span>
      </div>
      
      <div className="flex items-center gap-6 text-xs text-muted-foreground whitespace-nowrap">
        <span className="w-20 text-right">{file.isDir ? '--' : formatSize(file.size)}</span>
        <span className="w-40 md:block hidden">{file.modTime}</span>
        
        <div className="w-10 flex justify-end" onClick={e => e.stopPropagation()}>
          <FileItemActions 
            file={file} 
            onDelete={onDelete} 
            onRename={onRename} 
            onZip={onZip} 
            onExtract={onExtract} 
          />
        </div>
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
        <Button variant="ghost" size="icon" className="h-8 w-8 rounded-full hover:bg-background shadow-sm border border-transparent hover:border-border transition-all">
          <MoreVertical className="w-4 h-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-48 p-1">
        <DropdownMenuItem onClick={() => onRename(file.name)}>
          <Pencil className="mr-2 w-4 h-4" />
          Rename
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => onZip(file.name)}>
          <Box className="mr-2 w-4 h-4" />
          Archive (Zip)
        </DropdownMenuItem>
        {file.name.endsWith('.zip') && (
          <DropdownMenuItem onClick={() => onExtract(file.name)}>
            <ExternalLink className="mr-2 w-4 h-4" />
            Extract Here
          </DropdownMenuItem>
        )}
        <DropdownMenuSeparator />
        <DropdownMenuItem 
          className="text-destructive focus:text-destructive focus:bg-destructive/10"
          onClick={() => onDelete(file.name)}
        >
          <Trash2 className="mr-2 w-4 h-4" />
          Delete
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
