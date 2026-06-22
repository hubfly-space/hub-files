import React, { useState } from "react";
import "./index.css";

import { useFileSystem } from "./hooks/useFileSystem";
import { useUploads } from "./hooks/useUploads";
import { useSelection } from "./hooks/useSelection";

import { AppHeader } from "./components/app/AppHeader";
import { FileBrowserContent } from "./components/app/FileBrowserContent";
import { FileDropOverlay } from "./components/app/FileDropOverlay";
import { DeleteConfirmDialog } from "./components/dialogs/DeleteConfirmDialog";
import { NewFolderDialog } from "./components/dialogs/NewFolderDialog";
import { RenameDialog } from "./components/dialogs/RenameDialog";
import { UploadProgress } from "./components/UploadProgress";

import { Toaster } from "@/components/ui/toaster";
import { useToast } from "@/hooks/use-toast";
import { cn } from "@/lib/utils";

import { api } from "./api";
import { joinPath } from "./utils/path";
import { NewFileDialog } from "./components/dialogs/NewFileDialog";

function App() {
  const {
    path,
    files,
    loading,
    error,
    storage,
    session,
    viewMode,
    setViewMode,
    navigate,
    refresh,
    deleteItem,
    renameItem,
    createFolder,
    createFile,
    zipItem,
    extractItem,
  } = useFileSystem();

  const { toast } = useToast();

  const [openFile, setOpenFile] = useState<{
    path: string;
    name: string;
  } | null>(null);

  const [search, setSearch] = useState("");
  const [isDragging, setIsDragging] = useState(false);
  const [hostMounting, setHostMounting] = useState(false);

  const [renameDialog, setRenameDialog] = useState<{
    open: boolean;
    oldName: string;
    newName: string;
  }>({ open: false, oldName: "", newName: "" });

  const [newFolderDialog, setNewFolderDialog] = useState<{
    open: boolean;
    name: string;
  }>({ open: false, name: "" });
  const [newFileDialog, setNewFileDialog] = useState<{
    open: boolean;
    name: string;
  }>({ open: false, name: "" });

  const [deleteConfirm, setDeleteConfirm] = useState<{
    open: boolean;
    names: string[];
  }>({ open: false, names: [] });

  const {
    selectedItems,
    selectionMode,
    handleSelect,
    clearSelection,
    toggleSelectionMode,
    setSelectedItems,
  } = useSelection();

  const {
    fileInputRef,
    activeUploads,
    handleUploadClick,
    handleFileChange,
    uploadFiles,
    clearUpload,
    clearAllUploads,
  } = useUploads(path, refresh);

  const filteredFiles = files.filter((file) =>
    file.name.toLowerCase().includes(search.toLowerCase()),
  );

  const handleDrop = async (event: React.DragEvent) => {
    event.preventDefault();
    setIsDragging(false);

    const droppedFiles = Array.from(event.dataTransfer.files);
    if (droppedFiles.length > 0) {
      await uploadFiles(droppedFiles);
    }
  };

  const handleNavigate = (name: string) => {
    const item = files.find((file) => file.name === name);
    const nextPath = joinPath(path, name);

    if (item?.isDir) {
      navigate(nextPath);
      clearSelection();
      return;
    }

    setOpenFile({ path: nextPath, name });
  };

  const handleBulkDelete = async () => {
    const names = Array.from(selectedItems);
    let deleted = 0;
    let failed = 0;

    for (const name of names) {
      try {
        await deleteItem(name);
        deleted++;
      } catch (err) {
        console.error(`Failed to delete ${name}:`, err);
        failed++;
      }
    }

    clearSelection();

    if (failed > 0) {
      toast({
        title: "Partial delete",
        description: `${deleted} deleted, ${failed} failed.`,
        variant: "destructive",
      });
      return;
    }

    toast({ title: "Deleted", description: `${deleted} items removed.` });
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
        console.error(`Failed to zip ${name}:`, err);
        failed++;
      }
    }

    clearSelection();

    if (failed > 0) {
      toast({
        title: "Partial archive",
        description: `${zipped} zipped, ${failed} failed.`,
        variant: "destructive",
      });
      return;
    }

    toast({ title: "Archived", description: `${zipped} items zipped.` });
  };

  const handleHostMount = async () => {
    setHostMounting(true);
    try {
      const result = await api.hostMount();
      toast({
        title: result.alreadyMounted ? "Already mounted" : "SMB mounted",
        description: `Available on this machine at ${result.mountPath}`,
      });
    } catch (err: unknown) {
      toast({
        title: "Mount failed",
        description: err instanceof Error ? err.message : "Unable to mount SMB share",
        variant: "destructive",
      });
    } finally {
      setHostMounting(false);
    }
  };

  const handleMove = async (sourceName: string, targetFolderName: string) => {
    if (sourceName === targetFolderName) return;

    const sourcePath = joinPath(path, sourceName);
    const targetPath = joinPath(joinPath(path, targetFolderName), sourceName);

    try {
      await api.rename(sourcePath, targetPath);
      toast({
        title: "Moved Successfully",
        description: `Moved ${sourceName} into ${targetFolderName}`,
      });
      refresh();
    } catch (err: unknown) {
      toast({
        title: "Move Failed",
        description: err instanceof Error ? err.message : "error moving",
        variant: "destructive",
      });
    }
  };

  return (
    <div className="fixed inset-0 bg-background flex flex-col selection:bg-primary/10">
      <div className="w-full h-full flex flex-col overflow-hidden">
        {!openFile && (
          <AppHeader
            path={path}
            storage={storage}
            viewMode={viewMode}
            onNavigate={navigate}
            onViewToggle={() =>
              setViewMode(viewMode === "list" ? "grid" : "list")
            }
            onRefresh={refresh}
            onUpload={handleUploadClick}
            onNewFolder={() => setNewFolderDialog({ open: true, name: "" })}
            onNewFile={() => setNewFileDialog({ open: true, name: "" })}
            search={search}
            onSearchChange={setSearch}
            selectionMode={selectionMode}
            onSelectionModeToggle={toggleSelectionMode}
            selectedCount={selectedItems.size}
            onBulkDelete={() =>
              setDeleteConfirm({
                open: true,
                names: Array.from(selectedItems),
              })
            }
            onBulkZip={handleBulkZip}
            onClearSelection={clearSelection}
            fileInputRef={fileInputRef}
            onFileChange={handleFileChange}
            canHostMount={session?.type === "smb" && session.canHostMount}
            hostMounting={hostMounting}
            onHostMount={handleHostMount}
          />
        )}

        <main
          className={cn(
            "flex-1 min-h-0 relative",
            isDragging && "bg-primary/5 ring-2 ring-primary/20 ring-inset",
          )}
          onDragOver={(event) => {
            event.preventDefault();
            setIsDragging(true);
          }}
          onDragLeave={() => setIsDragging(false)}
          onDrop={handleDrop}
          onClick={clearSelection}
        >
          {isDragging && <FileDropOverlay />}

          <FileBrowserContent
            openFile={openFile}
            files={filteredFiles}
            loading={loading}
            error={error}
            search={search}
            viewMode={viewMode}
            selectedItems={selectedItems}
            selectionMode={selectionMode}
            onCloseFile={() => setOpenFile(null)}
            onRefresh={refresh}
            onNavigate={handleNavigate}
            onSelect={handleSelect}
            onDelete={(name) => setDeleteConfirm({ open: true, names: [name] })}
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
            onMove={handleMove}
          />
        </main>
      </div>

      <NewFolderDialog
        open={newFolderDialog.open}
        name={newFolderDialog.name}
        onOpenChange={(open) =>
          setNewFolderDialog((current) => ({ ...current, open }))
        }
        onNameChange={(name) =>
          setNewFolderDialog((current) => ({ ...current, name }))
        }
        onCreate={(name) => {
          createFolder(name);
          setNewFolderDialog({ open: false, name: "" });
          toast({ title: "Folder created" });
        }}
      />
      <NewFileDialog
        open={newFileDialog.open}
        name={newFileDialog.name}
        onOpenChange={(open) =>
          setNewFileDialog((current) => ({ ...current, open }))
        }
        onNameChange={(name) =>
          setNewFileDialog((current) => ({ ...current, name }))
        }
        onCreate={(name) => {
          createFile(name);
          setNewFileDialog({ open: false, name: "" });
          toast({ title: "File created" });
        }}
      />

      <RenameDialog
        open={renameDialog.open}
        oldName={renameDialog.oldName}
        newName={renameDialog.newName}
        onOpenChange={(open) =>
          setRenameDialog((current) => ({ ...current, open }))
        }
        onNewNameChange={(newName) =>
          setRenameDialog((current) => ({ ...current, newName }))
        }
        onRename={(oldName, newName) => {
          renameItem(oldName, newName);
          setRenameDialog({ open: false, oldName: "", newName: "" });
          toast({ title: "Renamed" });
        }}
      />

      <DeleteConfirmDialog
        open={deleteConfirm.open}
        names={deleteConfirm.names}
        onOpenChange={(open) =>
          setDeleteConfirm((current) => ({ ...current, open }))
        }
        onCancel={() => setDeleteConfirm({ open: false, names: [] })}
        onDelete={() => {
          if (deleteConfirm.names.length > 1) {
            handleBulkDelete();
          } else {
            const name = deleteConfirm.names[0];
            deleteItem(name);
            toast({
              title: "Deleted",
              description: `${name} removed.`,
            });
            setSelectedItems((current) => {
              const next = new Set(current);
              next.delete(name);
              return next;
            });
          }

          setDeleteConfirm({ open: false, names: [] });
        }}
      />

      <Toaster />

      <UploadProgress
        uploads={activeUploads}
        onClear={clearUpload}
        onClearAll={clearAllUploads}
      />
    </div>
  );
}

export default App;
