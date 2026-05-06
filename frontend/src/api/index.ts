// Use relative URL for API calls - works with same-origin or reverse proxy
const API_BASE = '/api';

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

export const api = {
  getToken: () => {
    const params = new URLSearchParams(window.location.search);
    return params.get('session') || 'demo';
  },

  headers: () => ({
    'Authorization': `Bearer ${api.getToken()}`,
    'Content-Type': 'application/json',
  }),

  list: async (path: string): Promise<FileInfo[]> => {
    const res = await fetch(`${API_BASE}/list?path=${encodeURIComponent(path)}`, {
      headers: api.headers(),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  storage: async (path: string): Promise<StorageInfo> => {
    const res = await fetch(`${API_BASE}/storage?path=${encodeURIComponent(path)}`, {
      headers: api.headers(),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
  },

  getFile: async (path: string): Promise<string> => {
    const res = await fetch(`${API_BASE}/file?path=${encodeURIComponent(path)}`, {
      headers: api.headers(),
    });
    if (!res.ok) throw new Error(await res.text());
    return res.text();
  },

  putFile: async (path: string, content: string): Promise<void> => {
    const res = await fetch(`${API_BASE}/file?path=${encodeURIComponent(path)}`, {
      method: 'PUT',
      headers: api.headers(),
      body: content,
    });
    if (!res.ok) throw new Error(await res.text());
  },

  upload: (
    path: string,
    file: File,
    onProgress?: (percent: number) => void
  ): Promise<void> => {
    return new Promise((resolve, reject) => {
      const formData = new FormData();
      formData.append('file', file);
      formData.append('path', path);

      const xhr = new XMLHttpRequest();
      xhr.open('POST', `${API_BASE}/upload`);
      xhr.setRequestHeader('Authorization', `Bearer ${api.getToken()}`);

      if (onProgress && xhr.upload) {
        xhr.upload.onprogress = (e) => {
          if (e.lengthComputable) {
            const percent = Math.round((e.loaded / e.total) * 100);
            onProgress(percent);
          }
        };
      }

      xhr.onload = () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          resolve();
        } else {
          reject(new Error(xhr.responseText || 'Upload failed'));
        }
      };

      xhr.onerror = () => reject(new Error('Network error'));
      xhr.send(formData);
    });
  },

  mkdir: async (path: string): Promise<void> => {
    const res = await fetch(`${API_BASE}/mkdir`, {
      method: 'POST',
      headers: api.headers(),
      body: JSON.stringify({ path }),
    });
    if (!res.ok) throw new Error(await res.text());
  },

  rename: async (oldPath: string, newPath: string): Promise<void> => {
    const res = await fetch(`${API_BASE}/rename`, {
      method: 'POST',
      headers: api.headers(),
      body: JSON.stringify({ oldPath, newPath }),
    });
    if (!res.ok) throw new Error(await res.text());
  },

  delete: async (path: string): Promise<void> => {
    const res = await fetch(`${API_BASE}/delete?path=${encodeURIComponent(path)}`, {
      method: 'DELETE',
      headers: api.headers(),
    });
    if (!res.ok) throw new Error(await res.text());
  },

  zip: async (source: string, target: string): Promise<void> => {
    const res = await fetch(`${API_BASE}/zip`, {
      method: 'POST',
      headers: api.headers(),
      body: JSON.stringify({ source, target }),
    });
    if (!res.ok) throw new Error(await res.text());
  },

  extract: async (source: string, target: string): Promise<void> => {
    const res = await fetch(`${API_BASE}/extract`, {
      method: 'POST',
      headers: api.headers(),
      body: JSON.stringify({ source, target }),
    });
    if (!res.ok) throw new Error(await res.text());
  },
};
