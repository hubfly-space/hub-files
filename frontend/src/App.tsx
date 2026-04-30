import { useRef, useState } from 'react';
import './index.css'
import './App.css'
import { useFileSystem } from './hooks/useFileSystem';
import { Breadcrumb } from './components/Breadcrumb';
import { Toolbar } from './components/Toolbar';
import { FileItem } from './components/FileItem';
import { FileViewer } from './components/FileViewer';
import { api } from './api';

function App() {
  const {
    path, files, loading, error, viewMode, setViewMode,
    navigate, refresh, deleteItem, renameItem, createFolder,
    zipItem, extractItem
  } = useFileSystem();

  const [openFile, setOpenFile] = useState<{ path: string, name: string } | null>(null);
  const [search, setSearch] = useState('');
  const [isDragging, setIsDragging] = useState(false);

  const filteredFiles = files.filter(f => 
    f.name.toLowerCase().includes(search.toLowerCase())
  );

  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleUploadClick = () => {
    fileInputRef.current?.click();
  };

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      try {
        await api.upload(path, file);
        refresh();
      } catch (err: any) {
        alert(`Upload failed: ${err.message}`);
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
        refresh();
      } catch (err: any) {
        alert(`Upload failed: ${err.message}`);
      }
    }
  };

  const handleNewFolder = () => {
    const name = prompt('Folder name:');
    if (name) createFolder(name);
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
    <div className="main-container">
      {!openFile && (
        <div className="header">
          <Breadcrumb path={path} onNavigate={navigate} />
          <Toolbar 
            viewMode={viewMode}
            onViewToggle={() => setViewMode(viewMode === 'list' ? 'grid' : 'list')}
            onRefresh={refresh}
            onUpload={handleUploadClick}
            onNewFolder={handleNewFolder}
          />
          <input 
            type="file" 
            ref={fileInputRef} 
            style={{ display: 'none' }} 
            onChange={handleFileChange}
          />
        </div>
      )}
      
      <div 
        className={`content ${isDragging ? 'dragging' : ''}`}
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
          <>
            {loading && <div className="empty-state">Loading...</div>}
            {error && <div className="empty-state" style={{ color: 'red' }}>Error: {error}</div>}
            
            {!loading && !error && files.length === 0 && (
              <div className="empty-state">This folder is empty</div>
            )}

            {!loading && !error && files.length > 0 && (
              <>
                <div className="search-bar">
                  <input 
                    placeholder="Search in folder..." 
                    value={search}
                    onChange={e => setSearch(e.target.value)}
                  />
                </div>
                <div className={viewMode === 'list' ? 'file-list' : 'file-grid'}>
                  {filteredFiles.map(file => (
                    <FileItem 
                      key={file.name}
                      file={file}
                      viewMode={viewMode}
                      onNavigate={handleNavigate}
                      onDelete={deleteItem}
                      onRename={renameItem}
                      onZip={zipItem}
                      onExtract={extractItem}
                    />
                  ))}
                </div>
              </>
            )}
          </>
        )}
      </div>
    </div>
  )
}

export default App
