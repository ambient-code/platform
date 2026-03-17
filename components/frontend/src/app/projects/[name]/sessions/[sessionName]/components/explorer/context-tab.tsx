"use client";

import { useState } from "react";
import {
  GitBranch,
  X,
  Loader2,
  CloudUpload,
  ChevronDown,
  ChevronRight,
  AlertTriangle,
  Plus,
  Upload,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import type { Repository, UploadedFile } from "../../lib/types";

type ContextTabProps = {
  repositories?: Repository[];
  uploadedFiles?: UploadedFile[];
  onAddRepository: () => void;
  onUploadFile: () => void;
  onRemoveRepository: (repoName: string) => void;
  onRemoveFile?: (fileName: string) => void;
};

export function ContextTab({
  repositories = [],
  uploadedFiles = [],
  onAddRepository,
  onUploadFile,
  onRemoveRepository,
  onRemoveFile,
}: ContextTabProps) {
  const [removingRepo, setRemovingRepo] = useState<string | null>(null);
  const [removingFile, setRemovingFile] = useState<string | null>(null);
  const [expandedRepos, setExpandedRepos] = useState<Set<string>>(new Set());

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
    <div className="flex flex-col h-full overflow-y-auto">
      {/* Repositories section */}
      <div className="border-b">
        <div className="px-3 py-2 flex items-center justify-between">
          <div>
            <h4 className="text-sm font-medium">Repositories</h4>
            <p className="text-xs text-muted-foreground">
              Git repositories cloned into this session.
            </p>
          </div>
          <Button variant="ghost" size="sm" onClick={onAddRepository} className="h-7">
            <Plus className="h-3 w-3 mr-1" />
            Add
          </Button>
        </div>

        <div className="px-3 pb-3">
          {repositories.length === 0 ? (
            <div className="text-center py-4">
              <div className="inline-flex items-center justify-center w-10 h-10 rounded-full bg-muted mb-2">
                <GitBranch className="h-4 w-4 text-muted-foreground/60" />
              </div>
              <p className="text-xs text-muted-foreground mb-2">
                No repositories added
              </p>
              <Button size="sm" variant="outline" onClick={onAddRepository}>
                <GitBranch className="mr-1.5 h-3 w-3" />
                Add Repository
              </Button>
            </div>
          ) : (
            <div className="space-y-2">
              {repositories.map((repo, idx) => {
                const repoName =
                  repo.name ||
                  repo.url.split("/").pop()?.replace(".git", "") ||
                  `repo-${idx}`;
                const isRemoving = removingRepo === repoName;
                const isExpanded = expandedRepos.has(repoName);
                const currentBranch =
                  repo.currentActiveBranch || repo.branch;
                const hasBranches =
                  repo.branches && repo.branches.length > 0;

                const toggleExpanded = () => {
                  setExpandedRepos((prev) => {
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
                  <div
                    key={repo.url}
                    className="border rounded bg-muted/30"
                  >
                    <div className="flex items-center gap-2 p-2 hover:bg-muted/50 transition-colors">
                      {hasBranches ? (
                        <button
                          type="button"
                          onClick={toggleExpanded}
                          className="h-4 w-4 text-muted-foreground flex-shrink-0 hover:text-foreground cursor-pointer"
                        >
                          {isExpanded ? (
                            <ChevronDown className="h-4 w-4" />
                          ) : (
                            <ChevronRight className="h-4 w-4" />
                          )}
                        </button>
                      ) : (
                        <div className="h-4 w-4 flex-shrink-0" />
                      )}
                      <GitBranch className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <div className="text-sm font-medium truncate">
                            {repoName}
                          </div>
                          {repo.status === "Cloning" ? (
                            <Badge
                              variant="outline"
                              className="text-xs px-1.5 py-0.5 bg-yellow-50 dark:bg-yellow-950 border-yellow-300 dark:border-yellow-800 text-yellow-700 dark:text-yellow-400"
                            >
                              <Loader2 className="h-3 w-3 animate-spin mr-1" />
                              Cloning...
                            </Badge>
                          ) : repo.status === "Removing" ? (
                            <Badge
                              variant="outline"
                              className="text-xs px-1.5 py-0.5 bg-orange-50 dark:bg-orange-950 border-orange-300 dark:border-orange-800 text-orange-700 dark:text-orange-400"
                            >
                              <Loader2 className="h-3 w-3 animate-spin mr-1" />
                              Removing...
                            </Badge>
                          ) : repo.status === "Failed" ? (
                            <Badge
                              variant="outline"
                              className="text-xs px-1.5 py-0.5 bg-red-50 dark:bg-red-950 border-red-300 dark:border-red-800 text-red-700 dark:text-red-400"
                            >
                              <AlertTriangle className="h-3 w-3 mr-1" />
                              Clone failed
                            </Badge>
                          ) : currentBranch ? (
                            <Badge
                              variant="outline"
                              className="text-xs px-1.5 py-0.5 max-w-full !whitespace-normal !overflow-visible break-words bg-blue-50 dark:bg-blue-950 border-blue-200 dark:border-blue-800"
                            >
                              {currentBranch}
                            </Badge>
                          ) : null}
                        </div>
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-7 w-7 p-0 flex-shrink-0"
                        onClick={() => handleRemoveRepo(repoName)}
                        disabled={
                          isRemoving ||
                          repo.status === "Cloning" ||
                          repo.status === "Removing"
                        }
                        aria-label={`Remove ${repoName}`}
                      >
                        {isRemoving ? (
                          <Loader2 className="h-3 w-3 animate-spin" />
                        ) : (
                          <X className="h-3 w-3" />
                        )}
                      </Button>
                    </div>

                    {isExpanded && hasBranches && (
                      <div className="px-2 pb-2 pl-10 space-y-1">
                        <div className="text-xs text-muted-foreground mb-1">
                          Available branches:
                        </div>
                        {repo.branches!.map((branch, branchIdx) => (
                          <div
                            key={branchIdx}
                            className="text-xs py-1 px-2 rounded bg-muted/50 flex items-center gap-2"
                          >
                            <GitBranch className="h-3 w-3 text-muted-foreground" />
                            <span className="font-mono">{branch}</span>
                            {branch === currentBranch && (
                              <Badge
                                variant="secondary"
                                className="text-xs px-1 py-0 h-4 ml-auto"
                              >
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
            </div>
          )}
        </div>
      </div>

      {/* Uploads section */}
      <div>
        <div className="px-3 py-2 flex items-center justify-between">
          <div>
            <h4 className="text-sm font-medium">Uploads</h4>
            <p className="text-xs text-muted-foreground">
              Files uploaded to the workspace.
            </p>
          </div>
          <Button variant="ghost" size="sm" onClick={onUploadFile} className="h-7">
            <Upload className="h-3 w-3 mr-1" />
            Upload
          </Button>
        </div>

        <div className="px-3 pb-3">
          {uploadedFiles.length === 0 ? (
            <div className="text-center py-4">
              <div className="inline-flex items-center justify-center w-10 h-10 rounded-full bg-muted mb-2">
                <CloudUpload className="h-4 w-4 text-muted-foreground/60" />
              </div>
              <p className="text-xs text-muted-foreground mb-2">
                No files uploaded
              </p>
              <Button size="sm" variant="outline" onClick={onUploadFile}>
                <Upload className="mr-1.5 h-3 w-3" />
                Upload File
              </Button>
            </div>
          ) : (
            <div className="space-y-2">
              {uploadedFiles.map((file) => {
                const isRemoving = removingFile === file.name;
                const fileSizeKB = file.size
                  ? (file.size / 1024).toFixed(1)
                  : null;

                return (
                  <div
                    key={file.path || file.name}
                    className="flex items-center gap-2 p-2 border rounded bg-muted/30 hover:bg-muted/50 transition-colors"
                  >
                    <CloudUpload className="h-4 w-4 text-blue-500 flex-shrink-0" />
                    <div className="flex-1 min-w-0">
                      <div className="text-sm font-medium truncate">
                        {file.name}
                      </div>
                      {fileSizeKB && (
                        <div className="text-xs text-muted-foreground">
                          {fileSizeKB} KB
                        </div>
                      )}
                    </div>
                    {onRemoveFile && (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-7 w-7 p-0 flex-shrink-0"
                        onClick={() => handleRemoveFile(file.name)}
                        disabled={isRemoving}
                        aria-label={`Remove ${file.name}`}
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
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
