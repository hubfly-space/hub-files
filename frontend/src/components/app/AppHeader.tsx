import React from "react";

import { Breadcrumb } from "../Breadcrumb";
import { Toolbar } from "../Toolbar";
import { StorageMeter } from "./StorageMeter";

import { FolderOpen } from "lucide-react";

type StorageInfo = {
  usedPercent: number;
  usedBytes: number;
  totalBytes: number;
};

type AppHeaderProps = {
  path: string;
  storage?: StorageInfo | null;
  viewMode: "list" | "grid";
  search: string;
  selectionMode: boolean;
  selectedCount: number;
  fileInputRef: React.RefObject<HTMLInputElement | null>;
  onNavigate: (path: string) => void;
  onViewToggle: () => void;
  onRefresh: () => void;
  onUpload: () => void;
  onNewFolder: () => void;
  onNewFile: () => void;
  onSearchChange: (value: string) => void;
  onSelectionModeToggle: () => void;
  onBulkDelete: () => void;
  onBulkZip: () => void;
  onClearSelection: () => void;
  onFileChange: (event: React.ChangeEvent<HTMLInputElement>) => void;
  canHostMount?: boolean;
  hostMounting?: boolean;
  onHostMount?: () => void;
  onHostUnmount?: () => void;
};

export function AppHeader({
  path,
  storage,
  viewMode,
  search,
  selectionMode,
  selectedCount,
  fileInputRef,
  onNavigate,
  onViewToggle,
  onRefresh,
  onUpload,
  onNewFolder,
  onNewFile,
  onSearchChange,
  onSelectionModeToggle,
  onBulkDelete,
  onBulkZip,
  onClearSelection,
  onFileChange,
  canHostMount,
  hostMounting,
  onHostMount,
  onHostUnmount,
}: AppHeaderProps) {
  return (
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
                  HubFiles
                </h1>
                <p className="text-[10px] uppercase tracking-widest text-muted-foreground mt-1 font-semibold">
                  Files Manager
                </p>
              </div>
            </div>

            <div className="h-6 w-px bg-border/60 mx-1 hidden md:block" />
            <Breadcrumb path={path} onNavigate={onNavigate} />
          </div>

          {storage && <StorageMeter storage={storage} />}
        </div>

        <Toolbar
          viewMode={viewMode}
          onViewToggle={onViewToggle}
          onRefresh={onRefresh}
          onUpload={onUpload}
          onNewFile={onNewFile}
          onNewFolder={onNewFolder}
          search={search}
          onSearchChange={onSearchChange}
          selectionMode={selectionMode}
          onSelectionModeToggle={onSelectionModeToggle}
          selectedCount={selectedCount}
          onBulkDelete={onBulkDelete}
          onBulkZip={onBulkZip}
          onClearSelection={onClearSelection}
          canHostMount={canHostMount}
          hostMounting={hostMounting}
          onHostMount={onHostMount}
          onHostUnmount={onHostUnmount}
        />
      </div>

      <input
        type="file"
        ref={fileInputRef}
        style={{ display: "none" }}
        onChange={onFileChange}
      />
    </header>
  );
}
