"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { InputWithHistory } from "@/components/input-with-history";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Skeleton } from "@/components/ui/skeleton";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Brain,
  Plus,
  ExternalLink,
  GitPullRequestDraft,
  Filter,
  Loader2,
  BookOpen,
  Lightbulb,
  AlertTriangle,
} from "lucide-react";
import { toast } from "sonner";
import { formatDistanceToNow } from "date-fns";
import { useLearnedFiles, useLearnedDraftPRs, useCreateMemory } from "@/services/queries/use-learned";
import { useInputHistory } from "@/hooks/use-input-history";
import type { LearnedEntry, LearnedDraftPR } from "@/services/api/learned";

type ProjectMemorySectionProps = {
  projectName: string;
};

const PAGE_SIZE = 50;

export function ProjectMemorySection({ projectName }: ProjectMemorySectionProps) {
  const [typeFilter, setTypeFilter] = useState<string>("");
  const [currentPage, setCurrentPage] = useState(0);
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [newMemory, setNewMemory] = useState({
    title: "",
    content: "",
    type: "correction" as "correction" | "pattern",
    repo: "",
  });

  const {
    data: learnedData,
    isLoading: entriesLoading,
  } = useLearnedFiles(projectName, {
    type: typeFilter || undefined,
    page: currentPage,
    pageSize: PAGE_SIZE,
  });

  const {
    data: prsData,
    isLoading: prsLoading,
  } = useLearnedDraftPRs(projectName);

  const createMemory = useCreateMemory();
  const { addToHistory: addRepoToHistory } = useInputHistory("memory-target-repo");

  const entries = learnedData?.entries || [];
  const totalCount = learnedData?.totalCount || 0;
  const draftPRs = prsData?.prs || [];
  const totalPages = Math.ceil(totalCount / PAGE_SIZE);

  const handleCreateMemory = () => {
    if (!newMemory.title.trim() || !newMemory.content.trim()) {
      toast.error("Title and content are required");
      return;
    }

    createMemory.mutate(
      {
        projectName,
        data: {
          title: newMemory.title.trim(),
          content: newMemory.content.trim(),
          type: newMemory.type,
          ...(newMemory.repo.trim() ? { repo: newMemory.repo.trim() } : {}),
        },
      },
      {
        onSuccess: (result) => {
          if (newMemory.repo.trim()) {
            addRepoToHistory(newMemory.repo.trim());
          }
          toast.success("Memory created successfully");
          setAddDialogOpen(false);
          setNewMemory({ title: "", content: "", type: "correction", repo: "" });
          if (result.prUrl) {
            window.open(result.prUrl, "_blank");
          }
        },
        onError: () => {
          toast.error("Failed to create memory");
        },
      }
    );
  };

  // Empty state
  if (!entriesLoading && entries.length === 0 && !typeFilter && draftPRs.length === 0) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold">Project Memory</h2>
            <p className="text-sm text-muted-foreground">
              Knowledge accumulated across sessions
            </p>
          </div>
          <AddMemoryButton
            open={addDialogOpen}
            onOpenChange={setAddDialogOpen}
            newMemory={newMemory}
            setNewMemory={setNewMemory}
            onSubmit={handleCreateMemory}
            isPending={createMemory.isPending}
          />
        </div>

        <Card>
          <CardContent className="flex flex-col items-center justify-center py-16 text-center">
            <Brain className="h-12 w-12 text-muted-foreground/40 mb-4" />
            <h3 className="text-lg font-medium mb-2">No memories yet</h3>
            <p className="text-sm text-muted-foreground max-w-md mb-4">
              Project memories are corrections and patterns learned from past sessions.
              They are stored as markdown files in{" "}
              <code className="text-xs bg-muted px-1 py-0.5 rounded">docs/learned/</code> and
              automatically injected into future sessions.
            </p>
            <p className="text-sm text-muted-foreground max-w-md">
              Memories can be suggested by the agent during a session or added
              manually using the &quot;Add Memory&quot; button above.
            </p>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">Project Memory</h2>
          <p className="text-sm text-muted-foreground">
            {totalCount} {totalCount === 1 ? "entry" : "entries"} learned across sessions
          </p>
        </div>
        <div className="flex items-center gap-2">
          {/* Type filter */}
          <Select
            value={typeFilter || "all"}
            onValueChange={(v) => {
              setTypeFilter(v === "all" ? "" : v);
              setCurrentPage(0);
            }}
          >
            <SelectTrigger className="w-[140px]">
              <Filter className="h-4 w-4 mr-2" />
              <SelectValue placeholder="All types" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All types</SelectItem>
              <SelectItem value="correction">Corrections</SelectItem>
              <SelectItem value="pattern">Patterns</SelectItem>
            </SelectContent>
          </Select>

          <AddMemoryButton
            open={addDialogOpen}
            onOpenChange={setAddDialogOpen}
            newMemory={newMemory}
            setNewMemory={setNewMemory}
            onSubmit={handleCreateMemory}
            isPending={createMemory.isPending}
          />
        </div>
      </div>

      {/* Pending Review Section */}
      {draftPRs.length > 0 && (
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <GitPullRequestDraft className="h-4 w-4 text-orange-500" />
              Pending Review
              <Badge variant="secondary" className="text-xs">
                {draftPRs.length}
              </Badge>
            </CardTitle>
            <CardDescription className="text-xs">
              Draft PRs awaiting curation before they become project memory
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            {prsLoading ? (
              <div className="space-y-2">
                {Array.from({ length: 2 }).map((_, i) => (
                  <Skeleton key={i} className="h-12 w-full" />
                ))}
              </div>
            ) : (
              draftPRs.map((pr: LearnedDraftPR) => (
                <div
                  key={pr.number}
                  className="flex items-center justify-between p-3 rounded-lg border bg-card hover:bg-accent/50 transition-colors"
                >
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium truncate">
                        {pr.title}
                      </span>
                      <Badge variant="outline" className="text-xs shrink-0">
                        #{pr.number}
                      </Badge>
                    </div>
                    <div className="text-xs text-muted-foreground mt-0.5">
                      by {pr.author}{" "}
                      {pr.createdAt &&
                        formatDistanceToNow(new Date(pr.createdAt), {
                          addSuffix: true,
                        })}
                    </div>
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => window.open(pr.url, "_blank")}
                  >
                    <ExternalLink className="h-4 w-4" />
                  </Button>
                </div>
              ))
            )}
          </CardContent>
        </Card>
      )}

      {/* Learned Entries */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-medium flex items-center gap-2">
            <BookOpen className="h-4 w-4" />
            Learned Files
          </CardTitle>
        </CardHeader>
        <CardContent>
          {entriesLoading ? (
            <div className="space-y-3">
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-20 w-full" />
              ))}
            </div>
          ) : entries.length === 0 ? (
            <div className="text-center py-8 text-sm text-muted-foreground">
              No {typeFilter ? `${typeFilter} ` : ""}entries found
            </div>
          ) : (
            <div className="space-y-3">
              {entries.map((entry: LearnedEntry, idx: number) => (
                <div
                  key={`${entry.filePath}-${idx}`}
                  className="p-4 rounded-lg border bg-card hover:bg-accent/50 transition-colors"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <span className="font-medium text-sm">
                          {entry.title}
                        </span>
                        <TypeBadge type={entry.type} />
                      </div>
                      <div className="text-xs text-muted-foreground mb-2">
                        {entry.date && <span>{entry.date}</span>}
                        {entry.author && <span> by {entry.author}</span>}
                        {entry.session && (
                          <span className="ml-2 text-muted-foreground/60">
                            session: {entry.session}
                          </span>
                        )}
                      </div>
                      {entry.contentPreview && (
                        <p className="text-sm text-muted-foreground line-clamp-3">
                          {entry.contentPreview}
                        </p>
                      )}
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="shrink-0"
                      title="View on GitHub"
                      onClick={() => {
                        // Best-effort GitHub link via file path
                        // The actual repo URL is resolved server-side
                      }}
                    >
                      <ExternalLink className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              ))}

              {/* Pagination */}
              {totalPages > 1 && (
                <div className="flex items-center justify-between pt-4 border-t">
                  <span className="text-xs text-muted-foreground">
                    Page {currentPage + 1} of {totalPages}
                  </span>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={currentPage === 0}
                      onClick={() => setCurrentPage((p) => Math.max(0, p - 1))}
                    >
                      Previous
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={currentPage >= totalPages - 1}
                      onClick={() => setCurrentPage((p) => p + 1)}
                    >
                      Next
                    </Button>
                  </div>
                </div>
              )}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

// ------------------------------------------------------------------
// Sub-components
// ------------------------------------------------------------------

function TypeBadge({ type }: { type: "correction" | "pattern" }) {
  if (type === "correction") {
    return (
      <Badge variant="destructive" className="text-xs">
        <AlertTriangle className="h-3 w-3 mr-1" />
        Correction
      </Badge>
    );
  }
  return (
    <Badge variant="secondary" className="text-xs">
      <Lightbulb className="h-3 w-3 mr-1" />
      Pattern
    </Badge>
  );
}

type AddMemoryButtonProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  newMemory: {
    title: string;
    content: string;
    type: "correction" | "pattern";
    repo: string;
  };
  setNewMemory: (m: {
    title: string;
    content: string;
    type: "correction" | "pattern";
    repo: string;
  }) => void;
  onSubmit: () => void;
  isPending: boolean;
};

function AddMemoryButton({
  open,
  onOpenChange,
  newMemory,
  setNewMemory,
  onSubmit,
  isPending,
}: AddMemoryButtonProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogTrigger asChild>
        <Button size="sm">
          <Plus className="h-4 w-4 mr-1" />
          Add Memory
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add Project Memory</DialogTitle>
          <DialogDescription>
            Create a new memory entry. This will open a draft PR in the target
            repository.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-2">
          <div className="space-y-2">
            <Label htmlFor="memory-repo">Target Repository</Label>
            <InputWithHistory
              id="memory-repo"
              historyKey="memory-target-repo"
              placeholder="owner/repo (e.g. jeremyeder/continuous-learning-example)"
              value={newMemory.repo}
              onChange={(e) =>
                setNewMemory({ ...newMemory, repo: e.target.value })
              }
            />
            <p className="text-xs text-muted-foreground">
              GitHub repository where the learned file will be added as a draft PR
            </p>
          </div>
          <div className="space-y-2">
            <Label htmlFor="memory-title">Title</Label>
            <Input
              id="memory-title"
              placeholder="Short descriptive title"
              value={newMemory.title}
              onChange={(e) =>
                setNewMemory({ ...newMemory, title: e.target.value })
              }
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="memory-type">Type</Label>
            <Select
              value={newMemory.type}
              onValueChange={(v) =>
                setNewMemory({
                  ...newMemory,
                  type: v as "correction" | "pattern",
                })
              }
            >
              <SelectTrigger id="memory-type">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="correction">
                  Correction -- something wrong to avoid
                </SelectItem>
                <SelectItem value="pattern">
                  Pattern -- an effective approach to repeat
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="memory-content">Content</Label>
            <Textarea
              id="memory-content"
              placeholder="Detailed description of the learned knowledge..."
              rows={6}
              value={newMemory.content}
              onChange={(e) =>
                setNewMemory({ ...newMemory, content: e.target.value })
              }
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={onSubmit}
            disabled={
              isPending ||
              !newMemory.title.trim() ||
              !newMemory.content.trim() ||
              !newMemory.repo.trim()
            }
          >
            {isPending ? (
              <>
                <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                Creating...
              </>
            ) : (
              "Create Memory"
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
