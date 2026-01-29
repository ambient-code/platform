"use client";

import { useState, useEffect } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Loader2 } from "lucide-react";

type ManageRemoteDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSave: (url: string, branch: string) => Promise<void>;
  directoryName: string;
  currentUrl?: string;
  currentBranch?: string; // Not used - always defaults to main
  isLoading?: boolean;
};

export function ManageRemoteDialog({
  open,
  onOpenChange,
  onSave,
  directoryName,
  currentUrl = "",
  isLoading = false,
}: ManageRemoteDialogProps) {
  const [remoteUrl, setRemoteUrl] = useState(currentUrl);

  // Update local state when props change
  useEffect(() => {
    setRemoteUrl(currentUrl);
  }, [currentUrl, open]);

  const handleSave = async () => {
    if (!remoteUrl.trim()) return;
    
    // Always use main branch
    await onSave(remoteUrl.trim(), "main");
  };

  const handleCancel = () => {
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Configure Remote for {directoryName}</DialogTitle>
          <DialogDescription>
            Provide the GitHub repository URL. Branch will default to <code className="bg-muted px-1 rounded">main</code>.
          </DialogDescription>
        </DialogHeader>
        
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="remote-repo-url">Repository URL *</Label>
            <Input
              id="remote-repo-url"
              placeholder="https://github.com/org/my-repo.git"
              value={remoteUrl}
              onChange={(e) => setRemoteUrl(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && remoteUrl.trim()) {
                  handleSave();
                }
              }}
            />
            <p className="text-xs text-muted-foreground">
              Use a repository you have write access to
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={handleCancel}
            disabled={isLoading}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSave}
            disabled={!remoteUrl.trim() || isLoading}
          >
            {isLoading ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Configuring...
              </>
            ) : (
              "Configure Remote"
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

