import React from 'react';

interface ToolbarProps {
  viewMode: 'list' | 'grid';
  onViewToggle: () => void;
  onRefresh: () => void;
  onUpload: () => void;
  onNewFolder: () => void;
}

export const Toolbar: React.FC<ToolbarProps> = ({ 
  viewMode, onViewToggle, onRefresh, onUpload, onNewFolder 
}) => {
  return (
    <div className="actions">
      <button title="Upload" onClick={onUpload}>⬆</button>
      <button title="New Folder" onClick={onNewFolder}>＋</button>
      <button title="Refresh" onClick={onRefresh}>⟳</button>
      <button title="View Toggle" onClick={onViewToggle}>
        {viewMode === 'list' ? '⧉' : '☰'}
      </button>
    </div>
  );
};
