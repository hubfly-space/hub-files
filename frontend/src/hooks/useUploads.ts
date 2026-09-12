import { useCallback, useRef, useState } from "react";

import { api } from "../api";
import type { UploadStatus } from "../components/UploadProgress";
import { uploadTree, type TreeItem } from "../utils/uploadTree";

const makeUploadId = () => {
  if (globalThis.crypto?.randomUUID) return globalThis.crypto.randomUUID();
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
};

const formatSpeed = (bytesPerSecond: number) => {
  if (bytesPerSecond > 1024 * 1024) {
    return `${(bytesPerSecond / (1024 * 1024)).toFixed(1)} MB/s`;
  }

  if (bytesPerSecond > 1024) {
    return `${(bytesPerSecond / 1024).toFixed(1)} KB/s`;
  }

  return `${bytesPerSecond.toFixed(0)} B/s`;
};

const formatEta = (seconds: number) => {
  if (seconds > 3600) {
    return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
  }

  if (seconds > 60) {
    return `${Math.floor(seconds / 60)}m ${Math.floor(seconds % 60)}s`;
  }

  return `${Math.max(0, Math.floor(seconds))}s`;
};

interface UploadTask {
  id: string;
  file: File;
  uploadId: string;
  uploadedBytes: number;
  targetPath: string;
}

export function useUploads(currentPath: string, onUploaded: () => void) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const folderInputRef = useRef<HTMLInputElement>(null);
  const [activeUploads, setActiveUploads] = useState<UploadStatus[]>([]);
  const controllersRef = useRef<Map<string, AbortController>>(new Map());
  const tasksRef = useRef<Map<string, UploadTask>>(new Map());

  const removeController = useCallback((id: string) => {
    controllersRef.current.delete(id);
  }, []);

  const updateUpload = useCallback(
    (
      id: string,
      updater: (
        upload: UploadStatus,
      ) => UploadStatus,
    ) => {
      setActiveUploads((prev) =>
        prev.map((u) => (u.id === id ? updater(u) : u)),
      );
    },
    [],
  );

  const cancelUpload = useCallback(
    (id: string) => {
      const controller = controllersRef.current.get(id);
      if (controller) {
        controller.abort();
        controllersRef.current.delete(id);
      }
      tasksRef.current.delete(id);
      setActiveUploads((prev) => prev.filter((u) => u.id !== id));
    },
    [],
  );

  const pauseUpload = useCallback((id: string) => {
    const controller = controllersRef.current.get(id);
    if (controller) {
      controller.abort();
      controllersRef.current.delete(id);
    }
    setActiveUploads((prev) =>
      prev.map((u) =>
        u.id === id && u.status === "uploading"
          ? { ...u, status: "paused", speed: undefined, eta: undefined }
          : u,
      ),
    );
  }, []);

  const resumeUpload = useCallback(
    (id: string) => {
      const task = tasksRef.current.get(id);
      if (!task) return;

      const controller = new AbortController();
      controllersRef.current.set(id, controller);

      const startTime = Date.now();
      let lastUiUpdate = 0;

      updateUpload(id, (u) => ({ ...u, status: "uploading", error: undefined }));

      api
        .upload(
          task.targetPath,
          task.file,
          (loaded, total) => {
            const now = Date.now();
            if (now - lastUiUpdate < 250 && loaded < total) return;
            lastUiUpdate = now;

            const progress = total > 0 ? Math.round((loaded / total) * 100) : 100;
            const elapsed = (now - startTime) / 1000;
            const speedBytes = elapsed > 0 ? loaded / elapsed : 0;
            const remainingBytes = Math.max(0, total - loaded);
            const eta =
              speedBytes > 0 ? formatEta(remainingBytes / speedBytes) : "";

            task.uploadedBytes = loaded;

            updateUpload(id, (u) => ({
              ...u,
              progress,
              speed: formatSpeed(speedBytes),
              eta,
            }));
          },
          controller.signal,
          task.uploadId,
          task.uploadedBytes,
        )
        .then(() => {
          removeController(id);
          tasksRef.current.delete(id);
          updateUpload(id, (u) => ({
            ...u,
            status: "completed",
            progress: 100,
            speed: undefined,
            eta: undefined,
          }));
          onUploaded();
        })
        .catch((err) => {
          removeController(id);
          if (err.name === "AbortError") {
            // Only remove if cancelled (not paused)
            if (!tasksRef.current.has(id)) return;
            updateUpload(id, (u) => ({
              ...u,
              status: "paused",
              speed: undefined,
              eta: undefined,
            }));
          } else {
            const message = err instanceof Error ? err.message : "Upload failed";
            updateUpload(id, (u) => ({
              ...u,
              status: "error",
              error: message,
              speed: undefined,
              eta: undefined,
            }));
          }
        });
    },
    [onUploaded, removeController, updateUpload],
  );

  const retryUpload = useCallback(
    (id: string) => {
      const oldTask = tasksRef.current.get(id);
      if (!oldTask) return;

      const newUploadId = makeUploadId();
      tasksRef.current.set(id, { ...oldTask, uploadId: newUploadId, uploadedBytes: 0 });

      updateUpload(id, (u) => ({
        ...u,
        progress: 0,
        status: "uploading",
        error: undefined,
        speed: undefined,
        eta: undefined,
      }));

      const controller = new AbortController();
      controllersRef.current.set(id, controller);
      const startTime = Date.now();
      let lastUiUpdate = 0;

      api
        .upload(
          oldTask.targetPath,
          oldTask.file,
          (loaded, total) => {
            const now = Date.now();
            if (now - lastUiUpdate < 250 && loaded < total) return;
            lastUiUpdate = now;

            const progress = total > 0 ? Math.round((loaded / total) * 100) : 100;
            const elapsed = (now - startTime) / 1000;
            const speedBytes = elapsed > 0 ? loaded / elapsed : 0;
            const remainingBytes = Math.max(0, total - loaded);
            const eta =
              speedBytes > 0 ? formatEta(remainingBytes / speedBytes) : "";

            const task = tasksRef.current.get(id);
            if (task) task.uploadedBytes = loaded;

            updateUpload(id, (u) => ({
              ...u,
              progress,
              speed: formatSpeed(speedBytes),
              eta,
            }));
          },
          controller.signal,
          newUploadId,
        )
        .then(() => {
          removeController(id);
          tasksRef.current.delete(id);
          updateUpload(id, (u) => ({
            ...u,
            status: "completed",
            progress: 100,
            speed: undefined,
            eta: undefined,
          }));
          onUploaded();
        })
        .catch((err) => {
          removeController(id);
          if (err.name === "AbortError") {
            if (tasksRef.current.has(id)) {
              updateUpload(id, (u) => ({
                ...u,
                status: "paused",
                speed: undefined,
                eta: undefined,
              }));
            }
          } else {
            const message = err instanceof Error ? err.message : "Upload failed";
            updateUpload(id, (u) => ({
              ...u,
              status: "error",
              error: message,
              speed: undefined,
              eta: undefined,
            }));
          }
        });
    },
    [onUploaded, removeController, updateUpload],
  );

  const uploadFiles = useCallback(
    async (files: File[], targetPath?: string) => {
      if (files.length === 0) return;

      const destPath = targetPath || currentPath;

      const uploads = files.map((file) => ({
        id: makeUploadId(),
        file,
        uploadId: makeUploadId(),
        uploadedBytes: 0,
        targetPath: destPath,
      }));

      for (const u of uploads) {
        tasksRef.current.set(u.id, u);
      }

      let hasSuccessfulUpload = false;

      setActiveUploads((previousUploads) => [
        ...uploads.map(({ id, file }) => ({
          id,
          name: file.name,
          progress: 0,
          status: "uploading" as const,
          uploadId: id,
        })),
        ...previousUploads,
      ]);

      const uploadOne = async (task: UploadTask) => {
        const controller = new AbortController();
        controllersRef.current.set(task.id, controller);
        const startTime = Date.now();
        let lastUiUpdate = 0;

        try {
          await api.upload(
            task.targetPath,
            task.file,
            (loaded, total) => {
              const now = Date.now();
              if (now - lastUiUpdate < 250 && loaded < total) return;
              lastUiUpdate = now;

              const progress = total > 0 ? Math.round((loaded / total) * 100) : 100;
              const elapsed = (now - startTime) / 1000;
              const speedBytes = elapsed > 0 ? loaded / elapsed : 0;
              const remainingBytes = Math.max(0, total - loaded);
              const eta =
                speedBytes > 0 ? formatEta(remainingBytes / speedBytes) : "";

              task.uploadedBytes = loaded;

              setActiveUploads((prev) =>
                prev.map((u) =>
                  u.id === task.id
                    ? {
                        ...u,
                        progress,
                        speed: formatSpeed(speedBytes),
                        eta,
                      }
                    : u,
                ),
              );
            },
            controller.signal,
            task.uploadId,
          );

          removeController(task.id);
          tasksRef.current.delete(task.id);
          hasSuccessfulUpload = true;

          setActiveUploads((prev) =>
            prev.map((u) =>
              u.id === task.id
                ? {
                    ...u,
                    status: "completed",
                    progress: 100,
                    speed: undefined,
                    eta: undefined,
                  }
                : u,
            ),
          );
        } catch (err: unknown) {
          removeController(task.id);
          if ((err as Error).name === "AbortError") {
            if (tasksRef.current.has(task.id)) {
              setActiveUploads((prev) =>
                prev.map((u) =>
                  u.id === task.id
                    ? {
                        ...u,
                        status: "paused",
                        speed: undefined,
                        eta: undefined,
                      }
                    : u,
                ),
              );
            }
          } else {
            const message = err instanceof Error ? err.message : "Upload failed";
            setActiveUploads((prev) =>
              prev.map((u) =>
                u.id === task.id
                  ? {
                      ...u,
                      status: "error",
                      error: message,
                      speed: undefined,
                      eta: undefined,
                    }
                  : u,
              ),
            );
          }
        }
      };

      const queue = [...uploads];

      const workers = Array.from(
        { length: Math.min(2, queue.length) },
        async () => {
          for (;;) {
            const next = queue.shift();
            if (!next) return;
            await uploadOne(next);
          }
        },
      );

      await Promise.all(workers);

      if (hasSuccessfulUpload) {
        onUploaded();
      }
    },
    [currentPath, onUploaded, removeController],
  );

  const handleFileChange = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      const files = Array.from(event.target.files || []);
      if (files.length > 0) {
        uploadFiles(files);
        event.target.value = "";
      }
    },
    [uploadFiles],
  );

  const handleFolderChange = useCallback(
    async (event: React.ChangeEvent<HTMLInputElement>) => {
      const files = Array.from(event.target.files || []);
      event.target.value = "";
      if (files.length === 0) return;

      const items: TreeItem[] = [];
      for (const file of files) {
        const relativePath = (
          file as File & { webkitRelativePath?: string }
        ).webkitRelativePath;
        if (relativePath) {
          items.push({ file, relativePath });
        }
      }

      if (items.length === 0) {
        uploadFiles(files);
        return;
      }

      await uploadTree(
        items,
        currentPath,
        (dirPath) => api.mkdir(dirPath),
        (groupFiles, targetPath) => uploadFiles(groupFiles, targetPath),
      );
    },
    [currentPath, uploadFiles],
  );

  const handleUploadClick = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  const handleUploadFolderClick = useCallback(() => {
    folderInputRef.current?.click();
  }, []);

  const clearUpload = useCallback((id: string) => {
    controllersRef.current.delete(id);
    tasksRef.current.delete(id);
    setActiveUploads((prev) => prev.filter((u) => u.id !== id));
  }, []);

  const clearAllUploads = useCallback(() => {
    for (const [id, controller] of controllersRef.current) {
      controller.abort();
    }
    controllersRef.current.clear();
    tasksRef.current.clear();
    setActiveUploads([]);
  }, []);

  return {
    fileInputRef,
    folderInputRef,
    activeUploads,
    handleUploadClick,
    handleUploadFolderClick,
    handleFileChange,
    handleFolderChange,
    uploadFiles,
    clearUpload,
    clearAllUploads,
    cancelUpload,
    pauseUpload,
    resumeUpload,
    retryUpload,
  };
}
