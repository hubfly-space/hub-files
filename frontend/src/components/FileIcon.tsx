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
  Video,
  Database,
  Terminal,
  Settings,
  Shield,
  Layout,
  FileSearch,
  Book,
  FileSpreadsheet
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
    case 'pdf': return <Book className={className} />;
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
    case 'md':
    case 'txt':
    case 'rtf':
    case 'log': return <FileText className={className} />;
    case 'js':
    case 'ts':
    case 'tsx':
    case 'jsx':
    case 'go':
    case 'py':
    case 'cpp':
    case 'c':
    case 'h':
    case 'rs':
    case 'java':
    case 'rb': return <Code className={className} />;
    case 'json': return <FileJson className={className} />;
    case 'html':
    case 'htm': return <Layout className={className} />;
    case 'css':
    case 'scss':
    case 'sass':
    case 'less': return <FileCode className={className} />;
    case 'php': return <FileCode className={className} />;
    case 'sh':
    case 'bat':
    case 'cmd':
    case 'ps1': return <Terminal className={className} />;
    case 'mp3':
    case 'wav':
    case 'ogg': return <Music className={className} />;
    case 'mp4':
    case 'mov':
    case 'avi':
    case 'mkv':
    case 'webm': return <Video className={className} />;
    case 'sql':
    case 'db':
    case 'sqlite': return <Database className={className} />;
    case 'env':
    case 'ini':
    case 'conf':
    case 'yaml':
    case 'yml':
    case 'toml': return <Settings className={className} />;
    case 'csv':
    case 'xls':
    case 'xlsx': return <FileSpreadsheet className={className} />;
    case 'key':
    case 'pem':
    case 'crt':
    case 'cer': return <Shield className={className} />;
    default: return <File className={className} />;
  }
};
