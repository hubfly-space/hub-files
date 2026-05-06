import React, { useRef, useState, useCallback } from "react";
import "./index.css";
import { useFileSystem } from "./hooks/useFileSystem";
import { Breadcrumb } from "./components/Breadcrumb";
import { Toolbar } from "./components/Toolbar";
import { FileItem } from "./components/FileItem";
import { FileViewer } from "./components/FileViewer";
import { UploadProgress } from "./components/UploadProgress";
import type { UploadStatus } from "./components/UploadProgress";
import { api } from "./api";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Toaster } from "@/components/ui/toaster";
import { useToast } from "@/hooks/use-toast";
import { motion, AnimatePresence } from "framer-motion";
import { HelpCircle, AlertCircle, Loader2, FolderOpen, Upload, Search } from "lucide-react";
import { cn } from "@/lib/utils";

function App() {
  const {
    path,
    files,
    loading,
    error,
    storage,
    viewMode,
    setViewMode,
    navigate,
    refresh,
    deleteItem,
    renameItem,
    createFolder,
    zipItem,
    extractItem,
  } = useFileSystem();

  const [openFile, setOpenFile] = useState<{
    path: string;
    name: string;
  } | null>(null);
  const [search, setSearch] = useState("");
  const [isDragging, setIsDragging] = useState(false);
  const { toast } = useToast();

  const [selectedItems, setSelectedItems] = useState<Set<string>>(new Set());
  const [selectionMode, setSelectionMode] = useState(false);
  const [activeUploads, setActiveUploads] = useState<UploadStatus[]>([]);

  const [renameDialog, setRenameDialog] = useState<{
    open: boolean;
    oldName: string;
    newName: string;
  }>({ open: false, oldName: "", newName: "" });
  const [newFolderDialog, setNewFolderDialog] = useState<{
    open: boolean;
    name: string;
  }>({ open: false, name: "" });
  const [deleteConfirm, setDeleteConfirm] = useState<{
    open: boolean;
    names: string[];
  }>({ open: false, names: [] });

  const formatBytes = (bytes: number) => {
    if (bytes <= 0) return "0 B";
    const units = ["B", "KB", "MB", "GB", "TB", "PB"];
    const exponent = Math.min(
      Math.floor(Math.log(bytes) / Math.log(1024)),
      units.length - 1,
    );
    const value = bytes / 1024 ** exponent;
    return `${value >= 100 || exponent === 0 ? value.toFixed(0) : value.toFixed(1)} ${units[exponent]}`;
  };

  const filteredFiles = files.filter((f) =>
    f.name.toLowerCase().includes(search.toLowerCase()),
  );

  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleUploadClick = () => fileInputRef.current?.click();

  const handleUpload = async (files: File[]) => {
    for (const file of files) {
      const id = Math.random().toString(36).substring(7);
      const newUpload: UploadStatus = {
        id,
        name: file.name,
        progress: 0,
        status: "uploading",
      };

      setActiveUploads((prev) => [newUpload, ...prev]);

      try {
        await api.upload(path, file, (progress) => {
          setActiveUploads((prev) =>
            prev.map((u) => (u.id === id ? { ...u, progress } : u)),
          );
        });

        setActiveUploads((prev) =>
          prev.map((u) =>
            u.id === id ? { ...u, status: "completed", progress: 100 } : u,
          ),
        );
        refresh();
      } catch (err: any) {
        setActiveUploads((prev) =>
          prev.map((u) =>
            u.id === id ? { ...u, status: "error", error: err.message } : u,
          ),
        );
      }
    }
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files || []);
    if (files.length > 0) {
      handleUpload(files);
      if (e.target) e.target.value = "";
    }
  };

  const handleDrop = async (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
    const files = Array.from(e.dataTransfer.files);
    if (files.length > 0) {
      handleUpload(files);
    }
  };

  const handleNavigate = (name: string) => {
    const item = files.find((f) => f.name === name);
    if (item?.isDir) {
      const newPath = path === "/" ? `/${name}` : `${path}/${name}`;
      navigate(newPath);
      setSelectedItems(new Set());
    } else {
      const filePath = path === "/" ? `/${name}` : `${path}/${name}`;
      setOpenFile({ path: filePath, name });
    }
  };

  const handleSelect = useCallback((name: string, multi: boolean) => {
    setSelectedItems((prev) => {
      const next = new Set(prev);
      if (multi) {
        if (next.has(name)) next.delete(name);
        else next.add(name);
      } else {
        if (next.has(name) && next.size === 1) next.delete(name);
        else {
          next.clear();
          next.add(name);
        }
      }
      return next;
    });
  }, []);

  const handleBulkDelete = async () => {
    const names = Array.from(selectedItems);
    let deleted = 0;
    let failed = 0;
    for (const name of names) {
      try {
        await deleteItem(name);
        deleted++;
      } catch (err) {
        console.error(`Failed to delete ${name}:`, err as unknown as string);
        failed++;
      }
    }
    setSelectedItems(new Set());
    if (failed > 0) {
      toast({
        title: "Partial delete",
        description: `${deleted} deleted, ${failed} failed.`,
        variant: "destructive",
      });
    } else {
      toast({ title: "Deleted", description: `${deleted} items removed.` });
    }
  };

  const handleBulkZip = async () => {
    const names = Array.from(selectedItems);
    let zipped = 0;
    let failed = 0;
    for (const name of names) {
      try {
        await zipItem(name);
        zipped++;
      } catch (err) {
        console.error(`Failed to zip ${name}:`, err as unknown as string);
        failed++;
      }
    }
    setSelectedItems(new Set());
    if (failed > 0) {
      toast({
        title: "Partial archive",
        description: `${zipped} zipped, ${failed} failed.`,
        variant: "destructive",
      });
    } else {
      toast({ title: "Archived", description: `${zipped} items zipped.` });
    }
  };

  return (
    <div className="fixed inset-0 bg-background flex flex-col selection:bg-primary/10">
      <div className="w-full h-full flex flex-col overflow-hidden">
        {!openFile && (
          <header className="shrink-0 border-b border-border/40 bg-background/95 backdrop-blur-xl z-30 px-6 py-4">
            <div className="flex flex-col gap-5 max-w-[1600px] mx-auto w-full">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                  <div className="flex items-center gap-2.5">
                    <div className="w-9 h-9 bg-primary rounded-xl flex items-center justify-center text-primary-foreground shadow-lg shadow-primary/20">
                      <FolderOpen className="w-5 h-5" />
                    </div>
                    <div className="hidden sm:block">
                      <h1 className="text-base font-bold tracking-tight leading-none">
                        HubFly
                      </h1>
                      <p className="text-[10px] uppercase tracking-widest text-muted-foreground mt-1 font-semibold">
                        Files Manager
                      </p>
                    </div>
                  </div>
                  <div className="h-6 w-px bg-border/60 mx-1 hidden md:block" />
                  <Breadcrumb path={path} onNavigate={navigate} />
                </div>

                {storage && (
                  <div className="hidden lg:flex items-center gap-4 bg-secondary/40 px-3 py-1.5 rounded-full border border-border/50">
                    <div className="flex flex-col items-end">
                      <div className="flex items-center gap-2">
                        <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">Storage</span>
                        <span className="text-xs font-semibold">{storage.usedPercent.toFixed(0)}%</span>
                      </div>
                      <p className="text-[10px] text-muted-foreground font-medium">
                        {formatBytes(storage.usedBytes)} of {formatBytes(storage.totalBytes)}
                      </p>
                    </div>
                    <div className="w-20 h-1.5 rounded-full bg-border/50 overflow-hidden">
                      <motion.div 
                        initial={{ width: 0 }}
                        animate={{ width: `${storage.usedPercent}%` }}
                        className={cn(
                          "h-full rounded-full transition-all",
                          storage.usedPercent > 90 ? "bg-destructive" : "bg-primary"
                        )}
                      />
                    </div>
                  </div>
                )}
              </div>

              <Toolbar
                viewMode={viewMode}
                onViewToggle={() =>
                  setViewMode(viewMode === "list" ? "grid" : "list")
                }
                onRefresh={refresh}
                onUpload={handleUploadClick}
                onNewFolder={() => setNewFolderDialog({ open: true, name: "" })}
                search={search}
                onSearchChange={setSearch}
                selectionMode={selectionMode}
                onSelectionModeToggle={() => {
                  setSelectionMode(!selectionMode);
                  if (selectionMode) setSelectedItems(new Set());
                }}
                selectedCount={selectedItems.size}
                onBulkDelete={() =>
                  setDeleteConfirm({
                    open: true,
                    names: Array.from(selectedItems),
                  })
                }
                onBulkZip={handleBulkZip}
                onClearSelection={() => setSelectedItems(new Set())}
              />
            </div>
            <input
              type="file"
              ref={fileInputRef}
              style={{ display: "none" }}
              onChange={handleFileChange}
            />
          </header>
        )}

        <main
          className={cn(
            "flex-1 min-h-0 relative",
            isDragging && "bg-primary/5 ring-2 ring-primary/20 ring-inset",
          )}
          onDragOver={(e) => {
            e.preventDefault();
            setIsDragging(true);
          }}
          onDragLeave={() => setIsDragging(false)}
          onDrop={handleDrop}
          onClick={() => setSelectedItems(new Set())}
        >
          {isDragging && (
            <div className="absolute inset-0 z-50 flex items-center justify-center bg-background/40 backdrop-blur-[2px]">
              <motion.div
                initial={{ scale: 0.9, opacity: 0 }}
                animate={{ scale: 1, opacity: 1 }}
                className="bg-background border-2 border-dashed border-primary/30 p-10 rounded-[2rem] shadow-2xl flex flex-col items-center gap-4"
              >
                <div className="w-16 h-16 bg-primary/10 rounded-full flex items-center justify-center text-primary">
                  <Upload className="w-8 h-8 animate-bounce" />
                </div>
                <div className="text-center">
                  <p className="text-lg font-bold">Ready to upload</p>
                  <p className="text-sm text-muted-foreground">Drop your files here to start</p>
                </div>
              </motion.div>
            </div>
          )}

          {openFile ? (
            <motion.div 
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              className="h-full p-6 max-w-[1600px] mx-auto w-full"
            >
              <FileViewer
                path={openFile.path}
                name={openFile.name}
                onClose={() => setOpenFile(null)}
              />
            </motion.div>
          ) : (
            <div className="h-full overflow-y-auto no-scrollbar scroll-smooth">
              <div className="p-6 max-w-[1600px] mx-auto w-full min-h-full flex flex-col">
                <AnimatePresence mode="wait">
                  {loading ? (
                    <motion.div
                      key="loading"
                      initial={{ opacity: 0 }}
                      animate={{ opacity: 1 }}
                      exit={{ opacity: 0 }}
                      className="flex-1 flex flex-col items-center justify-center gap-4"
                    >
                      <div className="relative">
                        <Loader2 className="w-10 h-10 animate-spin text-primary" />
                        <div className="absolute inset-0 blur-xl bg-primary/20 animate-pulse" />
                      </div>
                      <p className="text-sm font-medium text-muted-foreground animate-pulse">
                        Scanning filesystem...
                      </p>
                    </motion.div>
                  ) : error ? (
                    <motion.div
                      key="error"
                      initial={{ opacity: 0, scale: 0.95 }}
                      animate={{ opacity: 1, scale: 1 }}
                      exit={{ opacity: 0 }}
                      className="flex-1 flex flex-col items-center justify-center gap-6"
                    >
                      <div className="w-20 h-20 rounded-3xl bg-destructive/10 flex items-center justify-center shadow-inner">
                        <AlertCircle className="w-10 h-10 text-destructive" />
                      </div>
                      <div className="text-center space-y-2">
                        <h3 className="text-xl font-bold tracking-tight">Access Denied</h3>
                        <p className="text-sm text-muted-foreground max-w-xs mx-auto leading-relaxed">
                          {error}
                        </p>
                      </div>
                      <Button variant="secondary" className="rounded-xl px-8" onClick={refresh}>
                        Try Again
                      </Button>
                    </motion.div>
                  ) : filteredFiles.length === 0 ? (
                    <motion.div
                      key="empty"
                      initial={{ opacity: 0, scale: 0.95 }}
                      animate={{ opacity: 1, scale: 1 }}
                      exit={{ opacity: 0 }}
                      className="flex-1 flex flex-col items-center justify-center gap-6 opacity-60"
                    >
                      <div className="w-20 h-20 rounded-full bg-secondary/50 flex items-center justify-center">
                        <Search className="w-10 h-10 text-muted-foreground/40" />
                      </div>
                      <div className="text-center space-y-2">
                        <p className="text-lg font-bold text-muted-foreground">
                          {search ? "No matches found" : "Empty Space"}
                        </p>
                        <p className="text-sm text-muted-foreground/60 max-w-[200px] mx-auto">
                          {search
                            ? "Try refining your search terms"
                            : "Nothing here yet. Drag and drop to upload."}
                        </p>
                      </div>
                    </motion.div>
                  ) : (
                    <motion.div
                      key="content"
                      layout
                      className={cn(
                        "flex-1",
                        viewMode === "list" ? "space-y-1.5" : "file-grid-layout",
                      )}
                    >
                      {filteredFiles.map((file, index) => (
                        <FileItem
                          key={file.name}
                          file={file}
                          viewMode={viewMode}
                          isSelected={selectedItems.has(file.name)}
                          selectionMode={selectionMode}
                          onNavigate={handleNavigate}
                          onSelect={handleSelect}
                          onDelete={(name) =>
                            setDeleteConfirm({ open: true, names: [name] })
                          }
                          onRename={(name) =>
                            setRenameDialog({
                              open: true,
                              oldName: name,
                              newName: name,
                            })
                          }
                          onZip={(name) => {
                            zipItem(name);
                            toast({
                              title: "Archiving",
                              description: `Zipping ${name}...`,
                            });
                          }}
                          onExtract={(name) => {
                            extractItem(name);
                            toast({
                              title: "Extracting",
                              description: `Extracting ${name}...`,
                            });
                          }}
                        />
                      ))}
                    </motion.div>
                  )}
                </AnimatePresence>
              </div>
            </div>
          )}
        </main>
      </div>

      <Dialog
        open={newFolderDialog.open}
        onOpenChange={(open: boolean) =>
          setNewFolderDialog({ ...newFolderDialog, open })
        }
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>New Folder</DialogTitle>
            <DialogDescription>
              Create a new folder to organize your files.
            </DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <Input
              placeholder="Folder name"
              value={newFolderDialog.name}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                setNewFolderDialog({ ...newFolderDialog, name: e.target.value })
              }
              onKeyDown={(e: React.KeyboardEvent<HTMLInputElement>) => {
                if (e.key === "Enter" && newFolderDialog.name) {
                  createFolder(newFolderDialog.name);
                  setNewFolderDialog({ open: false, name: "" });
                  toast({ title: "Folder created" });
                }
              }}
              autoFocus
            />
          </div>
          <DialogFooter>
            <Button
              variant="ghost"
              onClick={() => setNewFolderDialog({ open: false, name: "" })}
            >
              Cancel
            </Button>
            <Button
              onClick={() => {
                createFolder(newFolderDialog.name);
                setNewFolderDialog({ open: false, name: "" });
                toast({ title: "Folder created" });
              }}
              disabled={!newFolderDialog.name}
            >
              Create
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={renameDialog.open}
        onOpenChange={(open: boolean) =>
          setRenameDialog({ ...renameDialog, open })
        }
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Rename</DialogTitle>
            <DialogDescription>
              Rename{" "}
              <span className="font-mono text-foreground">
                {renameDialog.oldName}
              </span>
            </DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <Input
              value={renameDialog.newName}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                setRenameDialog({ ...renameDialog, newName: e.target.value })
              }
              onKeyDown={(e: React.KeyboardEvent<HTMLInputElement>) => {
                if (e.key === "Enter" && renameDialog.newName) {
                  renameItem(renameDialog.oldName, renameDialog.newName);
                  setRenameDialog({ open: false, oldName: "", newName: "" });
                  toast({ title: "Renamed" });
                }
              }}
              autoFocus
            />
          </div>
          <DialogFooter>
            <Button
              variant="ghost"
              onClick={() =>
                setRenameDialog({ open: false, oldName: "", newName: "" })
              }
            >
              Cancel
            </Button>
            <Button
              onClick={() => {
                renameItem(renameDialog.oldName, renameDialog.newName);
                setRenameDialog({ open: false, oldName: "", newName: "" });
                toast({ title: "Renamed" });
              }}
              disabled={
                !renameDialog.newName ||
                renameDialog.newName === renameDialog.oldName
              }
            >
              Rename
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={deleteConfirm.open}
        onOpenChange={(open: boolean) =>
          setDeleteConfirm({ ...deleteConfirm, open })
        }
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-destructive flex items-center gap-2">
              <AlertCircle className="w-5 h-5" />
              Delete {deleteConfirm.names.length > 1 ? "Items" : "Item"}
            </DialogTitle>
            <DialogDescription className="pt-2">
              {deleteConfirm.names.length > 1 ? (
                <>
                  Delete{" "}
                  <span className="font-semibold">
                    {deleteConfirm.names.length} items
                  </span>
                  ? This cannot be undone.
                </>
              ) : (
                <>
                  Delete{" "}
                  <span className="font-semibold">
                    "{deleteConfirm.names[0]}"
                  </span>
                  ? This cannot be undone.
                </>
              )}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="ghost"
              onClick={() => setDeleteConfirm({ open: false, names: [] })}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={() => {
                if (deleteConfirm.names.length > 1) {
                  handleBulkDelete();
                } else {
                  deleteItem(deleteConfirm.names[0]);
                  toast({
                    title: "Deleted",
                    description: `${deleteConfirm.names[0]} removed.`,
                  });
                }
                setDeleteConfirm({ open: false, names: [] });
              }}
            >
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Toaster />
      <UploadProgress 
        uploads={activeUploads} 
        onClear={(id) => setActiveUploads(prev => prev.filter(u => u.id !== id))}
        onClearAll={() => setActiveUploads([])}
      />
    </div>
  );
}

export default App;
