"use client";

import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Download, AlertCircle, RefreshCw } from "lucide-react";
import { useWorkspaceFile } from "@/services/queries/use-workspace";
import { toast } from "sonner";
import { triggerDownload } from "@/utils/export-chat";
import { FileContentViewer } from "@/components/file-content-viewer";

type FileViewerProps = {
  projectName: string;
  sessionName: string;
  filePath: string;
  sessionPhase?: string;
};

export function FileViewer({
  projectName,
  sessionName,
  filePath,
  sessionPhase,
}: FileViewerProps) {
  const {
    data: content,
    isLoading,
    error,
    refetch,
  } = useWorkspaceFile(projectName, sessionName, filePath, {
    // Refetch when tab is first opened
    refetchOnMount: true,
    // Only poll while actively viewing this file tab (component is mounted) AND session is running
    // Automatically stops when switching to another tab (component unmounts)
    refetchInterval: sessionPhase === "Running" ? 5000 : false,
  });

  const handleDownload = () => {
    if (!content) return;
    const fileName = filePath.split("/").pop() ?? "file";
    triggerDownload(content, fileName, "text/plain");
  };

  const handleRefresh = async () => {
    try {
      await refetch();
      toast.success("File refreshed");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to refresh file");
    }
  };

  if (isLoading) {
    return (
      <div className="flex flex-col h-full p-4 gap-3">
        <Skeleton className="h-6 w-2/3" />
        <Skeleton className="h-4 w-1/4" />
        <div className="flex-1 space-y-2 mt-2">
          {Array.from({ length: 12 }).map((_, i) => (
            <Skeleton
              key={i}
              className="h-4"
              style={{ width: `${(i * 17 % 40) + 60}%` }}
            />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-3 text-muted-foreground">
        <AlertCircle className="w-8 h-8" />
        <p className="text-sm">Failed to load file</p>
        <p className="text-xs">
          {error instanceof Error ? error.message : "Unknown error"}
        </p>
      </div>
    );
  }

  if (!content) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-3 text-muted-foreground">
        <AlertCircle className="w-8 h-8" />
        <p className="text-sm">No content available</p>
      </div>
    );
  }

  const fileName = filePath.split("/").pop() ?? "file";

  // Build the direct workspace API URL for binary files (images, PDFs)
  const encodedPath = filePath.split('/').map(encodeURIComponent).join('/');
  const fileUrl = `/api/projects/${encodeURIComponent(projectName)}/agentic-sessions/${encodeURIComponent(sessionName)}/workspace/${encodedPath}`;

  return (
    <div className="flex flex-col h-full">
      {/* File header */}
      <div className="flex items-center justify-between px-4 py-2 border-b bg-muted/30">
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-sm text-muted-foreground truncate">
            {filePath}
          </span>
        </div>
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={handleRefresh}
            disabled={isLoading}
            title="Refresh file"
          >
            <RefreshCw className="w-4 h-4" />
            <span className="sr-only">Refresh file</span>
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={handleDownload}
            disabled={!content}
            title="Download file"
          >
            <Download className="w-4 h-4" />
            <span className="sr-only">Download file</span>
          </Button>
        </div>
      </div>

      {/* File content with rich viewer */}
      <div className="flex-1 min-h-0 p-4">
        <FileContentViewer
          fileName={fileName}
          content={content}
          fileUrl={fileUrl}
          onDownload={handleDownload}
        />
      </div>
    </div>
  );
}
