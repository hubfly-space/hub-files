import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";

import { AlertCircle } from "lucide-react";

type DeleteConfirmDialogProps = {
  open: boolean;
  names: string[];
  onOpenChange: (open: boolean) => void;
  onCancel: () => void;
  onDelete: () => void;
};

export function DeleteConfirmDialog({
  open,
  names,
  onOpenChange,
  onCancel,
  onDelete,
}: DeleteConfirmDialogProps) {
  const isBulkDelete = names.length > 1;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="text-destructive flex items-center gap-2">
            <AlertCircle className="w-5 h-5" />
            Delete {isBulkDelete ? "Items" : "Item"}
          </DialogTitle>

          <DialogDescription className="pt-2">
            {isBulkDelete ? (
              <>
                Delete <span className="font-semibold">{names.length} items</span>?
                This cannot be undone.
              </>
            ) : (
              <>
                Delete <span className="font-semibold">"{names[0]}"</span>? This
                cannot be undone.
              </>
            )}
          </DialogDescription>
        </DialogHeader>

        <DialogFooter>
          <Button variant="ghost" onClick={onCancel}>
            Cancel
          </Button>

          <Button variant="destructive" onClick={onDelete}>
            Delete
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
