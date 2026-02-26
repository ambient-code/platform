'use client';

import { useState, useEffect, useRef } from 'react';
import { Loader2, Tags } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { LabelEditor } from '@/components/label-editor';
import type { LabelEditorHandle } from '@/components/label-editor';

type EditSessionLabelsDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  currentLabels: Record<string, string>;
  onSave: (labels: Record<string, string>) => void;
  isLoading?: boolean;
};

export function EditSessionLabelsDialog({
  open,
  onOpenChange,
  currentLabels,
  onSave,
  isLoading = false,
}: EditSessionLabelsDialogProps) {
  const [labels, setLabels] = useState<Record<string, string>>(currentLabels);
  const editorRef = useRef<LabelEditorHandle>(null);

  useEffect(() => {
    if (open) {
      setLabels(currentLabels);
    }
  }, [open, currentLabels]);

  const handleSave = () => {
    // Auto-add any valid pending input the user forgot to click +Add for
    if (editorRef.current && !editorRef.current.flush()) {
      // Invalid partial text — let the user fix it
      return;
    }
    onSave(labels);
  };

  const hasChanged = JSON.stringify(labels) !== JSON.stringify(currentLabels);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[480px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Tags className="h-4 w-4" />
            Edit Session Labels
          </DialogTitle>
          <DialogDescription>
            Add or remove labels to organize and filter sessions.
          </DialogDescription>
        </DialogHeader>
        <div className="py-4">
          <LabelEditor
            ref={editorRef}
            labels={labels}
            onChange={setLabels}
            disabled={isLoading}
          />
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
            disabled={isLoading || !hasChanged}
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
