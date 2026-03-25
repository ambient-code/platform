"use client";

import { useState, useMemo } from "react";
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
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import { Loader2, Search } from "lucide-react";
import { toast } from "sonner";
import { useScanGitSource, useInstallItems } from "@/services/queries/use-marketplace";
import type { DiscoveredItem, InstalledItem } from "@/types/marketplace";
import { MARKETPLACE_CATEGORY_COLORS } from "@/types/marketplace";

type ImportSourceDialogProps = {
  projectName: string;
  prefillUrl?: string;
  prefillItems?: string[];
  onImported?: () => void;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  sessionName?: string;
};

export function ImportSourceDialog({
  projectName,
  prefillUrl = "",
  prefillItems = [],
  onImported,
  open,
  onOpenChange,
  sessionName,
}: ImportSourceDialogProps) {
  const [gitUrl, setGitUrl] = useState(prefillUrl);
  const [branch, setBranch] = useState("main");
  const [path, setPath] = useState("");
  const [selectedIds, setSelectedIds] = useState<Set<string>>(
    new Set(prefillItems)
  );
  const [importAsWorkflow, setImportAsWorkflow] = useState(false);

  const scanMutation = useScanGitSource();
  const installMutation = useInstallItems();

  const scannedItems = scanMutation.data?.items ?? [];
  const isWorkflow = scanMutation.data?.isWorkflow ?? false;

  const grouped = useMemo(() => {
    const groups: Record<string, DiscoveredItem[]> = {
      skill: [],
      command: [],
      agent: [],
    };
    for (const item of scannedItems) {
      (groups[item.type] ??= []).push(item);
    }
    return groups;
  }, [scannedItems]);

  const handleScan = () => {
    if (!gitUrl.trim()) return;
    scanMutation.mutate(
      { url: gitUrl.trim(), branch: branch.trim() || "main", path: path.trim() || undefined },
      {
        onSuccess: (data) => {
          setSelectedIds(new Set(data.items.map((i) => i.id)));
          setImportAsWorkflow(data.isWorkflow);
        },
      }
    );
  };

  const toggleItem = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const selectAll = () => setSelectedIds(new Set(scannedItems.map((i) => i.id)));
  const deselectAll = () => setSelectedIds(new Set());

  const handleImport = () => {
    const selected = scannedItems.filter((i) => selectedIds.has(i.id));
    if (selected.length === 0) return;

    const items: InstalledItem[] = selected.map((item) => ({
      sourceUrl: gitUrl.trim(),
      sourceBranch: branch.trim() || "main",
      sourcePath: path.trim() || undefined,
      itemId: item.id,
      itemType: importAsWorkflow ? "workflow" : item.type,
      itemName: item.name,
      filePath: item.filePath,
    }));

    installMutation.mutate(
      { projectName, items },
      {
        onSuccess: () => {
          toast.success(`Imported ${items.length} item${items.length > 1 ? "s" : ""}`);
          onImported?.();
          onOpenChange(false);
          resetForm();
        },
        onError: (error) => {
          toast.error(error instanceof Error ? error.message : "Import failed");
        },
      }
    );
  };

  const resetForm = () => {
    setGitUrl(prefillUrl);
    setBranch("main");
    setPath("");
    setSelectedIds(new Set(prefillItems));
    setImportAsWorkflow(false);
    scanMutation.reset();
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        if (!v) resetForm();
        onOpenChange(v);
      }}
    >
      <DialogContent className="sm:max-w-[600px] max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Import from Git Source</DialogTitle>
          <DialogDescription>
            Scan a Git repository for skills, commands, and agents to import
            {sessionName ? ` into session "${sessionName}"` : ""}.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="import-git-url">Git URL</Label>
            <Input
              id="import-git-url"
              placeholder="https://github.com/org/repo.git"
              value={gitUrl}
              onChange={(e) => setGitUrl(e.target.value)}
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-2">
              <Label htmlFor="import-branch">Branch</Label>
              <Input
                id="import-branch"
                placeholder="main"
                value={branch}
                onChange={(e) => setBranch(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="import-path">Path (optional)</Label>
              <Input
                id="import-path"
                placeholder="path/to/workflow"
                value={path}
                onChange={(e) => setPath(e.target.value)}
              />
            </div>
          </div>

          <Button
            onClick={handleScan}
            disabled={!gitUrl.trim() || scanMutation.isPending}
            className="w-full"
          >
            {scanMutation.isPending ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                Scanning...
              </>
            ) : (
              <>
                <Search className="w-4 h-4 mr-2" />
                Scan Repository
              </>
            )}
          </Button>

          {scanMutation.isError && (
            <p className="text-sm text-destructive">
              {scanMutation.error instanceof Error
                ? scanMutation.error.message
                : "Scan failed"}
            </p>
          )}

          {scanMutation.isSuccess && (
            <div className="space-y-3 border-t pt-3">
              {scannedItems.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  No items discovered in this repository.
                </p>
              ) : (
                <>
                  {isWorkflow && (
                    <div className="flex items-center gap-2 p-3 rounded-md bg-muted">
                      <Checkbox
                        id="import-as-workflow"
                        checked={importAsWorkflow}
                        onCheckedChange={(v) => setImportAsWorkflow(v === true)}
                      />
                      <Label htmlFor="import-as-workflow" className="cursor-pointer">
                        Import as Workflow
                      </Label>
                      {scanMutation.data?.workflowName && (
                        <span className="text-xs text-muted-foreground ml-auto">
                          {scanMutation.data.workflowName}
                        </span>
                      )}
                    </div>
                  )}

                  <div className="flex items-center gap-2">
                    <Button variant="ghost" size="sm" onClick={selectAll}>
                      Select All
                    </Button>
                    <Button variant="ghost" size="sm" onClick={deselectAll}>
                      Deselect All
                    </Button>
                    <span className="text-xs text-muted-foreground ml-auto">
                      {selectedIds.size} of {scannedItems.length} selected
                    </span>
                  </div>

                  {(["skill", "command", "agent"] as const).map((type) => {
                    const items = grouped[type];
                    if (items.length === 0) return null;
                    return (
                      <div key={type} className="space-y-2">
                        <h4 className="text-sm font-medium capitalize">
                          {type}s ({items.length})
                        </h4>
                        {items.map((item) => (
                          <div
                            key={item.id}
                            className="flex items-start gap-2 p-2 rounded-md border"
                          >
                            <Checkbox
                              id={`item-${item.id}`}
                              checked={selectedIds.has(item.id)}
                              onCheckedChange={() => toggleItem(item.id)}
                              className="mt-0.5"
                            />
                            <div className="flex-1 min-w-0">
                              <div className="flex items-center gap-2">
                                <Label
                                  htmlFor={`item-${item.id}`}
                                  className="text-sm font-medium cursor-pointer"
                                >
                                  {item.name}
                                </Label>
                                <Badge
                                  variant="secondary"
                                  className={MARKETPLACE_CATEGORY_COLORS[item.type]}
                                >
                                  {item.type}
                                </Badge>
                              </div>
                              {item.description && (
                                <p className="text-xs text-muted-foreground mt-0.5">
                                  {item.description}
                                </p>
                              )}
                            </div>
                          </div>
                        ))}
                      </div>
                    );
                  })}
                </>
              )}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleImport}
            disabled={
              selectedIds.size === 0 ||
              installMutation.isPending ||
              !scanMutation.isSuccess
            }
          >
            {installMutation.isPending ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                Importing...
              </>
            ) : (
              `Import ${selectedIds.size} Item${selectedIds.size !== 1 ? "s" : ""}`
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
