import { motion } from "framer-motion";
import { Upload } from "lucide-react";

export function FileDropOverlay() {
  return (
    <div className="absolute inset-0 z-50 flex items-center justify-center bg-background/40 backdrop-blur-[2px]">
      <motion.div
        initial={{ scale: 0.9, opacity: 0 }}
        animate={{ scale: 1, opacity: 1 }}
        className="bg-background border-2 border-dashed border-primary/30 p-10 rounded-[2rem] shadow-2xl flex flex-col items-center gap-4"
      >
        <div className="w-16 h-16 bg-primary/10 rounded-full flex items-center justify-center text-primary">
          <Upload className="w-8 h-8 animate-bounce" />
        </div>

        <div className="text-center">
          <p className="text-lg font-bold">Ready to upload</p>
          <p className="text-sm text-muted-foreground">
            Drop your files here to start
          </p>
        </div>
      </motion.div>
    </div>
  );
}
