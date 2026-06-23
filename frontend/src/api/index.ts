// Use relative URL for API calls - works with same-origin or reverse proxy
const API_BASE = "http://localhost:10015/api";

export interface FileInfo {
  name: string;
  isDir: boolean;
  size: number;
  modTime: string;
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
    const params = new URLSearchParams(window.location.search);
    return params.get("session") || "demo";
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

  list: async (path: string): Promise<FileInfo[]> => {
    const res = await fetch(
      `${API_BASE}/list?path=${encodeURIComponent(path)}`,
      {
        headers: api.headers(),
      },
    );
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
  ): Promise<void> => {
    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      const params = new URLSearchParams({
        path,
        filename: file.name,
      });

      xhr.open("POST", `${API_BASE}/upload?${params.toString()}`);
      xhr.setRequestHeader("Authorization", `Bearer ${api.getToken()}`);
      xhr.setRequestHeader(
        "Content-Type",
        file.type || "application/octet-stream",
      );

      if (onProgress && xhr.upload) {
        xhr.upload.onprogress = (e) => {
          if (e.lengthComputable) {
            onProgress(e.loaded, e.total);
          }
        };
      }

      xhr.onload = () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          resolve();
        } else {
          reject(new Error(xhr.responseText || "Upload failed"));
        }
      };

      xhr.onerror = () => reject(new Error("Network error"));
      xhr.onabort = () => reject(new Error("Upload cancelled"));
      xhr.send(file);
    });
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
