export const joinPath = (basePath: string, name: string) => {
  if (basePath === "/") return `/${name}`;
  return `${basePath}/${name}`;
};

export const dirname = (filePath: string) => {
  const parts = filePath.split("/");
  if (parts.length <= 1) return ".";
  return parts.slice(0, -1).join("/");
};
