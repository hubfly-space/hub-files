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
          title: "Error loading file",
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
        title: "File saved",
        description: `${name} has been updated.`,
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
      <div className="flex flex-col items-center justify-center h-full space-y-4 animate-pulse">
        <FileText className="w-12 h-12 text-muted-foreground/50" />
        <span className="text-sm text-muted-foreground">Loading file...</span>
      </div>
    );
  }

  return (
    <motion.div 
      initial={{ opacity: 0, scale: 0.98 }}
      animate={{ opacity: 1, scale: 1 }}
      className="flex flex-col h-full overflow-hidden"
    >
      <div className="flex items-center justify-between gap-4 mb-6 shrink-0">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={onClose} className="rounded-full">
            <ArrowLeft className="w-5 h-5" />
          </Button>
          <div>
            <h2 className="text-lg font-bold tracking-tight">{name}</h2>
            <p className="text-xs text-muted-foreground">{path}</p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {!isImage && (
            <Button 
              variant="outline" 
              size="sm" 
              onClick={() => setIsEditing(!isEditing)}
              className="gap-2"
            >
              {isEditing ? <Eye className="w-4 h-4" /> : <Edit3 className="w-4 h-4" />}
              {isEditing ? "Preview" : "Edit"}
            </Button>
          )}
          
          <AnimatePresence>
            {isEditing && (
              <motion.div
                initial={{ opacity: 0, x: 20 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: 20 }}
              >
                <Button size="sm" onClick={handleSave} className="gap-2 shadow-lg shadow-primary/20">
                  <Save className="w-4 h-4" />
                  Save Changes
                </Button>
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </div>
      
      <div className="flex-1 min-h-0 rounded-2xl border bg-card/50 backdrop-blur-sm overflow-hidden flex flex-col">
        {isImage ? (
          <div className="flex-1 flex items-center justify-center p-8 overflow-auto">
            <img 
              src={`http://localhost:8080/api/file?path=${encodeURIComponent(path)}&session=${api.getToken()}`} 
              alt={name} 
              className="max-w-full max-h-full rounded-lg shadow-2xl object-contain animate-in fade-in zoom-in duration-500"
            />
          </div>
        ) : (
          <ScrollArea className="flex-1">
            <div className="p-8">
              {isEditing ? (
                <textarea 
                  value={editedContent} 
                  onChange={e => setEditedContent(e.target.value)}
                  className="w-full min-h-[500px] bg-transparent border-none text-sm font-mono leading-relaxed resize-none focus:outline-none focus:ring-0 selection:bg-primary/20"
                  spellCheck={false}
                  autoFocus
                />
              ) : (
                <pre className="text-sm font-mono leading-relaxed whitespace-pre-wrap selection:bg-primary/20">
                  {content}
                </pre>
              )}
            </div>
          </ScrollArea>
        )}
      </div>
    </motion.div>
  );
};
