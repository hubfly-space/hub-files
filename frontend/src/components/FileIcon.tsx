import React from 'react';
import { 
  File, 
  Folder, 
  FileText, 
  Image as ImageIcon, 
  Archive, 
  Code, 
  FileJson,
  FileCode,
  Music,
  Video
} from 'lucide-react';

interface FileIconProps {
  name: string;
  isDir: boolean;
  className?: string;
}

export const FileIcon: React.FC<FileIconProps> = ({ name, isDir, className }) => {
  if (isDir) return <Folder className={className} />;
  
  const ext = name.split('.').pop()?.toLowerCase();
  
  switch (ext) {
    case 'pdf': return <FileText className={className} />;
    case 'jpg':
    case 'jpeg':
    case 'png':
    case 'gif':
    case 'svg':
    case 'webp': return <ImageIcon className={className} />;
    case 'zip':
    case 'gz':
    case 'tar':
    case 'rar':
    case '7z': return <Archive className={className} />;
    case 'md': return <FileText className={className} />;
    case 'js':
    case 'ts':
    case 'tsx':
    case 'jsx':
    case 'go':
    case 'py':
    case 'cpp':
    case 'h':
    case 'rs': return <Code className={className} />;
    case 'json': return <FileJson className={className} />;
    case 'html':
    case 'css': return <FileCode className={className} />;
    case 'mp3':
    case 'wav':
    case 'ogg': return <Music className={className} />;
    case 'mp4':
    case 'mov':
    case 'avi': return <Video className={className} />;
    default: return <File className={className} />;
  }
};
