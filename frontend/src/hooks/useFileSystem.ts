import { useState, useEffect, useCallback } from 'react';
import { api } from '../api';
import type { FileInfo } from '../api';

export function useFileSystem() {
  const [path, setPath] = useState('/');
  const [files, setFiles] = useState<FileInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<'list' | 'grid'>('list');

  const loadFiles = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.list(path);
      setFiles(data);
      setError(null);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, [path]);

  useEffect(() => {
    loadFiles();
  }, [loadFiles]);

  const navigate = (newPath: string) => {
    setPath(newPath);
  };

  const navigateUp = () => {
    if (path === '/') return;
    const parts = path.split('/').filter(Boolean);
    parts.pop();
    setPath('/' + parts.join('/'));
  };

  const refresh = () => loadFiles();

  const deleteItem = async (itemName: string) => {
    try {
      const itemPath = path === '/' ? `/${itemName}` : `${path}/${itemName}`;
      await api.delete(itemPath);
      refresh();
    } catch (err: any) {
      console.error(err);
    }
  };

  const renameItem = async (oldName: string, newName: string) => {
    try {
      const oldPath = path === '/' ? `/${oldName}` : `${path}/${oldName}`;
      const newPath = path === '/' ? `/${newName}` : `${path}/${newName}`;
      await api.rename(oldPath, newPath);
      refresh();
    } catch (err: any) {
      console.error(err);
    }
  };

  const createFolder = async (name: string) => {
    try {
      const folderPath = path === '/' ? `/${name}` : `${path}/${name}`;
      await api.mkdir(folderPath);
      refresh();
    } catch (err: any) {
      console.error(err);
    }
  };

  const zipItem = async (itemName: string) => {
    try {
      const itemPath = path === '/' ? `/${itemName}` : `${path}/${itemName}`;
      const targetPath = `${itemPath}.zip`;
      await api.zip(itemPath, targetPath);
      refresh();
    } catch (err: any) {
      console.error(err);
    }
  };

  const extractItem = async (itemName: string) => {
    if (!itemName.endsWith('.zip')) return;
    try {
      const itemPath = path === '/' ? `/${itemName}` : `${path}/${itemName}`;
      const targetPath = path; 
      await api.extract(itemPath, targetPath);
      refresh();
    } catch (err: any) {
      console.error(err);
    }
  };

  return {
    path,
    files,
    loading,
    error,
    viewMode,
    setViewMode,
    navigate,
    navigateUp,
    refresh,
    deleteItem,
    renameItem,
    createFolder,
    zipItem,
    extractItem,
  };
}
