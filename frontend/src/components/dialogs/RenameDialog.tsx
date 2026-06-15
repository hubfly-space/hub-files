import React from "react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

type RenameDialogProps = {
  open: boolean;
  oldName: string;
  newName: string;
  onOpenChange: (open: boolean) => void;
  onNewNameChange: (name: string) => void;
  onRename: (oldName: string, newName: string) => void;
};

export function RenameDialog({
  open,
  oldName,
  newName,
  onOpenChange,
  onNewNameChange,
  onRename,
}: RenameDialogProps) {
  const canRename = Boolean(newName) && newName !== oldName;

  const submit = () => {
    if (!canRename) return;
    onRename(oldName, newName);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Rename</DialogTitle>
          <DialogDescription>
            Rename <span className="font-mono text-foreground">{oldName}</span>
          </DialogDescription>
        </DialogHeader>

        <div className="py-4">
          <Input
            value={newName}
            onChange={(event: React.ChangeEvent<HTMLInputElement>) =>
              onNewNameChange(event.target.value)
            }
            onKeyDown={(event: React.KeyboardEvent<HTMLInputElement>) => {
              if (event.key === "Enter") submit();
            }}
            autoFocus
          />
        </div>

        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>

          <Button onClick={submit} disabled={!canRename}>
            Rename
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
