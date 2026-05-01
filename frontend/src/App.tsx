import React, { useRef, useState, useEffect, useCallback } from 'react';
import './index.css';
import { useFileSystem } from './hooks/useFileSystem';
import { Breadcrumb } from './components/Breadcrumb';
import { Toolbar } from './components/Toolbar';
import { FileItem } from './components/FileItem';
import { FileViewer } from './components/FileViewer';
import { api } from './api';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Toaster } from "@/components/ui/toaster";
import { useToast } from "@/hooks/use-toast";
import { motion, AnimatePresence, LayoutGroup } from "framer-motion";
import { FolderPlus, FileWarning, HelpCircle, AlertCircle, Loader2 } from "lucide-react";
import { cn } from '@/lib/utils';

function App() {
  const {
    path, files, loading, error, viewMode, setViewMode,
    navigate, refresh, deleteItem, renameItem, createFolder,
    zipItem, extractItem
  } = useFileSystem();

  const [openFile, setOpenFile] = useState<{ path: string, name: string } | null>(null);
  const [search, setSearch] = useState('');
  const [isDragging, setIsDragging] = useState(false);
  const { toast } = useToast();

  const [selectedItems, setSelectedItems] = useState<Set<string>>(new Set());
  const [selectionMode, setSelectionMode] = useState(false);

  const [renameDialog, setRenameDialog] = useState<{ open: boolean, oldName: string, newName: string }>({ open: false, oldName: '', newName: '' });
  const [newFolderDialog, setNewFolderDialog] = useState<{ open: boolean, name: string }>({ open: false, name: '' });
  const [deleteConfirm, setDeleteConfirm] = useState<{ open: boolean, names: string[] }>({ open: false, names: [] });

  const filteredFiles = files.filter(f =>
    f.name.toLowerCase().includes(search.toLowerCase())
  );

  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleUploadClick = () => fileInputRef.current?.click();

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      try {
        await api.upload(path, file);
        toast({ title: "Uploaded", description: `${file.name} uploaded successfully.` });
        refresh();
      } catch (err: any) {
        toast({ title: "Upload failed", description: err.message, variant: "destructive" });
      } finally {
        if (e.target) e.target.value = '';
      }
    }
  };

  const handleDrop = async (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
    const file = e.dataTransfer.files[0];
    if (file) {
      try {
        await api.upload(path, file);
        toast({ title: "Uploaded", description: `${file.name} uploaded via drop.` });
        refresh();
      } catch (err: any) {
        toast({ title: "Upload failed", description: err.message, variant: "destructive" });
      }
    }
  };

  const handleNavigate = (name: string) => {
    const item = files.find(f => f.name === name);
    if (item?.isDir) {
      const newPath = path === '/' ? `/${name}` : `${path}/${name}`;
      navigate(newPath);
      setSelectedItems(new Set());
    } else {
      const filePath = path === '/' ? `/${name}` : `${path}/${name}`;
      setOpenFile({ path: filePath, name });
    }
  };

  const handleSelect = useCallback((name: string, multi: boolean) => {
    setSelectedItems(prev => {
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
    for (const name of names) {
      await deleteItem(name);
    }
    setSelectedItems(new Set());
    toast({ title: "Deleted", description: `${names.length} items removed.` });
  };

  const handleBulkZip = async () => {
    const names = Array.from(selectedItems);
    for (const name of names) {
      await zipItem(name);
    }
    setSelectedItems(new Set());
    toast({ title: "Archived", description: `${names.length} items zipped.` });
  };

  return (
    <div className="fixed inset-0 bg-background flex flex-col">
      <div className="w-full h-full flex flex-col overflow-hidden bg-background">
        {!openFile && (
          <div className="shrink-0 border-b border-border/50 bg-background/80 backdrop-blur-xl sticky top-0 z-30">
            <div className="flex items-center justify-between px-6 py-4">
              <div className="flex items-center gap-3">
                <div className="w-8 h-8 bg-foreground rounded-lg flex items-center justify-center text-background font-bold text-sm">
                  H
                </div>
                <h1 className="text-lg font-semibold tracking-tight">
                  HubFly <span className="text-muted-foreground font-normal">Files</span>
                </h1>
              </div>
              <Breadcrumb path={path} onNavigate={navigate} />
            </div>

            <Toolbar
              viewMode={viewMode}
              onViewToggle={() => setViewMode(viewMode === 'list' ? 'grid' : 'list')}
              onRefresh={refresh}
              onUpload={handleUploadClick}
              onNewFolder={() => setNewFolderDialog({ open: true, name: '' })}
              search={search}
              onSearchChange={setSearch}
              selectionMode={selectionMode}
              onSelectionModeToggle={() => {
                setSelectionMode(!selectionMode);
                if (selectionMode) setSelectedItems(new Set());
              }}
              selectedCount={selectedItems.size}
              onBulkDelete={() => setDeleteConfirm({ open: true, names: Array.from(selectedItems) })}
              onBulkZip={handleBulkZip}
              onClearSelection={() => setSelectedItems(new Set())}
            />
            <input type="file" ref={fileInputRef} style={{ display: 'none' }} onChange={handleFileChange} />
          </div>
        )}

        <div
          className={cn(
            "flex-1 min-h-0 relative",
            isDragging && "bg-secondary/50"
          )}
          onDragOver={(e) => { e.preventDefault(); setIsDragging(true); }}
          onDragLeave={() => setIsDragging(false)}
          onDrop={handleDrop}
          onClick={() => setSelectedItems(new Set())}
        >
          {isDragging && (
            <div className="absolute inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm">
              <motion.div
                initial={{ scale: 0.9, opacity: 0 }}
                animate={{ scale: 1, opacity: 1 }}
                className="text-center space-y-2"
              >
                <div className="w-16 h-16 border-2 border-dashed border-muted-foreground/30 rounded-2xl flex items-center justify-center mx-auto">
                  <Loader2 className="w-8 h-8 text-muted-foreground animate-spin" />
                </div>
                <p className="text-sm text-muted-foreground">Drop file to upload</p>
              </motion.div>
            </div>
          )}

          {openFile ? (
            <div className="h-full p-6">
              <FileViewer
                path={openFile.path}
                name={openFile.name}
                onClose={() => setOpenFile(null)}
              />
            </div>
          ) : (
            <div className="h-full overflow-y-auto no-scrollbar p-6">
              <AnimatePresence mode="wait">
                {loading ? (
                  <motion.div
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    exit={{ opacity: 0 }}
                    className="h-full flex items-center justify-center"
                  >
                    <div className="flex flex-col items-center gap-4">
                      <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
                      <span className="text-sm text-muted-foreground">Loading...</span>
                    </div>
                  </motion.div>
                ) : error ? (
                  <motion.div
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    exit={{ opacity: 0 }}
                    className="h-full flex flex-col items-center justify-center gap-4"
                  >
                    <div className="w-16 h-16 rounded-2xl bg-destructive/10 flex items-center justify-center">
                      <AlertCircle className="w-8 h-8 text-destructive" />
                    </div>
                    <div className="text-center space-y-1">
                      <h3 className="text-lg font-semibold">Access Denied</h3>
                      <p className="text-sm text-muted-foreground max-w-sm">{error}</p>
                    </div>
                    <Button variant="outline" size="sm" onClick={refresh}>Try Again</Button>
                  </motion.div>
                ) : filteredFiles.length === 0 ? (
                  <motion.div
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    exit={{ opacity: 0 }}
                    className="h-full flex flex-col items-center justify-center gap-4"
                  >
                    <HelpCircle className="w-12 h-12 text-muted-foreground/30" />
                    <div className="text-center">
                      <p className="text-sm font-medium text-muted-foreground">
                        {search ? "No matches found" : "This folder is empty"}
                      </p>
                      <p className="text-xs text-muted-foreground/60 mt-1">
                        {search ? "Try a different search" : "Upload a file to get started"}
                      </p>
                    </div>
                  </motion.div>
                ) : (
                  <div className={cn(
                    viewMode === 'list' ? 'space-y-1' : 'file-grid-layout'
                  )}>
                    {filteredFiles.map((file) => (
                      <FileItem
                        key={file.name}
                        file={file}
                        viewMode={viewMode}
                        isSelected={selectedItems.has(file.name)}
                        selectionMode={selectionMode}
                        onNavigate={handleNavigate}
                        onSelect={handleSelect}
                        onDelete={(name) => setDeleteConfirm({ open: true, names: [name] })}
                        onRename={(name) => setRenameDialog({ open: true, oldName: name, newName: name })}
                        onZip={(name) => {
                          zipItem(name);
                          toast({ title: "Archiving", description: `Zipping ${name}...` });
                        }}
                        onExtract={(name) => {
                          extractItem(name);
                          toast({ title: "Extracting", description: `Extracting ${name}...` });
                        }}
                      />
                    ))}
                  </div>
                )}
              </AnimatePresence>
            </div>
          )}
        </div>
      </div>

      <Dialog open={newFolderDialog.open} onOpenChange={(open: boolean) => setNewFolderDialog({ ...newFolderDialog, open })}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>New Folder</DialogTitle>
            <DialogDescription>Create a new folder to organize your files.</DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <Input
              placeholder="Folder name"
              value={newFolderDialog.name}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setNewFolderDialog({ ...newFolderDialog, name: e.target.value })}
              onKeyDown={(e: React.KeyboardEvent<HTMLInputElement>) => {
                if (e.key === 'Enter' && newFolderDialog.name) {
                  createFolder(newFolderDialog.name);
                  setNewFolderDialog({ open: false, name: '' });
                  toast({ title: "Folder created" });
                }
              }}
              autoFocus
            />
          </div>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setNewFolderDialog({ open: false, name: '' })}>Cancel</Button>
            <Button onClick={() => {
              createFolder(newFolderDialog.name);
              setNewFolderDialog({ open: false, name: '' });
              toast({ title: "Folder created" });
            }} disabled={!newFolderDialog.name}>Create</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={renameDialog.open} onOpenChange={(open: boolean) => setRenameDialog({ ...renameDialog, open })}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Rename</DialogTitle>
            <DialogDescription>
              Rename <span className="font-mono text-foreground">{renameDialog.oldName}</span>
            </DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <Input
              value={renameDialog.newName}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setRenameDialog({ ...renameDialog, newName: e.target.value })}
              onKeyDown={(e: React.KeyboardEvent<HTMLInputElement>) => {
                if (e.key === 'Enter' && renameDialog.newName) {
                  renameItem(renameDialog.oldName, renameDialog.newName);
                  setRenameDialog({ open: false, oldName: '', newName: '' });
                  toast({ title: "Renamed" });
                }
              }}
              autoFocus
            />
          </div>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setRenameDialog({ open: false, oldName: '', newName: '' })}>Cancel</Button>
            <Button
              onClick={() => {
                renameItem(renameDialog.oldName, renameDialog.newName);
                setRenameDialog({ open: false, oldName: '', newName: '' });
                toast({ title: "Renamed" });
              }}
              disabled={!renameDialog.newName || renameDialog.newName === renameDialog.oldName}
            >Rename</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={deleteConfirm.open} onOpenChange={(open: boolean) => setDeleteConfirm({ ...deleteConfirm, open })}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-destructive flex items-center gap-2">
              <AlertCircle className="w-5 h-5" />
              Delete {deleteConfirm.names.length > 1 ? 'Items' : 'Item'}
            </DialogTitle>
            <DialogDescription className="pt-2">
              {deleteConfirm.names.length > 1 ? (
                <>Delete <span className="font-semibold">{deleteConfirm.names.length} items</span>? This cannot be undone.</>
              ) : (
                <>Delete <span className="font-semibold">"{deleteConfirm.names[0]}"</span>? This cannot be undone.</>
              )}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setDeleteConfirm({ open: false, names: [] })}>Cancel</Button>
            <Button variant="destructive" onClick={() => {
              if (deleteConfirm.names.length > 1) {
                handleBulkDelete();
              } else {
                deleteItem(deleteConfirm.names[0]);
                toast({ title: "Deleted", description: `${deleteConfirm.names[0]} removed.` });
              }
              setDeleteConfirm({ open: false, names: [] });
            }}>Delete</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Toaster />
    </div>
  );
}

export default App;
