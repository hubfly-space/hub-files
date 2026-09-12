import { dirname, joinPath } from "./path";

export interface TreeItem {
  file: File;
  relativePath: string;
}

function dirDepth(p: string): number {
  return p.split("/").filter(Boolean).length;
}

// Order directories shallow-to-deep so backends with non-recursive mkdir
// (e.g. FTP MakeDir) can create every parent before its children.
function sortDirPaths(dirs: string[]): string[] {
  return [...new Set(dirs)].sort(
    (a, b) => dirDepth(a) - dirDepth(b) || a.localeCompare(b),
  );
}

// Expand each directory to include all of its ancestors (up to but excluding
// basePath, which already exists). Folder pickers only enumerate leaf files,
// so intermediate container dirs (e.g. "static" holding only "static/description")
// must be derived explicitly for non-recursive mkdir backends.
function expandDirAncestors(dirs: string[], basePath: string): string[] {
  const expanded = new Set<string>();
  for (const dir of dirs) {
    let cur: string | undefined = dir;
    while (cur && cur !== basePath) {
      expanded.add(cur);
      const parent = dirname(cur);
      if (parent === cur) break;
      cur = parent;
    }
  }
  return Array.from(expanded);
}

export async function uploadTree(
  items: TreeItem[],
  basePath: string,
  mkdir: (dirPath: string) => Promise<void>,
  upload: (files: File[], targetPath: string) => Promise<void>,
  extraDirs: string[] = [],
): Promise<void> {
  if (items.length === 0 && extraDirs.length === 0) return;

  const dirs = new Set<string>();
  for (const dir of extraDirs) {
    if (dir) dirs.add(joinPath(basePath, dir));
  }
  for (const { relativePath } of items) {
    const dir = dirname(relativePath);
    if (dir && dir !== ".") dirs.add(joinPath(basePath, dir));
  }

  for (const dirPath of sortDirPaths(
    expandDirAncestors(Array.from(dirs), basePath),
  )) {
    try {
      await mkdir(dirPath);
    } catch {
      // Directory may already exist.
    }
  }

  const groups = new Map<string, File[]>();
  for (const { file, relativePath } of items) {
    const dir = dirname(relativePath);
    const baseName = relativePath.split("/").pop() || file.name;
    const targetDir = dir && dir !== "." ? joinPath(basePath, dir) : basePath;
    const renamedFile = new File([file], baseName, { type: file.type });
    const group = groups.get(targetDir) || [];
    group.push(renamedFile);
    groups.set(targetDir, group);
  }

  for (const [targetDir, groupFiles] of groups) {
    await upload(groupFiles, targetDir);
  }
}