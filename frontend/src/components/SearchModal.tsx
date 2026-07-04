import { useEffect, useRef, useState } from "react";
import { File, Folder, Loader2, Search, X } from "lucide-react";
import { api } from "../api";
import type { SearchResult } from "../api";

type SearchModalProps = {
  open: boolean;
  onClose: () => void;
  onNavigate: (path: string) => void;
  onOpenFile: (path: string, name: string) => void;
};

export function SearchModal({
  open,
  onClose,
  onNavigate,
  onOpenFile,
}: SearchModalProps) {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const searchTimeoutRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    if (!open) {
      setQuery("");
      setResults([]);
      return;
    }
    inputRef.current?.focus();
  }, [open]);

  useEffect(() => {
    if (searchTimeoutRef.current) {
      clearTimeout(searchTimeoutRef.current);
    }

    if (!query.trim()) {
      setResults([]);
      setLoading(false);
      return;
    }

    setLoading(true);
    searchTimeoutRef.current = setTimeout(async () => {
      try {
        const res = await api.search(query);
        setResults(res);
        setSelectedIndex(0);
      } catch {
        setResults([]);
      } finally {
        setLoading(false);
      }
    }, 300);

    return () => {
      if (searchTimeoutRef.current) clearTimeout(searchTimeoutRef.current);
    };
  }, [query]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") {
      onClose();
      return;
    }
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setSelectedIndex((i) => Math.min(i + 1, results.length - 1));
      return;
    }
    if (e.key === "ArrowUp") {
      e.preventDefault();
      setSelectedIndex((i) => Math.max(i - 1, 0));
      return;
    }
    if (e.key === "Enter" && results[selectedIndex]) {
      selectResult(results[selectedIndex]);
    }
  };

  const selectResult = (result: SearchResult) => {
    onClose();
    if (result.isDir) {
      onNavigate(result.relPath);
    } else {
      onOpenFile(result.relPath, result.baseName);
    }
  };

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[15vh]">
      <div
        className="fixed inset-0 bg-black/60 backdrop-blur-sm"
        onClick={onClose}
      />
      <div className="relative w-full max-w-2xl bg-background rounded-2xl shadow-2xl border border-border/50 overflow-hidden animate-in fade-in zoom-in-95 duration-200">
        <div className="flex items-center gap-3 px-5 py-4 border-b border-border/40">
          <Search className="w-5 h-5 text-muted-foreground shrink-0" />
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Search files and folders..."
            className="flex-1 bg-transparent text-lg outline-none placeholder:text-muted-foreground/40"
          />
          {query && (
            <button
              onClick={() => setQuery("")}
              className="text-muted-foreground hover:text-foreground transition-colors"
            >
              <X className="w-4 h-4" />
            </button>
          )}
        </div>

        <div className="max-h-[60vh] overflow-y-auto">
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="w-6 h-6 animate-spin text-muted-foreground" />
            </div>
          ) : query.trim() && results.length === 0 ? (
            <div className="flex flex-col items-center py-12 text-muted-foreground">
              <Search className="w-8 h-8 mb-2 opacity-40" />
              <p className="text-sm font-medium">No results found</p>
              <p className="text-xs opacity-60">Try a different search term</p>
            </div>
          ) : (
            results.map((result, i) => (
              <button
                key={result.relPath}
                onClick={() => selectResult(result)}
                className={`w-full flex items-center gap-3 px-5 py-3 text-left hover:bg-accent transition-colors ${
                  i === selectedIndex ? "bg-accent" : ""
                }`}
              >
                {result.isDir ? (
                  <Folder className="w-5 h-5 text-primary shrink-0" />
                ) : (
                  <File className="w-5 h-5 text-muted-foreground shrink-0" />
                )}
                <div className="min-w-0 flex-1">
                  <div className="text-sm font-medium truncate">
                    {result.baseName}
                  </div>
                  <div className="text-xs text-muted-foreground/60 truncate">
                    /{result.relPath}
                  </div>
                </div>
                <div className="text-xs text-muted-foreground/40 shrink-0">
                  {result.isDir ? "Folder" : "File"}
                </div>
              </button>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
