"use client";

import { X, FolderOpen, Link, Layers } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { FilesTab } from "./files-tab";
import { ContextTab } from "./context-tab";
import { BackgroundTasksTab } from "./background-tasks-tab";
import { useProjectAccess } from "@/services/queries/use-project-access";
import type { FileTreeNode } from "@/components/file-tree";
import type { DirectoryOption, Repository, UploadedFile, GitStatusSummary } from "../../lib/types";
import type { WorkspaceItem } from "@/services/api/workspace";
import type { BackgroundTask } from "@/types/background-task";

const noop = () => {};

export type ExplorerPanelProps = {
  visible?: boolean;
  activeTab: "files" | "context" | "tasks";
  onTabChange: (tab: "files" | "context" | "tasks") => void;
  onClose: () => void;
  projectName: string;
  sessionName: string;
  // Files tab props
  directoryOptions: DirectoryOption[];
  selectedDirectory: DirectoryOption;
  onDirectoryChange: (option: DirectoryOption) => void;
  files: WorkspaceItem[];
  currentSubPath: string;
  viewingFile: { path: string; content: string; size?: number } | null;
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
  // Background tasks tab props
  backgroundTasks?: Map<string, BackgroundTask>;
  onOpenTranscript?: (task: BackgroundTask) => void;
};

export function ExplorerPanel({
  activeTab,
  onTabChange,
  onClose,
  projectName,
  sessionName,
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
  // Background tasks tab
  backgroundTasks,
  onOpenTranscript,
}: ExplorerPanelProps) {
  const { data: access } = useProjectAccess(projectName);
  const canModify = !!access?.userRole && access.userRole !== 'view';
  const runningTaskCount = backgroundTasks
    ? Array.from(backgroundTasks.values()).filter((t) => t.status === "running").length
    : 0;

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
          <button
            type="button"
            onClick={() => onTabChange("tasks")}
            className={cn(
              "px-3 py-2 text-sm font-medium transition-colors flex items-center gap-1.5",
              activeTab === "tasks"
                ? "text-foreground border-b-2 border-primary"
                : "text-muted-foreground hover:text-foreground",
            )}
          >
            <Layers className="h-3.5 w-3.5" />
            Tasks
            {runningTaskCount > 0 && (
              <Badge variant="secondary" className="h-4 min-w-4 px-1 text-[10px] leading-none">
                {runningTaskCount}
              </Badge>
            )}
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
            canModify={canModify}
          />
        ) : activeTab === "tasks" ? (
          <BackgroundTasksTab
            backgroundTasks={backgroundTasks ?? new Map()}
            projectName={projectName}
            sessionName={sessionName}
            onOpenTranscript={onOpenTranscript ?? noop}
          />
        ) : (
          <ContextTab
            repositories={repositories}
            uploadedFiles={uploadedFiles}
            onAddRepository={onAddRepository}
            onUploadFile={onUploadFile}
            onRemoveRepository={onRemoveRepository}
            onRemoveFile={onRemoveFile}
            canModify={canModify}
          />
        )}
      </div>
    </div>
  );
}
