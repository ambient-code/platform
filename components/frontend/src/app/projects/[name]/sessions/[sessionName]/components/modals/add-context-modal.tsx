"use client";

import { useState } from "react";
import { Loader2, Info, Upload } from "lucide-react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Separator } from "@/components/ui/separator";

type AddContextModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onAddRepository: (url: string, branch: string, autoPush?: boolean) => Promise<void>;
  onUploadFile?: () => void;
  isLoading?: boolean;
  autoBranch?: string;   // Auto-generated branch from backend (single source of truth)
};

export function AddContextModal({
  open,
  onOpenChange,
  onAddRepository,
  onUploadFile,
  isLoading = false,
  autoBranch,
}: AddContextModalProps) {
  const [contextUrl, setContextUrl] = useState("");
  const [contextBranch, setContextBranch] = useState("");  // Empty = use auto-generated branch
  const [autoPush, setAutoPush] = useState(false);

  const handleSubmit = async () => {
    if (!contextUrl.trim()) return;

    // Trim URL and remove trailing slash
    const sanitizedUrl = contextUrl.trim().replace(/\/+$/, '');

    // Use autoBranch from backend (single source of truth), or empty to let runner auto-generate
    const defaultBranch = autoBranch || '';
    await onAddRepository(sanitizedUrl, contextBranch.trim() || defaultBranch, autoPush);

    // Reset form
    setContextUrl("");
    setContextBranch("");
    setAutoPush(false);
  };

  const handleCancel = () => {
    setContextUrl("");
    setContextBranch("");
    setAutoPush(false);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>Add Context</DialogTitle>
          <DialogDescription>
            Add additional context to improve AI responses.
          </DialogDescription>
        </DialogHeader>
        
        <div className="space-y-4">
          <Alert>
            <Info className="h-4 w-4" />
            <AlertDescription>
              Note: additional data sources like Jira, Google Drive, files, and MCP Servers are on the roadmap!
            </AlertDescription>
          </Alert>

          <div className="space-y-2">
            <Label htmlFor="context-url">Repository URL</Label>
            <Input
              id="context-url"
              placeholder="https://github.com/org/repo"
              value={contextUrl}
              onChange={(e) => setContextUrl(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Currently supports GitHub repositories for code context
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="context-branch">Branch (optional)</Label>
            <Input
              id="context-branch"
              // Use autoBranch from backend (single source of truth)
              placeholder={autoBranch}
              value={contextBranch}
              onChange={(e) => setContextBranch(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              If left empty, a unique feature branch will be created for this session
            </p>
          </div>

          <div className="flex items-start space-x-2">
            <Checkbox
              id="auto-push"
              checked={autoPush}
              onCheckedChange={(checked) => setAutoPush(checked === true)}
            />
            <div className="space-y-1">
              <Label
                htmlFor="auto-push"
                className="text-sm font-normal cursor-pointer"
              >
                Enable auto-push
              </Label>
              <p className="text-xs text-muted-foreground">
                Instructs Claude to commit and push changes made to this
                repository during the session. Requires git credentials to be
                configured.
              </p>
            </div>
          </div>

          {onUploadFile && (
            <>
              <Separator className="my-4" />
              <div className="space-y-2">
                <Label>Upload Files</Label>
                <p className="text-xs text-muted-foreground mb-2">
                  Upload files directly to your workspace for use as context
                </p>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => {
                    onUploadFile();
                    onOpenChange(false);
                  }}
                  className="w-full"
                >
                  <Upload className="h-4 w-4 mr-2" />
                  Upload Files
                </Button>
              </div>
            </>
          )}
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={handleCancel}
          >
            Cancel
          </Button>
          <Button
            type="button"
            onClick={handleSubmit}
            disabled={!contextUrl.trim() || isLoading}
          >
            {isLoading ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Adding...
              </>
            ) : (
              'Add'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

