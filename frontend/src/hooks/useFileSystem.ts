import { useState, useEffect, useCallback } from "react";
import { api } from "../api";
import type { FileInfo, SessionInfo, StorageInfo } from "../api";

const PAGE_SIZE = 50;

function pathFromURL(): string {
  const p = window.location.pathname;
  return p === "/" ? "/" : p.replace(/\/$/, "");
}

export function useFileSystem() {
  const [path, setPath] = useState(pathFromURL);
  const [files, setFiles] = useState<FileInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<"list" | "grid">("list");
  const [storage, setStorage] = useState<StorageInfo | null>(null);
  const [session, setSession] = useState<SessionInfo | null>(null);
  const [total, setTotal] = useState(0);

  const hasMore = total > files.length;

  const loadPage = useCallback(
    async (pageOffset: number) => {
      setLoading(true);
      try {
        const [sessionData, result, storageData] = await Promise.all([
          api.session(),
          api.list(path, { offset: pageOffset, limit: PAGE_SIZE }),
          api.storage(path),
        ]);
        setSession(sessionData);
        setFiles((prev) =>
          pageOffset === 0
            ? result.items ?? []
            : [...prev, ...(result.items ?? [])],
        );
        setTotal(result.total);
        setStorage(storageData);
        setError(null);
      } catch (err: unknown) {
        setError(err instanceof Error ? err.message : "Something went wrong");
      } finally {
        setLoading(false);
      }
    },
    [path],
  );

  useEffect(() => {
    setFiles([]);
    setTotal(0);
    loadPage(0);
  }, [loadPage]);

  useEffect(() => {
    const handlePopState = () => {
      setPath(pathFromURL());
    };
    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, []);

  const navigate = (newPath: string) => {
    window.history.pushState(null, "", newPath === "/" ? "/" : newPath);
    setPath(newPath);
  };

  const navigateUp = () => {
    if (path === "/") return;
    const parts = path.split("/").filter(Boolean);
    parts.pop();
    navigate("/" + parts.join("/"));
  };

  const refresh = () => {
    setFiles([]);
    setTotal(0);
    loadPage(0);
  };

  const loadMore = () => {
    if (!hasMore || loading) return;
    loadPage(files.length);
  };

  const deleteItem = async (itemName: string) => {
    const itemPath = path === "/" ? `/${itemName}` : `${path}/${itemName}`;
    await api.delete(itemPath);
    refresh();
  };

  const renameItem = async (oldName: string, newName: string) => {
    const oldPath = path === "/" ? `/${oldName}` : `${path}/${oldName}`;
    const newPath = path === "/" ? `/${newName}` : `${path}/${newName}`;

    await api.rename(oldPath, newPath);
    refresh();
  };

  const createFolder = async (name: string) => {
    const folderPath = path === "/" ? `/${name}` : `${path}/${name}`;
    await api.mkdir(folderPath);
    refresh();
  };

  const createFile = async (name: string) => {
    try {
      const filePath = path === "/" ? `/${name}` : `${path}/${name}`;
      await api.touch(filePath);
      refresh();
    } catch (err: unknown) {
      if (err instanceof Error) {
        console.error(err);
        throw err;
      }
    }
  };

  const zipItem = async (itemName: string) => {
    const itemPath = path === "/" ? `/${itemName}` : `${path}/${itemName}`;
    // Only append .zip if not already a zip file
    const targetPath = itemName.endsWith(".zip") ? itemPath : `${itemPath}.zip`;
    await api.zip(itemPath, targetPath);
    refresh();
  };

  const extractItem = async (itemName: string) => {
    if (!itemName.endsWith(".zip")) return;

    const itemPath = path === "/" ? `/${itemName}` : `${path}/${itemName}`;
    const targetPath = path;
    await api.extract(itemPath, targetPath);
    refresh();
  };

  return {
    path,
    files,
    loading,
    error,
    storage,
    session,
    viewMode,
    setViewMode,
    navigate,
    navigateUp,
    refresh,
    deleteItem,
    renameItem,
    createFolder,
    zipItem,
    createFile,
    extractItem,
    total,
    hasMore,
    loadMore,
  };
}
