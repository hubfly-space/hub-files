import React from 'react';

interface FileIconProps {
  name: string;
  isDir: boolean;
  className?: string;
}

export const FileIcon: React.FC<FileIconProps> = ({ name, isDir, className }) => {
  if (isDir) return <span className={className}>📁</span>;
  
  const ext = name.split('.').pop()?.toLowerCase();
  
  switch (ext) {
    case 'pdf': return <span className={className}>📄</span>;
    case 'jpg':
    case 'jpeg':
    case 'png':
    case 'gif':
    case 'svg': return <span className={className}>🖼️</span>;
    case 'zip':
    case 'gz':
    case 'tar': return <span className={className}>📦</span>;
    case 'md': return <span className={className}>📝</span>;
    case 'js':
    case 'ts':
    case 'tsx':
    case 'go':
    case 'py': return <span className={className}>💻</span>;
    default: return <span className={className}>📄</span>;
  }
};
