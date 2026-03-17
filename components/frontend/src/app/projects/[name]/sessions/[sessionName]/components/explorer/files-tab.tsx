"use client";

import { useMemo } from "react";
import {
  Folder,
  FolderTree,
  GitBranch,
  Sparkles,
  CloudUpload,
  FolderSync,
  Download,
  Loader2,
  Upload,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { FileTree, type FileTreeNode } from "@/components/file-tree";
import type { DirectoryOption, GitStatusSummary } from "../../lib/types";
import type { WorkspaceItem } from "@/services/api/workspace";

type FilesTabProps = {
  directoryOptions: DirectoryOption[];
  selectedDirectory: DirectoryOption;
  onDirectoryChange: (option: DirectoryOption) => void;
  files: WorkspaceItem[];
  currentSubPath: string;
  viewingFile: { path: string; content: string } | null;
  isLoadingFile: boolean;
  onFileOrFolderSelect: (node: FileTreeNode) => void;
  onNavigateBack: () => void;
  onRefresh: () => void;
  onDownloadFile: () => void;
  onUploadFile: () => void;
  onFileOpen?: (filePath: string) => void;
  gitStatus?: GitStatusSummary;
  repoBranches?: Record<string, string | undefined>;
};

export function FilesTab({
  directoryOptions,
  selectedDirectory,
  onDirectoryChange,
  files,
  currentSubPath,
  viewingFile,
  isLoadingFile,
  onFileOrFolderSelect,
  onNavigateBack,
  onRefresh,
  onDownloadFile,
  onUploadFile,
  onFileOpen,
  gitStatus,
  repoBranches,
}: FilesTabProps) {
  const fileNodes = useMemo(
    () =>
      files.map(
        (item): FileTreeNode => ({
          name: item.name,
          path: item.path,
          type: item.isDir ? "folder" : "file",
          sizeKb: item.size ? item.size / 1024 : undefined,
        }),
      ),
    [files],
  );

  const handleSelect = (node: FileTreeNode) => {
    if (node.type === "file" && onFileOpen) {
      const fullPath = currentSubPath
        ? `${selectedDirectory.path}/${currentSubPath}/${node.path}`
        : `${selectedDirectory.path}/${node.path}`;
      onFileOpen(fullPath);
    } else {
      onFileOrFolderSelect(node);
    }
  };

  return (
    <div className="flex flex-col h-full">
      {/* Directory selector */}
      <div className="px-3 py-2 border-b">
        <Select
          value={`${selectedDirectory.type}:${selectedDirectory.path}`}
          onValueChange={(value) => {
            const [type, ...pathParts] = value.split(":");
            const path = pathParts.join(":");
            const option = directoryOptions.find(
              (opt) => opt.type === type && opt.path === path,
            );
            if (option) onDirectoryChange(option);
          }}
        >
          <SelectTrigger className="w-full h-auto min-h-[2rem] py-1.5">
            <div className="flex items-center gap-2 w-full pr-4">
              <SelectValue />
            </div>
          </SelectTrigger>
          <SelectContent>
            {directoryOptions.map((opt) => {
              const branchName =
                opt.type === "repo"
                  ? repoBranches?.[opt.path.replace(/^repos\//, "")]
                  : undefined;

              return (
                <SelectItem
                  key={`${opt.type}:${opt.path}`}
                  value={`${opt.type}:${opt.path}`}
                  className="py-2"
                >
                  <div className="flex items-center gap-2 flex-wrap w-full">
                    {opt.type === "artifacts" && (
                      <Folder className="h-3 w-3" />
                    )}
                    {opt.type === "file-uploads" && (
                      <CloudUpload className="h-3 w-3" />
                    )}
                    {opt.type === "repo" && <GitBranch className="h-3 w-3" />}
                    {opt.type === "workflow" && (
                      <Sparkles className="h-3 w-3" />
                    )}
                    <span className="text-xs">{opt.name}</span>
                    {branchName && (
                      <Badge
                        variant="outline"
                        className="text-xs px-1.5 py-0.5 bg-blue-50 dark:bg-blue-950 border-blue-200 dark:border-blue-800"
                      >
                        {branchName}
                      </Badge>
                    )}
                  </div>
                </SelectItem>
              );
            })}
          </SelectContent>
        </Select>
      </div>

      {/* Action bar */}
      <div className="px-3 py-1.5 border-b flex items-center justify-between bg-muted/30">
        <div className="flex items-center gap-1 text-xs text-muted-foreground min-w-0 flex-1">
          {(currentSubPath || viewingFile) && (
            <Button
              variant="ghost"
              size="sm"
              onClick={onNavigateBack}
              className="h-6 px-1.5 mr-1"
            >
              ← Back
            </Button>
          )}
          <Folder className="inline h-3 w-3 mr-1 flex-shrink-0" />
          <code className="bg-muted px-1 py-0.5 rounded text-xs truncate">
            {selectedDirectory.path}
            {currentSubPath && `/${currentSubPath}`}
            {viewingFile && `/${viewingFile.path}`}
          </code>
        </div>

        <div className="flex items-center gap-1">
          {viewingFile ? (
            <Button
              variant="ghost"
              size="sm"
              onClick={onDownloadFile}
              className="h-6 px-2 flex-shrink-0"
              title="Download file"
            >
              <Download className="h-3 w-3" />
            </Button>
          ) : (
            <>
              <Button
                variant="ghost"
                size="sm"
                onClick={onUploadFile}
                className="h-6 px-2 flex-shrink-0"
                title="Upload file"
              >
                <Upload className="h-3 w-3" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={onRefresh}
                className="h-6 px-2 flex-shrink-0"
                title="Refresh"
              >
                <FolderSync className="h-3 w-3" />
              </Button>
            </>
          )}
        </div>
      </div>

      {/* Git status badges */}
      {gitStatus?.hasChanges && (
        <div className="px-3 py-1 flex gap-1 border-b">
          {gitStatus.totalAdded > 0 && (
            <Badge
              variant="outline"
              className="bg-green-50 text-green-700 border-green-200 dark:bg-green-950/50 dark:text-green-300 dark:border-green-800 text-xs"
            >
              +{gitStatus.totalAdded}
            </Badge>
          )}
          {gitStatus.totalRemoved > 0 && (
            <Badge
              variant="outline"
              className="bg-red-50 text-red-700 border-red-200 dark:bg-red-950/50 dark:text-red-300 dark:border-red-800 text-xs"
            >
              -{gitStatus.totalRemoved}
            </Badge>
          )}
        </div>
      )}

      {/* File tree */}
      <div className="flex-1 overflow-y-auto p-2">
        {isLoadingFile ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        ) : viewingFile ? (
          <div className="text-xs">
            <pre className="bg-muted/50 p-2 rounded overflow-x-auto">
              <code>{viewingFile.content}</code>
            </pre>
          </div>
        ) : files.length === 0 ? (
          <div className="text-center py-4 text-sm text-muted-foreground">
            <FolderTree className="h-8 w-8 mx-auto mb-2 opacity-30" />
            <p>No files yet</p>
            <p className="text-xs mt-1">Files will appear here</p>
          </div>
        ) : (
          <FileTree nodes={fileNodes} onSelect={handleSelect} />
        )}
      </div>
    </div>
  );
}
