"use client";

import { useState } from "react";
import { GitBranch, X, Link, Loader2, CloudUpload } from "lucide-react";
import { AccordionItem, AccordionTrigger, AccordionContent } from "@/components/ui/accordion";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import type { SessionRepo } from "@/types/agentic-session";
import { getRepoDisplayName } from "@/utils/repo";

type UploadedFile = {
  name: string;
  path: string;
  size?: number;
};

type RepositoriesAccordionProps = {
  repositories?: SessionRepo[];
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
                const repoName = getRepoDisplayName(repo, idx);
                const repoUrl = repo.input.url;
                const isRemoving = removingRepo === repoName;

                return (
                  <div key={`repo-${idx}`} className="flex items-center gap-2 p-2 border rounded bg-muted/30 hover:bg-muted/50 transition-colors">
                    <GitBranch className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                    <div className="flex-1 min-w-0">
                      <div className="text-sm font-medium truncate">{repoName}</div>
                      <div className="text-xs text-muted-foreground truncate">{repoUrl}</div>
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
