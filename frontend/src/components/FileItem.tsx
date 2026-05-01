import React from 'react';
import type { FileInfo } from '../api';
import { FileIcon } from './FileIcon';
import { 
  MoreVertical, 
  Pencil, 
  Trash2, 
  Box, 
  ExternalLink 
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

interface FileItemProps {
  file: FileInfo;
  viewMode: 'list' | 'grid';
  onNavigate: (name: string) => void;
  onDelete: (name: string) => void;
  onRename: (name: string) => void;
  onZip: (name: string) => void;
  onExtract: (name: string) => void;
}

export const FileItem: React.FC<FileItemProps> = ({ 
  file, viewMode, onNavigate, onDelete, onRename, onZip, onExtract
}) => {
  const formatSize = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  };

  const itemVariants = {
    initial: { opacity: 0, y: 10 },
    animate: { opacity: 1, y: 0 },
    exit: { opacity: 0, scale: 0.95 }
  };

  if (viewMode === 'grid') {
    return (
      <motion.div 
        variants={itemVariants}
        layout
        className="group relative flex flex-col items-center p-4 rounded-xl border border-transparent hover:border-border hover:bg-accent/50 transition-all cursor-pointer"
        onDoubleClick={() => onNavigate(file.name)}
      >
        <div className="mb-3 p-4 rounded-lg bg-secondary/50 group-hover:bg-secondary transition-colors">
          <FileIcon isDir={file.isDir} name={file.name} className="w-12 h-12 text-primary/80" />
        </div>
        
        <span className="text-sm font-medium text-center truncate w-full px-1">
          {file.name}
        </span>
        
        <div className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity">
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
      className="group flex items-center justify-between p-2 px-4 rounded-lg hover:bg-accent/50 transition-colors cursor-pointer border border-transparent hover:border-border"
      onDoubleClick={() => onNavigate(file.name)}
    >
      <div className="flex items-center gap-4 flex-1 min-w-0">
        <FileIcon isDir={file.isDir} name={file.name} className="w-5 h-5 text-muted-foreground group-hover:text-primary transition-colors" />
        <span className="text-sm font-medium truncate">{file.name}</span>
      </div>
      
      <div className="flex items-center gap-6 text-xs text-muted-foreground whitespace-nowrap">
        <span className="w-20 text-right">{file.isDir ? '--' : formatSize(file.size)}</span>
        <span className="w-40">{file.modTime}</span>
        
        <div className="w-10 flex justify-end">
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
        <Button variant="ghost" size="icon" className="h-8 w-8">
          <MoreVertical className="w-4 h-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-48">
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
