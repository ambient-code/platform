'use client';

import { useState, useEffect } from 'react';
import { Loader2, Pencil } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

type EditSessionNameDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  currentName: string;
  onSave: (newName: string) => void;
  isLoading?: boolean;
};

export function EditSessionNameDialog({
  open,
  onOpenChange,
  currentName,
  onSave,
  isLoading = false,
}: EditSessionNameDialogProps) {
  const [name, setName] = useState(currentName);

  // Reset the input when dialog opens with a new currentName
  useEffect(() => {
    if (open) {
      setName(currentName);
    }
  }, [open, currentName]);

  const handleSave = () => {
    const trimmedName = name.trim();
    if (trimmedName && trimmedName !== currentName) {
      onSave(trimmedName);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !isLoading) {
      e.preventDefault();
      handleSave();
    }
  };

  const isValid = name.trim().length > 0;
  const hasChanged = name.trim() !== currentName;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Pencil className="h-4 w-4" />
            Edit Session Name
          </DialogTitle>
          <DialogDescription>
            Give this session a descriptive name to help identify it later.
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          <div className="grid gap-2">
            <Label htmlFor="session-name">Display Name</Label>
            <Input
              id="session-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Enter a display name..."
              maxLength={50}
              disabled={isLoading}
              autoFocus
            />
            <p className="text-xs text-muted-foreground">
              {name.length}/50 characters
            </p>
          </div>
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isLoading}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSave}
            disabled={isLoading || !isValid || !hasChanged}
          >
            {isLoading ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Saving...
              </>
            ) : (
              'Save'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

