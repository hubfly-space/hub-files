import React, { useState, useEffect } from 'react';
import { api } from '../api';
import {
  ArrowLeft,
  Save,
  Edit3,
  Eye,
  FileText
} from 'lucide-react';
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { motion, AnimatePresence } from "framer-motion";
import { useToast } from "@/hooks/use-toast";

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
  const { toast } = useToast();

  const ext = name.split('.').pop()?.toLowerCase();
  const isImage = ['jpg', 'jpeg', 'png', 'gif', 'svg', 'webp'].includes(ext || '');

  useEffect(() => {
    if (!isImage) {
      api.getFile(path).then(data => {
        setContent(data);
        setEditedContent(data);
        setLoading(false);
      }).catch(err => {
        toast({
          title: "Error",
          description: err.message,
          variant: "destructive",
        });
        onClose();
      });
    } else {
      setLoading(false);
    }
  }, [path, isImage, onClose, toast]);

  const handleSave = async () => {
    try {
      await api.putFile(path, editedContent);
      setContent(editedContent);
      setIsEditing(false);
      toast({
        title: "Saved",
        description: `${name} updated.`,
      });
    } catch (err: any) {
      toast({
        title: "Save failed",
        description: err.message,
        variant: "destructive",
      });
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="flex flex-col items-center gap-3">
          <div className="w-8 h-8 border-2 border-foreground/20 border-t-foreground rounded-full animate-spin" />
          <span className="text-sm text-muted-foreground">Loading...</span>
        </div>
      </div>
    );
  }

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      className="flex flex-col h-full"
    >
      <div className="flex items-center justify-between mb-4 shrink-0">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon" onClick={onClose} className="rounded-lg">
            <ArrowLeft className="w-5 h-5" />
          </Button>
          <div>
            <h2 className="text-base font-semibold">{name}</h2>
            <p className="text-xs text-muted-foreground">{path}</p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {!isImage && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setIsEditing(!isEditing)}
            >
              {isEditing ? <Eye className="w-4 h-4 mr-2" /> : <Edit3 className="w-4 h-4 mr-2" />}
              {isEditing ? "Preview" : "Edit"}
            </Button>
          )}

          <AnimatePresence>
            {isEditing && (
              <motion.div
                initial={{ opacity: 0, x: 10 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: 10 }}
              >
                <Button size="sm" onClick={handleSave}>
                  <Save className="w-4 h-4 mr-2" />
                  Save
                </Button>
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </div>

      <div className="flex-1 min-h-0 rounded-xl border bg-secondary/30 overflow-hidden">
        {isImage ? (
          <div className="h-full flex items-center justify-center p-6">
            <img
              src={`http://localhost:8080/api/file?path=${encodeURIComponent(path)}&session=${api.getToken()}`}
              alt={name}
              className="max-w-full max-h-full rounded-lg object-contain"
            />
          </div>
        ) : (
          <ScrollArea className="h-full">
            <div className="p-6">
              {isEditing ? (
                <textarea
                  value={editedContent}
                  onChange={e => setEditedContent(e.target.value)}
                  className="w-full min-h-[500px] bg-transparent border-none text-sm font-mono leading-relaxed resize-none focus:outline-none"
                  spellCheck={false}
                  autoFocus
                />
              ) : (
                <pre className="text-sm font-mono leading-relaxed whitespace-pre-wrap">{content}</pre>
              )}
            </div>
          </ScrollArea>
        )}
      </div>
    </motion.div>
  );
};
