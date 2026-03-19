"use client";

import { useEffect, useMemo, useRef } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Download, AlertCircle, RefreshCw } from "lucide-react";
import { useWorkspaceFile } from "@/services/queries/use-workspace";
import { toast } from "sonner";
import { triggerDownload } from "@/utils/export-chat";
import { cn } from "@/lib/utils";
import hljs from "highlight.js";

type FileViewerProps = {
  projectName: string;
  sessionName: string;
  filePath: string;
  sessionPhase?: string;
};

const EXTENSION_TO_LANGUAGE: Record<string, string> = {
  ts: "typescript",
  tsx: "typescript",
  js: "javascript",
  jsx: "javascript",
  py: "python",
  go: "go",
  rs: "rust",
  rb: "ruby",
  java: "java",
  kt: "kotlin",
  sql: "sql",
  sh: "bash",
  bash: "bash",
  zsh: "bash",
  yml: "yaml",
  yaml: "yaml",
  json: "json",
  md: "markdown",
  css: "css",
  scss: "scss",
  html: "html",
  xml: "xml",
  toml: "toml",
  dockerfile: "dockerfile",
  makefile: "makefile",
  tf: "hcl",
  proto: "protobuf",
};

function getLanguage(filePath: string): string {
  const fileName = filePath.split("/").pop() ?? "";
  const lowerName = fileName.toLowerCase();

  // Handle special filenames
  if (lowerName === "dockerfile") return "dockerfile";
  if (lowerName === "makefile" || lowerName === "gnumakefile") return "makefile";

  const ext = fileName.split(".").pop()?.toLowerCase() ?? "";
  return EXTENSION_TO_LANGUAGE[ext] ?? "";
}

export function FileViewer({
  projectName,
  sessionName,
  filePath,
  sessionPhase,
}: FileViewerProps) {
  const codeRef = useRef<HTMLElement>(null);
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

  const { language, languageLabel } = useMemo(() => {
    const lang = getLanguage(filePath);
    const label = lang || (filePath.split(".").pop()?.toLowerCase() ?? "text");
    return { language: lang, languageLabel: label };
  }, [filePath]);

  useEffect(() => {
    if (codeRef.current && content !== undefined) {
      // Reset previous highlighting
      codeRef.current.removeAttribute("data-highlighted");
      if (language) {
        codeRef.current.className = `language-${language}`;
      } else {
        codeRef.current.className = "";
      }
      hljs.highlightElement(codeRef.current);
    }
  }, [content, language]);

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

  const lines = content?.split("\n") ?? [];

  return (
    <div className="flex flex-col h-full">
      {/* File header */}
      <div className="flex items-center justify-between px-4 py-2 border-b bg-muted/30">
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-sm text-muted-foreground truncate">
            {filePath}
          </span>
          <Badge variant="secondary" className="text-xs flex-shrink-0">
            {languageLabel}
          </Badge>
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

      {/* Code content */}
      <div className="flex-1 overflow-auto">
        <div className="flex text-sm font-mono">
          {/* Line numbers */}
          <div
            className={cn(
              "select-none text-right pr-3 pl-3 py-3",
              "text-muted-foreground/50 border-r bg-muted/20"
            )}
            aria-hidden
          >
            {lines.map((_, i) => (
              <div key={i} className="leading-6">
                {i + 1}
              </div>
            ))}
          </div>

          {/* Code */}
          <pre className="flex-1 py-3 pl-4 pr-4 overflow-x-auto m-0 bg-transparent">
            <code ref={codeRef} className={language ? `language-${language}` : ""}>
              {content}
            </code>
          </pre>
        </div>
      </div>
    </div>
  );
}
