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
    <div className="flex flex-col h-full bg-card rounded-3xl border border-border/50 shadow-2xl overflow-hidden">
      <div className="flex items-center justify-between p-4 border-b border-border/40 bg-secondary/20 shrink-0">
        <div className="flex items-center gap-4">
          <Button variant="secondary" size="icon" onClick={onClose} className="rounded-xl h-10 w-10 bg-background hover:bg-primary hover:text-primary-foreground transition-all shadow-sm">
            <ArrowLeft className="w-5 h-5" />
          </Button>
          <div className="flex flex-col">
            <h2 className="text-sm font-bold tracking-tight flex items-center gap-2">
              <FileText className="w-4 h-4 text-primary" />
              {name}
            </h2>
            <p className="text-[10px] text-muted-foreground font-medium uppercase tracking-wider">{path}</p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {!isImage && (
            <Button
              variant="secondary"
              size="sm"
              onClick={() => setIsEditing(!isEditing)}
              className="rounded-lg px-4 h-9 font-semibold"
            >
              {isEditing ? <Eye className="w-4 h-4 mr-2" /> : <Edit3 className="w-4 h-4 mr-2" />}
              {isEditing ? "Preview" : "Edit"}
            </Button>
          )}

          {isEditing && (
            <Button size="sm" onClick={handleSave} className="rounded-lg px-4 h-9 font-semibold shadow-lg shadow-primary/20">
              <Save className="w-4 h-4 mr-2" />
              Save Changes
            </Button>
          )}
        </div>
      </div>

      <div className="flex-1 min-h-0 bg-background relative">
        {isImage ? (
          <div className="h-full flex items-center justify-center p-12 bg-[radial-gradient(#e5e7eb_1px,transparent_1px)] [background-size:20px_20px] dark:bg-[radial-gradient(#1f2937_1px,transparent_1px)]">
            <img
              src={`/api/file?path=${encodeURIComponent(path)}`}
              alt={name}
              className="max-w-full max-h-full rounded-2xl object-contain shadow-2xl ring-1 ring-black/5"
            />
          </div>
        ) : (
          <ScrollArea className="h-full">
            <div className="p-8">
              {isEditing ? (
                <textarea
                  value={editedContent}
                  onChange={e => setEditedContent(e.target.value)}
                  className="w-full min-h-[600px] bg-transparent border-none text-[13px] font-mono leading-relaxed resize-none focus:outline-none selection:bg-primary/20"
                  spellCheck={false}
                  autoFocus
                />
              ) : (
                <div className="max-w-4xl mx-auto">
                  <pre className="text-[13px] font-mono leading-relaxed whitespace-pre-wrap text-foreground/90">{content}</pre>
                </div>
              )}
            </div>
          </ScrollArea>
        )}
      </div>
    </div>
  );
};
