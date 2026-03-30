"use client";

import { useState } from "react";
import { Loader2, Sparkles } from "lucide-react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

type ImportSkillsModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onImportSkills: (source: { url: string; branch: string; path?: string }) => Promise<void>;
  isLoading?: boolean;
};

export function ImportSkillsModal({
  open,
  onOpenChange,
  onImportSkills,
  isLoading = false,
}: ImportSkillsModalProps) {
  const [gitUrl, setGitUrl] = useState("");
  const [branch, setBranch] = useState("");
  const [path, setPath] = useState("");

  const handleSubmit = async () => {
    if (!gitUrl.trim()) return;

    await onImportSkills({
      url: gitUrl.trim(),
      branch: branch.trim() || "main",
      path: path.trim() || undefined,
    });

    // Reset form
    setGitUrl("");
    setBranch("");
    setPath("");
  };

  const handleCancel = () => {
    setGitUrl("");
    setBranch("");
    setPath("");
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Sparkles className="h-5 w-5" />
            Import Skills
          </DialogTitle>
          <DialogDescription>
            Import skills, commands, and agents from a Git repository into this session.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="skills-git-url">Git URL</Label>
            <Input
              id="skills-git-url"
              placeholder="https://github.com/org/skills-repo"
              value={gitUrl}
              onChange={(e) => setGitUrl(e.target.value)}
              disabled={isLoading}
            />
            <p className="text-xs text-muted-foreground">
              URL of the Git repository containing skill definitions
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="skills-branch">Branch (optional)</Label>
            <Input
              id="skills-branch"
              placeholder="main"
              value={branch}
              onChange={(e) => setBranch(e.target.value)}
              disabled={isLoading}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="skills-path">Path (optional)</Label>
            <Input
              id="skills-path"
              placeholder="e.g., helpers"
              value={path}
              onChange={(e) => setPath(e.target.value)}
              disabled={isLoading}
            />
            <p className="text-xs text-muted-foreground">
              Subdirectory within the repository to import from
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={handleCancel}
            disabled={isLoading}
          >
            Cancel
          </Button>
          <Button
            type="button"
            onClick={handleSubmit}
            disabled={!gitUrl.trim() || isLoading}
          >
            {isLoading ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Importing...
              </>
            ) : (
              'Import'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
