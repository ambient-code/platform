"use client";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { useRepoBranches } from "@/services/queries";
import type { SessionRepo } from "@/types/agentic-session";
import { DEFAULT_BRANCH, sanitizeUrlForDisplay } from "@/utils/repo";
import { useState } from "react";

type RepositoryDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  repo: SessionRepo;
  onRepoChange: (repo: SessionRepo) => void;
  onSave: () => void;
  isEditing: boolean;
  projectName: string;
  defaultAutoPush?: boolean;
};

export function RepositoryDialog({
  open,
  onOpenChange,
  repo,
  onRepoChange,
  onSave,
  isEditing,
  projectName,
  defaultAutoPush = false,
}: RepositoryDialogProps) {
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [validationError, setValidationError] = useState<string | null>(null);

  // Fetch branches for the repository
  const inputUrl = repo.input?.url || "";
  const { data: branchesData, isLoading: branchesLoading, error: branchesError } = useRepoBranches(
    projectName,
    inputUrl,
    { enabled: !!inputUrl && open }
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{isEditing ? "Edit Repository" : "Add Repository"}</DialogTitle>
          <DialogDescription>Configure repository source, auto-push, and optional output destination</DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          {/* Input Configuration */}
          <div className="space-y-4 border rounded-md p-4">
            <div className="space-y-2">
              <Label className="text-sm font-medium">Repository URL</Label>
              <Input
                placeholder="https://github.com/org/repo.git"
                value={inputUrl}
                onChange={(e) => onRepoChange({
                  ...repo,
                  input: { ...repo.input, url: e.target.value }
                })}
              />
            </div>
            <div className="space-y-2">
              <Label className="text-sm font-medium">Branch</Label>
              <Select
                value={repo.input?.branch || DEFAULT_BRANCH}
                onValueChange={(value) => onRepoChange({
                  ...repo,
                  input: { ...repo.input, url: inputUrl, branch: value }
                })}
              >
                <SelectTrigger>
                  <SelectValue placeholder={branchesLoading ? "Loading branches..." : "Select branch"} />
                </SelectTrigger>
                <SelectContent>
                  {branchesLoading ? (
                    <SelectItem value="loading" disabled>Loading branches...</SelectItem>
                  ) : branchesData?.branches && branchesData.branches.length > 0 ? (
                    branchesData.branches.map((branch) => (
                      <SelectItem key={branch.name} value={branch.name}>
                        {branch.name}
                      </SelectItem>
                    ))
                  ) : (
                    <>
                      <SelectItem value={DEFAULT_BRANCH}>{DEFAULT_BRANCH}</SelectItem>
                      <SelectItem value="master">master</SelectItem>
                      <SelectItem value="develop">develop</SelectItem>
                    </>
                  )}
                </SelectContent>
              </Select>
              {!inputUrl && (
                <p className="text-xs text-muted-foreground">Enter repository URL first to load branches</p>
              )}
              {branchesError && (
                <p className="text-xs text-red-600 dark:text-red-400">
                  Failed to load branches. Using default branches. Error: {branchesError instanceof Error ? branchesError.message : String(branchesError)}
                </p>
              )}
            </div>
          </div>

          {/* Auto-push Configuration */}
          <div className="flex items-start space-x-3 space-y-0 rounded-md border p-4">
            <Checkbox
              id="autoPush"
              // If autoPush is explicitly set (true/false), use that value.
              // Otherwise, fall back to the session's defaultAutoPush setting.
              // This ensures new repos inherit the default, while edited repos keep their value.
              checked={repo.autoPush ?? defaultAutoPush}
              onCheckedChange={(checked) => onRepoChange({
                ...repo,
                autoPush: Boolean(checked)
              })}
            />
            <div className="space-y-1 leading-none">
              <Label htmlFor="autoPush" className="text-sm font-medium cursor-pointer">
                Auto-push changes on completion
              </Label>
              <p className="text-xs text-muted-foreground">
                When enabled, Claude will automatically commit and push changes to this repository when the session completes.
              </p>
            </div>
          </div>

          {/* Advanced: Output Configuration */}
          <div className="space-y-2">
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => setShowAdvanced(!showAdvanced)}
              className="h-8 text-xs"
              aria-expanded={showAdvanced}
            >
              {showAdvanced ? "Hide" : "Show"} Advanced Options
            </Button>

            {/* Show summary when collapsed but output is configured */}
            {!showAdvanced && repo.output?.url && (
              <div className="text-xs text-muted-foreground pl-2 border-l-2 border-muted">
                <span className="font-medium">Output:</span> {sanitizeUrlForDisplay(repo.output.url)}
                {repo.output.branch && <span> ({repo.output.branch})</span>}
              </div>
            )}

            {showAdvanced && (
              <div className="space-y-4 border rounded-md p-4 bg-muted/50">
                <p className="text-xs text-muted-foreground">
                  By default, changes are pushed back to the input repository. Configure output to push to a different fork or branch.
                </p>
                <div className="space-y-2">
                  <Label className="text-sm font-medium">Output Repository URL (optional)</Label>
                  <Input
                    placeholder="https://github.com/your-fork/repo.git"
                    value={repo.output?.url || ""}
                    onChange={(e) => onRepoChange({
                      ...repo,
                      output: e.target.value ? {
                        url: e.target.value,
                        ...(repo.output?.branch && { branch: repo.output.branch })
                      } : undefined
                    })}
                  />
                  <p className="text-xs text-muted-foreground">
                    Leave empty to push to the same repository as input
                  </p>
                </div>
                {repo.output?.url && (
                  <div className="space-y-2">
                    <Label className="text-sm font-medium">Output Branch (optional)</Label>
                    <Input
                      placeholder="feature-branch"
                      value={repo.output?.branch || ""}
                      onChange={(e) => onRepoChange({
                        ...repo,
                        output: {
                          ...repo.output,
                          url: repo.output!.url,
                          branch: e.target.value || undefined
                        }
                      })}
                    />
                    <p className="text-xs text-muted-foreground">
                      Leave empty to use the same branch as input
                    </p>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
        <div className="space-y-2">
          {validationError && (
            <p className="text-xs text-red-600 dark:text-red-400 px-2">
              {validationError}
            </p>
          )}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => {
              setValidationError(null);
              onOpenChange(false);
            }}>
              Cancel
            </Button>
            <Button
              type="button"
              onClick={() => {
                // Clear previous validation errors
                setValidationError(null);

                // Validate required fields
                if (!inputUrl) {
                  setValidationError("Repository URL is required");
                  return;
                }

                // Validate output differs from input
                if (repo.output?.url) {
                  const inputUrlTrimmed = (repo.input?.url || "").trim();
                  const outputUrlTrimmed = (repo.output?.url || "").trim();
                  const inputBranch = (repo.input?.branch || "").trim();
                  const outputBranch = (repo.output?.branch || "").trim();

                  if (inputUrlTrimmed === outputUrlTrimmed && inputBranch === outputBranch) {
                    setValidationError("Output repository must differ from input (different URL or branch required)");
                    return;
                  }
                }

                onSave();
                onOpenChange(false);
              }}
            >
              {isEditing ? "Update" : "Add"}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
