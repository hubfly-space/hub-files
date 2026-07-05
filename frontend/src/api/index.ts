// Use relative URL for API calls - works with same-origin or reverse proxy
const API_BASE = "http://localhost:10015/api";

const CHUNK_SIZE = 5 * 1024 * 1024; // 5 MB per chunk

const SESSION_COOKIE = "session";

function getCookie(name: string): string | null {
  const match = document.cookie.match(
    new RegExp(`(?:^|;\\s*)${name}=([^;]*)`),
  );
  return match ? decodeURIComponent(match[1]) : null;
}

function setCookie(
  name: string,
  value: string,
  maxAgeSeconds: number,
): void {
  document.cookie = `${name}=${encodeURIComponent(value)}; path=/; max-age=${maxAgeSeconds}; SameSite=Lax`;
}

function initSession(): void {
  const params = new URLSearchParams(window.location.search);
  const token = params.get("session");
  if (!token) return;

  setCookie(SESSION_COOKIE, token, 86400);
  params.delete("session");
  const newURL = params.toString()
    ? `${window.location.pathname}?${params}`
    : window.location.pathname;
  window.history.replaceState(null, "", newURL);
}

initSession();

export interface FileInfo {
  name: string;
  isDir: boolean;
  size: number;
  modTime: string;
}

export interface SearchResult {
  baseName: string;
  relPath: string;
  isDir: boolean;
  size: number;
  modTime: string;
}

export interface ListResult {
  items: FileInfo[];
  total: number;
}

export interface StorageInfo {
  path: string;
  totalBytes: number;
  usedBytes: number;
  availableBytes: number;
  usedPercent: number;
}

export interface SessionInfo {
  root: string;
  type: "local" | "smb" | "ftp";
  canHostMount: boolean;
  hostMountRoot?: string;
  readonly: boolean;
  allowUpload: boolean;
  allowEdit: boolean;
  allowDelete: boolean;
}

export interface HostMountResult {
  mountPath: string;
  alreadyMounted: boolean;
}

export interface HostUnmountResult {
  mountPath: string;
  wasMounted: boolean;
}

export const api = {
  getToken: () => {
    return getCookie(SESSION_COOKIE) || "demo";
  },

  headers: () => ({
    Authorization: `Bearer ${api.getToken()}`,
    "Content-Type": "application/json",
  }),

  session: async (): Promise<SessionInfo> => {
    const res = await fetch(`${API_BASE}/session`, {
      headers: api.headers(),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  hostMount: async (): Promise<HostMountResult> => {
    const res = await fetch(`${API_BASE}/host-mount`, {
      method: "POST",
      headers: api.headers(),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  hostUnmount: async (): Promise<HostUnmountResult> => {
    const res = await fetch(`${API_BASE}/host-unmount`, {
      method: "POST",
      headers: api.headers(),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  list: async (
    path: string,
    options?: { offset?: number; limit?: number },
  ): Promise<ListResult> => {
    const params = new URLSearchParams({ path });
    if (options?.offset) params.set("offset", String(options.offset));
    if (options?.limit) params.set("limit", String(options.limit));
    const res = await fetch(`${API_BASE}/list?${params}`, {
      headers: api.headers(),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  storage: async (path: string): Promise<StorageInfo> => {
    const res = await fetch(
      `${API_BASE}/storage?path=${encodeURIComponent(path)}`,
      {
        headers: api.headers(),
      },
    );
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  search: async (query: string): Promise<SearchResult[]> => {
    const res = await fetch(`${API_BASE}/search?q=${encodeURIComponent(query)}`, {
      headers: api.headers(),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  getFile: async (path: string): Promise<string> => {
    const res = await fetch(
      `${API_BASE}/file?path=${encodeURIComponent(path)}`,
      {
        headers: api.headers(),
      },
    );
    if (!res.ok) throw new Error(await res.text());
    return res.text();
  },

  putFile: async (path: string, content: string): Promise<void> => {
    const res = await fetch(
      `${API_BASE}/file?path=${encodeURIComponent(path)}`,
      {
        method: "PUT",
        headers: api.headers(),
        body: content,
      },
    );
    if (!res.ok) throw new Error(await res.text());
  },

  upload: (
    path: string,
    file: File,
    onProgress?: (loaded: number, total: number) => void,
    signal?: AbortSignal,
    uploadId?: string,
    startOffset?: number,
  ): Promise<void> => {
    const totalSize = file.size;
    const totalChunks = Math.ceil(totalSize / CHUNK_SIZE);
    const id =
      uploadId ||
      globalThis.crypto?.randomUUID?.() ||
      `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
    let uploadedBytes = startOffset ?? 0;
    const startChunk = startOffset
      ? Math.floor(startOffset / CHUNK_SIZE)
      : 0;

    const uploadChunk = (chunkIndex: number): Promise<void> =>
      new Promise((resolve, reject) => {
        if (signal?.aborted) {
          reject(new DOMException("Upload cancelled", "AbortError"));
          return;
        }

        const start = chunkIndex * CHUNK_SIZE;
        const end = Math.min(start + CHUNK_SIZE, totalSize) - 1;
        const chunk = file.slice(start, end + 1);
        const params = new URLSearchParams({
          path,
          filename: file.name,
          uploadId: id,
        });

        const xhr = new XMLHttpRequest();

        const onAbort = () => {
          xhr.abort();
          reject(new DOMException("Upload cancelled", "AbortError"));
        };
        signal?.addEventListener("abort", onAbort, { once: true });

        if (onProgress && xhr.upload) {
          xhr.upload.onprogress = (e) => {
            if (e.lengthComputable) {
              onProgress(uploadedBytes + e.loaded, totalSize);
            }
          };
        }

        xhr.open("POST", `${API_BASE}/upload?${params.toString()}`);
        xhr.setRequestHeader("Authorization", `Bearer ${api.getToken()}`);
        xhr.setRequestHeader(
          "Content-Type",
          file.type || "application/octet-stream",
        );
        xhr.setRequestHeader(
          "Content-Range",
          `bytes ${start}-${end}/${totalSize}`,
        );

        xhr.onload = () => {
          signal?.removeEventListener("abort", onAbort);
          if (xhr.status >= 200 && xhr.status < 300) {
            uploadedBytes += chunk.size;
            resolve();
          } else {
            reject(new Error(xhr.responseText || "Chunk upload failed"));
          }
        };

        xhr.onerror = () => {
          signal?.removeEventListener("abort", onAbort);
          reject(new Error("Network error"));
        };
        xhr.onabort = () => {
          signal?.removeEventListener("abort", onAbort);
          reject(new DOMException("Upload cancelled", "AbortError"));
        };

        xhr.send(chunk);
      });

    const uploadAllChunks = async () => {
      for (let i = startChunk; i < totalChunks; i++) {
        if (signal?.aborted)
          throw new DOMException("Upload cancelled", "AbortError");
        await uploadChunk(i);
      }
    };

    return uploadAllChunks();
  },

  mkdir: async (path: string): Promise<void> => {
    const res = await fetch(`${API_BASE}/mkdir`, {
      method: "POST",
      headers: api.headers(),
      body: JSON.stringify({ path }),
    });
    if (!res.ok) throw new Error(await res.text());
  },

  touch: async (path: string): Promise<void> =>{
    const res = await fetch(`${API_BASE}/touch`, {
      method: "POST",
      headers: api.headers(),
      body:JSON.stringify({path})
    })
    if (!res.ok) throw new Error(await res.text());
},

  rename: async (oldPath: string, newPath: string): Promise<void> => {
    const res = await fetch(`${API_BASE}/rename`, {
      method: "POST",
      headers: api.headers(),
      body: JSON.stringify({ oldPath, newPath }),
    });
    if (!res.ok) throw new Error(await res.text());
  },

  delete: async (path: string): Promise<void> => {
    const res = await fetch(
      `${API_BASE}/delete?path=${encodeURIComponent(path)}`,
      {
        method: "DELETE",
        headers: api.headers(),
      },
    );
    if (!res.ok) throw new Error(await res.text());
  },

  zip: async (source: string, target: string): Promise<void> => {
    const res = await fetch(`${API_BASE}/zip`, {
      method: "POST",
      headers: api.headers(),
      body: JSON.stringify({ source, target }),
    });
    if (!res.ok) throw new Error(await res.text());
  },

  extract: async (source: string, target: string): Promise<void> => {
    const res = await fetch(`${API_BASE}/extract`, {
      method: "POST",
      headers: api.headers(),
      body: JSON.stringify({ source, target }),
    });
    if (!res.ok) throw new Error(await res.text());
  },
};
