import React, { useState } from 'react';
import type { FileInfo } from '../api';
import { FileIcon } from './FileIcon';

interface FileItemProps {
  file: FileInfo;
  viewMode: 'list' | 'grid';
  onNavigate: (name: string) => void;
  onDelete: (name: string) => void;
  onRename: (oldName: string, newName: string) => void;
  onZip: (name: string) => void;
  onExtract: (name: string) => void;
}

export const FileItem: React.FC<FileItemProps> = ({ 
  file, viewMode, onNavigate, onDelete, onRename, onZip, onExtract
}) => {
  const [isRenaming, setIsRenaming] = useState(false);
  const [newName, setNewName] = useState(file.name);

  const handleRenameSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onRename(file.name, newName);
    setIsRenaming(false);
  };

  const formatSize = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  };

  if (viewMode === 'grid') {
    return (
      <div className="file-item grid" onDoubleClick={() => onNavigate(file.name)}>
        <FileIcon isDir={file.isDir} name={file.name} className="icon-large" />
        <div className="file-name">
          {isRenaming ? (
            <form onSubmit={handleRenameSubmit}>
              <input 
                autoFocus 
                value={newName} 
                onChange={e => setNewName(e.target.value)}
                onBlur={() => setIsRenaming(false)}
              />
            </form>
          ) : (
            <span>{file.name}</span>
          )}
        </div>
        <div className="hover-actions">
          <button title="Zip" onClick={() => onZip(file.name)}>📦</button>
          {file.name.endsWith('.zip') && (
            <button title="Extract" onClick={() => onExtract(file.name)}>📂</button>
          )}
          <button title="Rename" onClick={() => setIsRenaming(true)}>✎</button>
          <button title="Delete" onClick={() => onDelete(file.name)}>✕</button>
        </div>
      </div>
    );
  }

  return (
    <div className="file-item list" onDoubleClick={() => onNavigate(file.name)}>
      <div className="file-info-main">
        <FileIcon isDir={file.isDir} name={file.name} className="icon-small" />
        {isRenaming ? (
          <form onSubmit={handleRenameSubmit}>
            <input 
              autoFocus 
              value={newName} 
              onChange={e => setNewName(e.target.value)}
              onBlur={() => setIsRenaming(false)}
            />
          </form>
        ) : (
          <span className="name">{file.name}</span>
        )}
      </div>
      <div className="file-meta">
        <span className="size">{file.isDir ? '--' : formatSize(file.size)}</span>
        <span className="date">{file.modTime}</span>
      </div>
      <div className="hover-actions">
        <button title="Zip" onClick={() => onZip(file.name)}>📦</button>
        {file.name.endsWith('.zip') && (
          <button title="Extract" onClick={() => onExtract(file.name)}>📂</button>
        )}
        <button title="Rename" onClick={() => setIsRenaming(true)}>✎</button>
        <button title="Delete" onClick={() => onDelete(file.name)}>✕</button>
      </div>
    </div>
  );
};
