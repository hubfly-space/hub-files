import { motion } from "framer-motion";

import { cn } from "@/lib/utils";
import { formatBytes } from "../../utils/format";

type StorageMeterProps = {
  storage: {
    usedPercent: number;
    usedBytes: number;
    totalBytes: number;
  };
};

export function StorageMeter({ storage }: StorageMeterProps) {
  return (
    <div className="hidden lg:flex items-center gap-4 bg-secondary/40 px-3 py-1.5 rounded-full border border-border/50">
      <div className="flex flex-col items-end">
        <div className="flex items-center gap-2">
          <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
            Storage
          </span>
          <span className="text-xs font-semibold">
            {storage.usedPercent.toFixed(0)}%
          </span>
        </div>

        <p className="text-[10px] text-muted-foreground font-medium">
          {formatBytes(storage.usedBytes)} of {formatBytes(storage.totalBytes)}
        </p>
      </div>

      <div className="w-20 h-1.5 rounded-full bg-border/50 overflow-hidden">
        <motion.div
          initial={{ width: 0 }}
          animate={{ width: `${storage.usedPercent}%` }}
          className={cn(
            "h-full rounded-full transition-all",
            storage.usedPercent > 90 ? "bg-destructive" : "bg-primary",
          )}
        />
      </div>
    </div>
  );
}
