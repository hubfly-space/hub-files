import React, { useRef, useState } from 'react';
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
import { motion, AnimatePresence } from "framer-motion";
import { FolderPlus, FileWarning, HelpCircle } from "lucide-react";

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

  // Dialog states
  const [renameDialog, setRenameDialog] = useState<{ open: boolean, oldName: string, newName: string }>({ open: false, oldName: '', newName: '' });
  const [newFolderDialog, setNewFolderDialog] = useState<{ open: boolean, name: string }>({ open: false, name: '' });
  const [deleteConfirm, setDeleteConfirm] = useState<{ open: boolean, name: string }>({ open: false, name: '' });

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
    } else {
      const filePath = path === '/' ? `/${name}` : `${path}/${name}`;
      setOpenFile({ path: filePath, name });
    }
  };

  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-4 md:p-8">
      <div className="w-full max-w-6xl h-[85vh] bg-card rounded-[2rem] shadow-2xl shadow-primary/5 border border-border/50 flex flex-col overflow-hidden relative backdrop-blur-sm">
        
        {/* Background Decorative Gradient */}
        <div className="absolute inset-0 bg-gradient-to-br from-primary/5 via-transparent to-accent/5 pointer-events-none" />

        {!openFile && (
          <div className="shrink-0 flex flex-col px-8 pt-8 pb-4 relative z-10">
            <div className="flex items-center justify-between mb-2">
              <h1 className="text-2xl font-black tracking-tight flex items-center gap-2">
                HubFly <span className="text-primary">Files</span>
              </h1>
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
            />
            <input type="file" ref={fileInputRef} style={{ display: 'none' }} onChange={handleFileChange} />
          </div>
        )}
        
        <div 
          className={`flex-1 min-h-0 px-8 pb-8 relative z-10 transition-colors duration-300 ${isDragging ? 'bg-primary/5' : ''}`}
          onDragOver={(e) => { e.preventDefault(); setIsDragging(true); }}
          onDragLeave={() => setIsDragging(false)}
          onDrop={handleDrop}
        >
          {openFile ? (
            <FileViewer 
              path={openFile.path} 
              name={openFile.name} 
              onClose={() => setOpenFile(null)} 
            />
          ) : (
            <AnimatePresence mode="wait">
              {loading ? (
                <motion.div 
                  initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
                  className="h-full flex flex-col items-center justify-center space-y-4"
                >
                  <div className="w-12 h-12 rounded-full border-4 border-primary/20 border-t-primary animate-spin" />
                  <span className="text-sm font-medium text-muted-foreground">Indexing files...</span>
                </motion.div>
              ) : error ? (
                <motion.div 
                  initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
                  className="h-full flex flex-col items-center justify-center space-y-4 p-8 text-center"
                >
                  <div className="p-4 rounded-full bg-destructive/10 text-destructive">
                    <FileWarning className="w-8 h-8" />
                  </div>
                  <div>
                    <h3 className="text-lg font-bold">Failed to load directory</h3>
                    <p className="text-sm text-muted-foreground max-w-xs">{error}</p>
                  </div>
                  <Button variant="outline" onClick={refresh}>Try Again</Button>
                </motion.div>
              ) : filteredFiles.length === 0 ? (
                <motion.div 
                  initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
                  className="h-full flex flex-col items-center justify-center space-y-4"
                >
                  <div className="p-4 rounded-full bg-muted/50 text-muted-foreground">
                    <HelpCircle className="w-8 h-8" />
                  </div>
                  <span className="text-sm font-medium text-muted-foreground">
                    {search ? "No files match your search" : "This folder is empty"}
                  </span>
                </motion.div>
              ) : (
                <motion.div 
                  initial={{ opacity: 0, y: 10 }} 
                  animate={{ opacity: 1, y: 0 }}
                  className={`h-full overflow-y-auto no-scrollbar pt-2 ${viewMode === 'list' ? 'flex flex-col gap-1' : 'grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4'}`}
                >
                  {filteredFiles.map(file => (
                    <FileItem 
                      key={file.name}
                      file={file}
                      viewMode={viewMode}
                      onNavigate={handleNavigate}
                      onDelete={(name) => setDeleteConfirm({ open: true, name })}
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
                </motion.div>
              )}
            </AnimatePresence>
          )}
        </div>
      </div>

      {/* Dialogs */}
      <Dialog open={newFolderDialog.open} onOpenChange={(open: boolean) => setNewFolderDialog({ ...newFolderDialog, open })}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <FolderPlus className="w-5 h-5 text-primary" />
              New Folder
            </DialogTitle>
            <DialogDescription>Create a new directory in the current path.</DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <Input 
              placeholder="Enter folder name" 
              value={newFolderDialog.name}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setNewFolderDialog({ ...newFolderDialog, name: e.target.value })}
              onKeyDown={(e: React.KeyboardEvent<HTMLInputElement>) => {
                if (e.key === 'Enter') {
                  createFolder(newFolderDialog.name);
                  setNewFolderDialog({ open: false, name: '' });
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
            }}>Create Folder</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={renameDialog.open} onOpenChange={(open: boolean) => setRenameDialog({ ...renameDialog, open })}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Rename Item</DialogTitle>
            <DialogDescription>Enter a new name for your file or folder.</DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <Input 
              value={renameDialog.newName}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setRenameDialog({ ...renameDialog, newName: e.target.value })}
              onKeyDown={(e: React.KeyboardEvent<HTMLInputElement>) => {
                if (e.key === 'Enter') {
                  renameItem(renameDialog.oldName, renameDialog.newName);
                  setRenameDialog({ open: false, oldName: '', newName: '' });
                }
              }}
              autoFocus
            />
          </div>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setRenameDialog({ open: false, oldName: '', newName: '' })}>Cancel</Button>
            <Button onClick={() => {
              renameItem(renameDialog.oldName, renameDialog.newName);
              setRenameDialog({ open: false, oldName: '', newName: '' });
            }}>Rename</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={deleteConfirm.open} onOpenChange={(open: boolean) => setDeleteConfirm({ ...deleteConfirm, open })}>
        <DialogContent className="border-destructive/20">
          <DialogHeader>
            <DialogTitle className="text-destructive">Confirm Deletion</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete <span className="font-bold text-foreground">"{deleteConfirm.name}"</span>? 
              This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setDeleteConfirm({ open: false, name: '' })}>Cancel</Button>
            <Button variant="destructive" onClick={() => {
              deleteItem(deleteConfirm.name);
              setDeleteConfirm({ open: false, name: '' });
              toast({ title: "Item deleted", description: `${deleteConfirm.name} has been removed.` });
            }}>Delete Forever</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Toaster />
    </div>
  )
}

export default App
