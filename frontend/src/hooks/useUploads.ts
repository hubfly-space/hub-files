import { useCallback, useRef, useState } from "react";

import { api } from "../api";
import type { UploadStatus } from "../components/UploadProgress";

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

export function useUploads(currentPath: string, onUploaded: () => void) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [activeUploads, setActiveUploads] = useState<UploadStatus[]>([]);

  const handleUploadClick = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  const uploadFiles = useCallback(
    async (files: File[]) => {
      if (files.length === 0) return;

      const uploads = files.map((file) => ({
        id: makeUploadId(),
        file,
      }));

      const uploadPath = currentPath;
      let hasSuccessfulUpload = false;

      setActiveUploads((previousUploads) => [
        ...uploads.map(({ id, file }) => ({
          id,
          name: file.name,
          progress: 0,
          status: "uploading" as const,
        })),
        ...previousUploads,
      ]);

      const uploadOne = async ({ id, file }: (typeof uploads)[number]) => {
        const startTime = Date.now();
        let lastUiUpdate = 0;

        try {
          await api.upload(uploadPath, file, (loaded, total) => {
            const now = Date.now();

            if (now - lastUiUpdate < 250 && loaded < total) return;
            lastUiUpdate = now;

            const progress = total > 0 ? Math.round((loaded / total) * 100) : 100;
            const elapsed = (now - startTime) / 1000;
            const speedBytes = elapsed > 0 ? loaded / elapsed : 0;
            const remainingBytes = Math.max(0, total - loaded);
            const eta =
              speedBytes > 0 ? formatEta(remainingBytes / speedBytes) : "";

            setActiveUploads((previousUploads) =>
              previousUploads.map((upload) =>
                upload.id === id
                  ? {
                      ...upload,
                      progress,
                      speed: formatSpeed(speedBytes),
                      eta,
                    }
                  : upload,
              ),
            );
          });

          hasSuccessfulUpload = true;

          setActiveUploads((previousUploads) =>
            previousUploads.map((upload) =>
              upload.id === id
                ? {
                    ...upload,
                    status: "completed",
                    progress: 100,
                    speed: undefined,
                    eta: undefined,
                  }
                : upload,
            ),
          );
        } catch (err: unknown) {
          const message = err instanceof Error ? err.message : "Upload failed";

          setActiveUploads((previousUploads) =>
            previousUploads.map((upload) =>
              upload.id === id
                ? {
                    ...upload,
                    status: "error",
                    error: message,
                    speed: undefined,
                    eta: undefined,
                  }
                : upload,
            ),
          );
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
    [currentPath, onUploaded],
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

  const clearUpload = useCallback((id: string) => {
    setActiveUploads((previousUploads) =>
      previousUploads.filter((upload) => upload.id !== id),
    );
  }, []);

  const clearAllUploads = useCallback(() => {
    setActiveUploads([]);
  }, []);

  return {
    fileInputRef,
    activeUploads,
    handleUploadClick,
    handleFileChange,
    uploadFiles,
    clearUpload,
    clearAllUploads,
  };
}
