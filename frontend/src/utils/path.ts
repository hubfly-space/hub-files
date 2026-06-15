export const joinPath = (basePath: string, name: string) => {
  if (basePath === "/") return `/${name}`;
  return `${basePath}/${name}`;
};
