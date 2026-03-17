"use client";

import { X, FolderOpen, Link } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { FilesTab } from "./files-tab";
import { ContextTab } from "./context-tab";
import type { FileTreeNode } from "@/components/file-tree";
import type { DirectoryOption, Repository, UploadedFile, GitStatusSummary } from "../../lib/types";
import type { WorkspaceItem } from "@/services/api/workspace";

type ExplorerPanelProps = {
  visible?: boolean;
  activeTab: "files" | "context";
  onTabChange: (tab: "files" | "context") => void;
  onClose: () => void;
  // Files tab props
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
  // Context tab props
  repositories?: Repository[];
  uploadedFiles?: UploadedFile[];
  onAddRepository: () => void;
  onRemoveRepository: (repoName: string) => void;
  onRemoveFile?: (fileName: string) => void;
};

export function ExplorerPanel({
  activeTab,
  onTabChange,
  onClose,
  // Files tab
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
  // Context tab
  repositories,
  uploadedFiles,
  onAddRepository,
  onRemoveRepository,
  onRemoveFile,
}: ExplorerPanelProps) {
  return (
    <div className="flex flex-col h-full border-l bg-background overflow-hidden">
      {/* Tab header */}
      <div className="flex items-center justify-between border-b px-1">
        <div className="flex">
          <button
            type="button"
            onClick={() => onTabChange("files")}
            className={cn(
              "px-3 py-2 text-sm font-medium transition-colors flex items-center gap-1.5",
              activeTab === "files"
                ? "text-foreground border-b-2 border-primary"
                : "text-muted-foreground hover:text-foreground",
            )}
          >
            <FolderOpen className="h-3.5 w-3.5" />
            Files
          </button>
          <button
            type="button"
            onClick={() => onTabChange("context")}
            className={cn(
              "px-3 py-2 text-sm font-medium transition-colors flex items-center gap-1.5",
              activeTab === "context"
                ? "text-foreground border-b-2 border-primary"
                : "text-muted-foreground hover:text-foreground",
            )}
          >
            <Link className="h-3.5 w-3.5" />
            Context
          </button>
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={onClose}
          className="h-7 w-7 p-0 mr-1"
        >
          <X className="h-4 w-4" />
        </Button>
      </div>

      {/* Tab content */}
      <div className="flex-1 overflow-hidden">
        {activeTab === "files" ? (
          <FilesTab
            directoryOptions={directoryOptions}
            selectedDirectory={selectedDirectory}
            onDirectoryChange={onDirectoryChange}
            files={files}
            currentSubPath={currentSubPath}
            viewingFile={viewingFile}
            isLoadingFile={isLoadingFile}
            onFileOrFolderSelect={onFileOrFolderSelect}
            onNavigateBack={onNavigateBack}
            onRefresh={onRefresh}
            onDownloadFile={onDownloadFile}
            onUploadFile={onUploadFile}
            onFileOpen={onFileOpen}
            gitStatus={gitStatus}
            repoBranches={repoBranches}
          />
        ) : (
          <ContextTab
            repositories={repositories}
            uploadedFiles={uploadedFiles}
            onAddRepository={onAddRepository}
            onUploadFile={onUploadFile}
            onRemoveRepository={onRemoveRepository}
            onRemoveFile={onRemoveFile}
          />
        )}
      </div>
    </div>
  );
}
