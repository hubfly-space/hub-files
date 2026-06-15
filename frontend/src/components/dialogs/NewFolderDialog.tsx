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

type NewFolderDialogProps = {
  open: boolean;
  name: string;
  onOpenChange: (open: boolean) => void;
  onNameChange: (name: string) => void;
  onCreate: (name: string) => void;
};

export function NewFolderDialog({
  open,
  name,
  onOpenChange,
  onNameChange,
  onCreate,
}: NewFolderDialogProps) {
  const submit = () => {
    if (!name) return;
    onCreate(name);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>New Folder</DialogTitle>
          <DialogDescription>
            Create a new folder to organize your files.
          </DialogDescription>
        </DialogHeader>

        <div className="py-4">
          <Input
            placeholder="Folder name"
            value={name}
            onChange={(event: React.ChangeEvent<HTMLInputElement>) =>
              onNameChange(event.target.value)
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

          <Button onClick={submit} disabled={!name}>
            Create
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
