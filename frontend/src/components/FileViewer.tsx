import React, { useState, useEffect } from 'react';
import { api } from '../api';

interface FileViewerProps {
  path: string;
  name: string;
  onClose: () => void;
}

export const FileViewer: React.FC<FileViewerProps> = ({ path, name, onClose }) => {
  const [content, setContent] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [isEditing, setIsEditing] = useState(false);
  const [editedContent, setEditedContent] = useState('');

  const ext = name.split('.').pop()?.toLowerCase();
  const isImage = ['jpg', 'jpeg', 'png', 'gif', 'svg'].includes(ext || '');

  useEffect(() => {
    if (!isImage) {
      api.getFile(path).then(data => {
        setContent(data);
        setEditedContent(data);
        setLoading(false);
      }).catch(err => {
        alert(`Failed to load file: ${err.message}`);
        onClose();
      });
    } else {
      setLoading(false);
    }
  }, [path, isImage, onClose]);

  const handleSave = async () => {
    try {
      await api.putFile(path, editedContent);
      setContent(editedContent);
      setIsEditing(false);
    } catch (err: any) {
      alert(`Save failed: ${err.message}`);
    }
  };

  if (loading) return <div className="viewer-container">Loading...</div>;

  return (
    <div className="viewer-container">
      <div className="viewer-header">
        <button onClick={onClose}>← Back</button>
        <span className="file-title">{name}</span>
        {!isImage && (
          <div className="viewer-actions">
            {isEditing ? (
              <>
                <button onClick={handleSave}>Save</button>
                <button onClick={() => setIsEditing(false)}>Cancel</button>
              </>
            ) : (
              <button onClick={() => setIsEditing(true)}>Edit</button>
            )}
          </div>
        )}
      </div>
      
      <div className="viewer-content">
        {isImage ? (
          <img 
            src={`http://localhost:8080/api/file?path=${encodeURIComponent(path)}&session=${api.getToken()}`} 
            alt={name} 
          />
        ) : isEditing ? (
          <textarea 
            value={editedContent} 
            onChange={e => setEditedContent(e.target.value)}
          />
        ) : (
          <pre>{content}</pre>
        )}
      </div>
    </div>
  );
};
