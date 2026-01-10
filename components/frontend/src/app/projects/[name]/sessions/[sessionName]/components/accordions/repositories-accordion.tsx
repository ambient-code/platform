"use client";

import { useState } from "react";
import { GitBranch, X, Link, Loader2, CloudUpload, ChevronDown, ChevronRight } from "lucide-react";
import { AccordionItem, AccordionTrigger, AccordionContent } from "@/components/ui/accordion";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

type Repository = {
  url: string;
  name?: string;
  branch?: string; // DEPRECATED: Use currentActiveBranch instead
  branches?: string[]; // All local branches available
  currentActiveBranch?: string; // Currently checked out branch
  defaultBranch?: string; // Default branch of remote
};

type UploadedFile = {
  name: string;
  path: string;
  size?: number;
};

type RepositoriesAccordionProps = {
  repositories?: Repository[];
  uploadedFiles?: UploadedFile[];
  onAddRepository: () => void;
  onRemoveRepository: (repoName: string) => void;
  onRemoveFile?: (fileName: string) => void;
};

export function RepositoriesAccordion({
  repositories = [],
  uploadedFiles = [],
  onAddRepository,
  onRemoveRepository,
  onRemoveFile,
}: RepositoriesAccordionProps) {
  const [removingRepo, setRemovingRepo] = useState<string | null>(null);
  const [removingFile, setRemovingFile] = useState<string | null>(null);
  const [expandedRepos, setExpandedRepos] = useState<Set<string>>(new Set());

  const totalContextItems = repositories.length + uploadedFiles.length;

  const handleRemoveRepo = async (repoName: string) => {
    if (confirm(`Remove repository ${repoName}?`)) {
      setRemovingRepo(repoName);
      try {
        await onRemoveRepository(repoName);
      } finally {
        setRemovingRepo(null);
      }
    }
  };

  const handleRemoveFile = async (fileName: string) => {
    if (!onRemoveFile) return;
    if (confirm(`Remove file ${fileName}?`)) {
      setRemovingFile(fileName);
      try {
        await onRemoveFile(fileName);
      } finally {
        setRemovingFile(null);
      }
    }
  };

  return (
    <AccordionItem value="context" className="border rounded-lg px-3 bg-card">
      <AccordionTrigger className="text-base font-semibold hover:no-underline py-3">
        <div className="flex items-center gap-2">
          <Link className="h-4 w-4" />
          <span>Context</span>
          {totalContextItems > 0 && (
            <Badge variant="secondary" className="ml-auto mr-2">
              {totalContextItems}
            </Badge>
          )}
        </div>
      </AccordionTrigger>
      <AccordionContent className="pt-2 pb-3">
        <div className="space-y-3">
          <p className="text-sm text-muted-foreground">
            Add additional context to improve AI responses.
          </p>
          
          {/* Context Items List (Repos + Uploaded Files) */}
          {totalContextItems === 0 ? (
            <div className="text-center py-6">
              <div className="inline-flex items-center justify-center w-12 h-12 rounded-full bg-muted mb-2">
                <Link className="h-5 w-5 text-muted-foreground/60" />
              </div>
              <p className="text-sm text-muted-foreground mb-3">No context added yet</p>
              <Button size="sm" variant="outline" onClick={onAddRepository}>
                <Link className="mr-2 h-3 w-3" />
                Add Context
              </Button>
            </div>
          ) : (
            <div className="space-y-2">
              {/* Repositories */}
              {repositories.map((repo, idx) => {
                const repoName = repo.name || repo.url.split('/').pop()?.replace('.git', '') || `repo-${idx}`;
                const isRemoving = removingRepo === repoName;
                const isExpanded = expandedRepos.has(repoName);
                const currentBranch = repo.currentActiveBranch || repo.branch;
                const hasBranches = repo.branches && repo.branches.length > 0;

                const toggleExpanded = () => {
                  setExpandedRepos(prev => {
                    const next = new Set(prev);
                    if (next.has(repoName)) {
                      next.delete(repoName);
                    } else {
                      next.add(repoName);
                    }
                    return next;
                  });
                };

                return (
                  <div key={`repo-${idx}`} className="border rounded bg-muted/30">
                    <div className="flex items-center gap-2 p-2 hover:bg-muted/50 transition-colors">
                      {hasBranches && (
                        <button
                          onClick={toggleExpanded}
                          className="h-4 w-4 text-muted-foreground flex-shrink-0 hover:text-foreground"
                        >
                          {isExpanded ? (
                            <ChevronDown className="h-4 w-4" />
                          ) : (
                            <ChevronRight className="h-4 w-4" />
                          )}
                        </button>
                      )}
                      {!hasBranches && <div className="h-4 w-4 flex-shrink-0" />}
                      <GitBranch className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <div className="text-sm font-medium truncate">{repoName}</div>
                          {currentBranch && (
                            <Badge variant="outline" className="text-xs px-1.5 py-0 h-5 flex-shrink-0 bg-blue-50 dark:bg-blue-950 border-blue-200 dark:border-blue-800">
                              {currentBranch}
                            </Badge>
                          )}
                        </div>
                        <div className="text-xs text-muted-foreground truncate">{repo.url}</div>
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-7 w-7 p-0 flex-shrink-0"
                        onClick={() => handleRemoveRepo(repoName)}
                        disabled={isRemoving}
                      >
                        {isRemoving ? (
                          <Loader2 className="h-3 w-3 animate-spin" />
                        ) : (
                          <X className="h-3 w-3" />
                        )}
                      </Button>
                    </div>

                    {/* Expandable branches list */}
                    {isExpanded && hasBranches && (
                      <div className="px-2 pb-2 pl-10 space-y-1">
                        <div className="text-xs text-muted-foreground mb-1">Available branches:</div>
                        {repo.branches!.map((branch, branchIdx) => (
                          <div
                            key={branchIdx}
                            className="text-xs py-1 px-2 rounded bg-muted/50 flex items-center gap-2"
                          >
                            <GitBranch className="h-3 w-3 text-muted-foreground" />
                            <span className="font-mono">{branch}</span>
                            {branch === currentBranch && (
                              <Badge variant="secondary" className="text-xs px-1 py-0 h-4 ml-auto">
                                active
                              </Badge>
                            )}
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                );
              })}

              {/* Uploaded Files */}
              {uploadedFiles.map((file, idx) => {
                const isRemoving = removingFile === file.name;
                const fileSizeKB = file.size ? (file.size / 1024).toFixed(1) : null;

                return (
                  <div key={`file-${idx}`} className="flex items-center gap-2 p-2 border rounded bg-muted/30 hover:bg-muted/50 transition-colors">
                    <CloudUpload className="h-4 w-4 text-blue-500 flex-shrink-0" />
                    <div className="flex-1 min-w-0">
                      <div className="text-sm font-medium truncate">{file.name}</div>
                      {fileSizeKB && (
                        <div className="text-xs text-muted-foreground">{fileSizeKB} KB</div>
                      )}
                    </div>
                    {onRemoveFile && (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-7 w-7 p-0 flex-shrink-0"
                        onClick={() => handleRemoveFile(file.name)}
                        disabled={isRemoving}
                      >
                        {isRemoving ? (
                          <Loader2 className="h-3 w-3 animate-spin" />
                        ) : (
                          <X className="h-3 w-3" />
                        )}
                      </Button>
                    )}
                  </div>
                );
              })}

              <Button onClick={onAddRepository} variant="outline" className="w-full" size="sm">
                <Link className="mr-2 h-3 w-3" />
                Add Context
              </Button>
            </div>
          )}
        </div>
      </AccordionContent>
    </AccordionItem>
  );
}

