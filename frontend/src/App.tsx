import React, { useRef, useState, useEffect, useCallback } from 'react';
import './index.css'
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
import { FolderPlus, FileWarning, HelpCircle, AlertCircle } from "lucide-react";
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

  // Multi-select state
  const [selectedItems, setSelectedItems] = useState<Set<string>>(new Set());
  const [selectionMode, setSelectionMode] = useState(false);

  // Dialog states
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
        toast({ title: "Upload successful", description: `${file.name} uploaded.` });
        refresh();
      } catch (err: any) {
        toast({ title: "Upload failed", description: err.message, variant: "destructive" });
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
        toast({ title: "Upload successful", description: `${file.name} uploaded via drop.` });
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
      setSelectedItems(newSet => { newSet.clear(); return new Set(); });
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
    toast({ title: "Bulk delete finished", description: `${names.length} items removed.` });
  };

  const handleBulkZip = async () => {
    toast({ title: "Bulk zip not yet optimized", description: "Zipping items individually..." });
    const names = Array.from(selectedItems);
    for (const name of names) {
      await zipItem(name);
    }
    setSelectedItems(new Set());
  };

  return (
    <div className="fixed inset-0 bg-background flex flex-col md:p-6 lg:p-10">
      <div className="w-full h-full bg-card rounded-none md:rounded-[2.5rem] shadow-2xl shadow-primary/10 border border-border/50 flex flex-col overflow-hidden relative backdrop-blur-md">
        
        {/* Decorative branding background */}
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_50%_0%,rgba(99,102,241,0.05),transparent)] pointer-events-none" />

        {!openFile && (
          <div className="shrink-0 flex flex-col relative z-20">
            <div className="flex items-center justify-between px-8 py-6 pb-2">
              <div className="flex items-center gap-4">
                <div className="w-10 h-10 bg-primary rounded-xl flex items-center justify-center text-primary-foreground font-black text-lg shadow-lg shadow-primary/30">
                  H-F
                </div>
                <h1 className="text-xl font-bold tracking-tight hidden sm:block">
                  HubFly <span className="text-primary/70 font-medium text-lg">Files</span>
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
          className={`flex-1 min-h-0 relative z-10 transition-colors duration-300 px-4 md:px-8 pb-8 ${isDragging ? 'bg-primary/5' : ''}`}
          onDragOver={(e) => { e.preventDefault(); setIsDragging(true); }}
          onDragLeave={() => setIsDragging(false)}
          onDrop={handleDrop}
          onClick={() => setSelectedItems(new Set())}
        >
          {openFile ? (
            <div className="h-full pt-8 animate-in fade-in zoom-in-95 duration-300">
              <FileViewer 
                path={openFile.path} 
                name={openFile.name} 
                onClose={() => setOpenFile(null)} 
              />
            </div>
          ) : (
            <AnimatePresence mode="wait">
              {loading ? (
                <motion.div 
                  initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
                  className="h-full flex flex-col items-center justify-center space-y-6"
                >
                  <div className="relative">
                    <div className="w-16 h-16 rounded-2xl border-4 border-primary/10 border-t-primary animate-spin" />
                    <div className="absolute inset-0 flex items-center justify-center">
                      <div className="w-2 h-2 rounded-full bg-primary animate-pulse" />
                    </div>
                  </div>
                  <span className="text-sm font-semibold text-muted-foreground animate-pulse tracking-widest uppercase">Indexing...</span>
                </motion.div>
              ) : error ? (
                <motion.div 
                  initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
                  className="h-full flex flex-col items-center justify-center space-y-4 p-8 text-center"
                >
                  <div className="p-5 rounded-3xl bg-destructive/10 text-destructive ring-8 ring-destructive/5">
                    <FileWarning className="w-10 h-10" />
                  </div>
                  <div className="space-y-1">
                    <h3 className="text-xl font-bold tracking-tight">Access Denied</h3>
                    <p className="text-sm text-muted-foreground max-w-xs mx-auto">{error}</p>
                  </div>
                  <Button variant="outline" size="sm" onClick={refresh} className="rounded-full px-6">Try Again</Button>
                </motion.div>
              ) : filteredFiles.length === 0 ? (
                <motion.div 
                  initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
                  className="h-full flex flex-col items-center justify-center space-y-4"
                >
                  <div className="p-6 rounded-3xl bg-muted/30 text-muted-foreground/40">
                    <HelpCircle className="w-12 h-12" />
                  </div>
                  <div className="text-center">
                    <p className="text-sm font-bold text-muted-foreground/60 tracking-tight">
                      {search ? "Zero matches found" : "Quiet in here..."}
                    </p>
                    <p className="text-xs text-muted-foreground/40 mt-1">
                      {search ? "Try a different search term" : "Upload something to get started"}
                    </p>
                  </div>
                </motion.div>
              ) : (
                <motion.div 
                  layout
                  className={cn(
                    "h-full overflow-y-auto no-scrollbar pt-6 pb-20",
                    viewMode === 'list' ? 'flex flex-col space-y-1' : 'file-grid-layout'
                  )}
                >
                  <LayoutGroup>
                    {filteredFiles.map(file => (
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
                          toast({ title: "Archiving started", description: `Zipping ${name}...` });
                        }}
                        onExtract={(name) => {
                          extractItem(name);
                          toast({ title: "Extraction started", description: `Extracting ${name}...` });
                        }}
                      />
                    ))}
                  </LayoutGroup>
                </motion.div>
              )}
            </AnimatePresence>
          )}
        </div>
      </div>

      {/* Dialogs - Portal managed by Shadcn, fixed z-index via CSS */}
      <Dialog open={newFolderDialog.open} onOpenChange={(open: boolean) => setNewFolderDialog({ ...newFolderDialog, open })}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-3 text-xl font-bold">
              <div className="p-2 bg-primary/10 rounded-lg text-primary">
                <FolderPlus className="w-5 h-5" />
              </div>
              Create New Folder
            </DialogTitle>
            <DialogDescription>Directories help you stay organized. Name it something clear.</DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <Input 
              placeholder="Folder name (e.g. Assets, Research)" 
              value={newFolderDialog.name}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setNewFolderDialog({ ...newFolderDialog, name: e.target.value })}
              onKeyDown={(e: React.KeyboardEvent<HTMLInputElement>) => {
                if (e.key === 'Enter' && newFolderDialog.name) {
                  createFolder(newFolderDialog.name);
                  setNewFolderDialog({ open: false, name: '' });
                  toast({ title: "Folder created" });
                }
              }}
              className="h-11 rounded-xl"
              autoFocus
            />
          </div>
          <DialogFooter className="gap-2">
            <Button variant="ghost" onClick={() => setNewFolderDialog({ open: false, name: '' })} className="rounded-xl">Cancel</Button>
            <Button 
              disabled={!newFolderDialog.name}
              onClick={() => {
                createFolder(newFolderDialog.name);
                setNewFolderDialog({ open: false, name: '' });
                toast({ title: "Folder created" });
              }} 
              className="rounded-xl px-6"
            >
              Create Folder
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={renameDialog.open} onOpenChange={(open: boolean) => setRenameDialog({ ...renameDialog, open })}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-xl font-bold">Rename Item</DialogTitle>
            <DialogDescription>This will change the name of <span className="font-mono text-primary bg-primary/5 px-1.5 py-0.5 rounded">{renameDialog.oldName}</span>.</DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <Input 
              value={renameDialog.newName}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setRenameDialog({ ...renameDialog, newName: e.target.value })}
              onKeyDown={(e: React.KeyboardEvent<HTMLInputElement>) => {
                if (e.key === 'Enter' && renameDialog.newName) {
                  renameItem(renameDialog.oldName, renameDialog.newName);
                  setRenameDialog({ open: false, oldName: '', newName: '' });
                  toast({ title: "Item renamed" });
                }
              }}
              className="h-11 rounded-xl"
              autoFocus
            />
          </div>
          <DialogFooter className="gap-2">
            <Button variant="ghost" onClick={() => setRenameDialog({ open: false, oldName: '', newName: '' })} className="rounded-xl">Cancel</Button>
            <Button 
              disabled={!renameDialog.newName || renameDialog.newName === renameDialog.oldName}
              onClick={() => {
                renameItem(renameDialog.oldName, renameDialog.newName);
                setRenameDialog({ open: false, oldName: '', newName: '' });
                toast({ title: "Item renamed" });
              }}
              className="rounded-xl px-6"
            >
              Rename Item
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={deleteConfirm.open} onOpenChange={(open: boolean) => setDeleteConfirm({ ...deleteConfirm, open })}>
        <DialogContent className="sm:max-w-md border-destructive/20">
          <DialogHeader>
            <DialogTitle className="text-destructive flex items-center gap-3 text-xl font-bold">
              <AlertCircle className="w-6 h-6" />
              Delete Permanently?
            </DialogTitle>
            <DialogDescription className="text-base pt-2">
              {deleteConfirm.names.length > 1 ? (
                <>You are about to delete <span className="font-bold text-foreground">{deleteConfirm.names.length} items</span>. This cannot be undone.</>
              ) : (
                <>Are you sure you want to delete <span className="font-bold text-foreground">"{deleteConfirm.names[0]}"</span>? It will be gone forever.</>
              )}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="mt-4 gap-2">
            <Button variant="ghost" onClick={() => setDeleteConfirm({ open: false, names: [] })} className="rounded-xl">Keep Items</Button>
            <Button variant="destructive" onClick={() => {
              if (deleteConfirm.names.length > 1) {
                handleBulkDelete();
              } else {
                deleteItem(deleteConfirm.names[0]);
                toast({ title: "Item deleted", description: `${deleteConfirm.names[0]} removed.` });
              }
              setDeleteConfirm({ open: false, names: [] });
            }} className="rounded-xl px-6 shadow-lg shadow-destructive/20">
              Delete Forever
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Toaster />
    </div>
  )
}

export default App
