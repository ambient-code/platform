"use client";

import { useState, useCallback } from "react";
import type { UseFormReturn } from "react-hook-form";
import { Check, X } from "lucide-react";
import { Form } from "@/components/ui/form";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  AgentOptionsFields,
  claudeAgentOptionsDefaults,
  type ClaudeAgentOptionsForm,
} from "./claude-agent-options";

type AdvancedSdkOptionsProps = {
  projectName: string;
  form: UseFormReturn<ClaudeAgentOptionsForm>;
  disabled?: boolean;
  onSave?: () => void;
};

export function AdvancedSdkOptions({
  form,
  disabled,
  onSave,
}: AdvancedSdkOptionsProps) {
  const [showAbandonDialog, setShowAbandonDialog] = useState(false);
  const [snapshotValues, setSnapshotValues] =
    useState<Partial<ClaudeAgentOptionsForm> | null>(null);

  const takeSnapshot = useCallback(() => {
    if (!snapshotValues) {
      setSnapshotValues(form.getValues());
    }
  }, [form, snapshotValues]);

  const isDirty = useCallback(() => {
    const current = form.getValues();
    const compare = snapshotValues ?? claudeAgentOptionsDefaults;
    return JSON.stringify(current) !== JSON.stringify(compare);
  }, [form, snapshotValues]);

  const handleSave = useCallback(() => {
    setSnapshotValues(form.getValues());
    onSave?.();
  }, [form, onSave]);

  const handleCancel = useCallback(() => {
    if (isDirty()) {
      setShowAbandonDialog(true);
    } else {
      onSave?.();
    }
  }, [isDirty, onSave]);

  const handleAbandon = useCallback(() => {
    if (snapshotValues) {
      form.reset(snapshotValues as ClaudeAgentOptionsForm);
    } else {
      form.reset(claudeAgentOptionsDefaults as ClaudeAgentOptionsForm);
    }
    setShowAbandonDialog(false);
    onSave?.();
  }, [form, snapshotValues, onSave]);

  const handleSaveFromDialog = useCallback(() => {
    handleSave();
    setShowAbandonDialog(false);
  }, [handleSave]);

  // Take snapshot on first render inside the dialog
  takeSnapshot();

  return (
    <>
      <div className="space-y-3">
        <Form {...form}>
          <AgentOptionsFields form={form} disabled={disabled} />
        </Form>
        <div className="flex items-center gap-2 pt-2">
          <Button
            type="button"
            size="sm"
            onClick={handleSave}
            disabled={disabled}
          >
            <Check className="h-3.5 w-3.5 mr-1.5" />
            Save Options
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={handleCancel}
            disabled={disabled}
          >
            Cancel
          </Button>
        </div>
      </div>

      {/* Dirty form guard */}
      <Dialog open={showAbandonDialog} onOpenChange={setShowAbandonDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Unsaved SDK Options</DialogTitle>
            <DialogDescription>
              You have unsaved changes to the SDK options. Save them or discard?
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={handleAbandon}>
              <X className="h-3.5 w-3.5 mr-1.5" />
              Discard
            </Button>
            <Button onClick={handleSaveFromDialog}>
              <Check className="h-3.5 w-3.5 mr-1.5" />
              Save
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
