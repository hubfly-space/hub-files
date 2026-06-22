import React, { useState, useEffect } from "react";
import { api } from "../api";
import {
  ArrowLeft,
  Download,
  FileText,
  Play,
  Image as ImageIcon,
  File,
  Save,
  Loader2,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { useToast } from "@/hooks/use-toast";

interface FileViewerProps {
  path: string;
  name: string;
  onClose: () => void;
}

export const FileViewer: React.FC<FileViewerProps> = ({
  path,
  name,
  onClose,
}) => {
  // const [loading, setLoading] = useState(true);
  const [content, setContent] = useState<string>("");
  const [saving, setSaving] = useState(false);
  const { toast } = useToast();

  const ext = name.split(".").pop()?.toLowerCase();
  const isImage = ["jpg", "jpeg", "png", "gif", "svg", "webp"].includes(
    ext || "",
  );
  const isVideo = ["mp4", "webm", "ogg", "mov"].includes(ext || "");
  const isText = [
    "txt",
    "md",
    "json",
    "js",
    "jsx",
    "ts",
    "tsx",
    "html",
    "htm",
    "css",
    "scss",
    "php",
    "py",
    "go",
    "rs",
    "mod",
    "java",
    "c",
    "cpp",
    "h",
    "yaml",
    "yml",
    "toml",
    "ini",
    "conf",
    "sh",
    "bat",
    "csv",
    "sql",
    "xml",
    "env",
  ].includes(ext || "");

  useEffect(() => {
    if (!isText) return;

    let cancelled = false;

    api
      .getFile(path)
      .then((text) => {
        if (!cancelled) {
          setContent(text);
        }
      })
      .catch((err) => {
        if (!cancelled) {
          toast({
            title: "Failed to load file",
            description: err.message,
            variant: "destructive",
          });
        }
      });

    return () => {
      cancelled = true;
    };
  }, [path, isText, toast]);

  const handleDownload = () => {
    const session = api.getToken();
    const url = `/api/file?path=${encodeURIComponent(path)}&download=1&session=${session}`;
    const a = document.createElement("a");
    a.href = url;
    a.download = name;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  };

  const handleSave = async () => {
    try {
      setSaving(true);
      await api.putFile(path, content);
      toast({
        title: "Saved successfully",
        description: "Your changes have been saved.",
      });
    } catch {
      toast({
        title: "Failed to save",
        variant: "destructive",
      });
    } finally {
      setSaving(false);
    }
  };

  // if (loading) {
  //   return (
  //     <div className="flex items-center justify-center h-full">
  //       <div className="flex flex-col items-center gap-3">
  //         <Loader2 className="w-8 h-8 animate-spin text-primary" />
  //         <span className="text-sm text-muted-foreground">Loading file...</span>
  //       </div>
  //     </div>
  //   );
  // }

  const session = api.getToken();
  // const fileUrl = `/api/file?path=${encodeURIComponent(path)}&session=${session}`;
  const fileUrl = `http://localhost:10015/api/file?path=${encodeURIComponent(path)}&session=${session}`;

  return (
    <div className="flex flex-col h-full bg-card rounded-[2rem] border border-border/50 shadow-2xl overflow-hidden relative">
      {/* Background Pattern */}
      <div
        className="absolute inset-0 z-0 opacity-[0.03] dark:opacity-[0.05] pointer-events-none"
        style={{
          backgroundImage:
            "radial-gradient(circle, currentColor 1px, transparent 1px)",
          backgroundSize: "24px 24px",
        }}
      />

      <div className="flex items-center justify-between p-5 border-b border-border/40 bg-background/50 backdrop-blur-md shrink-0 z-10">
        <div className="flex items-center gap-4">
          <Button
            variant="secondary"
            size="icon"
            onClick={onClose}
            className="rounded-2xl h-11 w-11 bg-background hover:bg-primary hover:text-primary-foreground transition-all shadow-sm border border-border/50"
          >
            <ArrowLeft className="w-5 h-5" />
          </Button>
          <div className="flex flex-col">
            <h2 className="text-sm font-bold tracking-tight flex items-center gap-2">
              {isImage ? (
                <ImageIcon className="w-4 h-4 text-blue-500" />
              ) : isVideo ? (
                <Play className="w-4 h-4 text-purple-500" />
              ) : (
                <FileText className="w-4 h-4 text-primary" />
              )}
              {name}
            </h2>
            <p className="text-[10px] text-muted-foreground font-medium uppercase tracking-widest">
              {path}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {isText && (
            <Button
              variant="default"
              size="sm"
              onClick={handleSave}
              disabled={saving}
              className="rounded-xl px-4 h-10 font-bold shadow-sm transition-all gap-2"
            >
              {saving ? (
                <Loader2 className="w-4 h-4 animate-spin" />
              ) : (
                <Save className="w-4 h-4" />
              )}
              Save
            </Button>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={handleDownload}
            className="rounded-xl px-4 h-10 font-bold bg-background shadow-sm hover:bg-primary hover:text-primary-foreground transition-all gap-2"
          >
            <Download className="w-4 h-4" />
            Download
          </Button>
        </div>
      </div>

      <div
        className={`flex-1 min-h-0 relative z-10 flex items-center justify-center ${isText ? "" : "p-6 md:p-12"}`}
      >
        {isImage ? (
          <div className="relative group w-full h-full flex items-center justify-center overflow-auto bg-black/5 rounded-b-[2rem]">
            <img
              src={fileUrl}
              alt={name}
              className="max-w-full max-h-full object-contain shadow-2xl transition-transform hover:scale-105 cursor-zoom-in"
              onClick={() => window.open(fileUrl, "_blank")}
            />
          </div>
        ) : isVideo ? (
          <div className="w-full h-full flex flex-col bg-black rounded-b-[2rem] overflow-hidden shadow-2xl relative">
            <video
              src={fileUrl}
              controls
              autoPlay
              className="w-full h-full object-contain"
            />
          </div>
        ) : isText ? (
          <textarea
            value={content}
            onChange={(e) => setContent(e.target.value)}
            className="w-full h-full p-6 bg-transparent border-none resize-none focus:outline-none focus:ring-0 font-mono text-sm leading-relaxed"
            spellCheck={false}
          />
        ) : (
          <div className="flex flex-col items-center gap-8 text-center max-w-md animate-in fade-in zoom-in duration-300">
            <div className="relative">
              <div className="w-32 h-32 bg-secondary/50 rounded-[2.5rem] flex items-center justify-center shadow-inner">
                <File className="w-16 h-16 text-muted-foreground/50" />
              </div>
              <div className="absolute -bottom-2 -right-2 w-12 h-12 bg-primary rounded-2xl flex items-center justify-center text-primary-foreground shadow-lg">
                <Download className="w-6 h-6" />
              </div>
            </div>

            <div className="space-y-3">
              <h3 className="text-2xl font-bold tracking-tight">{name}</h3>
              <p className="text-muted-foreground text-sm leading-relaxed">
                This file type cannot be previewed directly. You can download it
                to view it on your device.
              </p>
            </div>

            <div className="flex flex-col w-full gap-3">
              <Button
                onClick={handleDownload}
                size="lg"
                className="w-full rounded-2xl h-14 text-base font-bold shadow-xl shadow-primary/20"
              >
                <Download className="mr-2 h-5 w-5" />
                Download File
              </Button>
              {/*<Button
                variant="ghost"
                size="sm"
                onClick={() => window.open(fileUrl, "_blank")}
                className="rounded-xl text-muted-foreground"
              >
                <ExternalLink className="mr-2 h-4 w-4" />
                Open in new tab
              </Button>*/}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
