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

  upload: async (path: string, file: File): Promise<void> => {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('path', path);

    const res = await fetch(`${API_BASE}/upload`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${api.getToken()}`,
      },
      body: formData,
    });
    if (!res.ok) throw new Error(await res.text());
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
