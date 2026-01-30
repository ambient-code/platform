"use client";

import { useState } from "react";
import { Folder, FolderOpen, FileText, ExternalLink } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { getIdeUrl, getIdeName } from "@/utils/ide-urls";

export type FileTreeNode = {
  name: string;
  path: string;
  type: "file" | "folder";
  children?: FileTreeNode[];
  expanded?: boolean;
  sizeKb?: number;
  data?: unknown;
  branch?: string; // Git branch for repository folders
};

export type FileTreeProps = {
  nodes: FileTreeNode[];
  selectedPath?: string;
  onSelect: (node: FileTreeNode) => void;
  onToggle?: (node: FileTreeNode) => Promise<void> | void;
  className?: string;
  // Repository context for IDE integration
  repoUrl: string;
  currentBranch: string;
  gitlabInstance?: string;
};

export function FileTree({
  nodes,
  selectedPath,
  onSelect,
  onToggle,
  className,
  repoUrl,
  currentBranch,
  gitlabInstance,
}: FileTreeProps) {
  return (
    <div className={className}>
      {nodes.map((node) => (
        <FileTreeItem
          key={node.path}
          node={node}
          selectedPath={selectedPath}
          onSelect={onSelect}
          onToggle={onToggle}
          repoUrl={repoUrl}
          currentBranch={currentBranch}
          gitlabInstance={gitlabInstance}
        />
      ))}
    </div>
  );
}

type ItemProps = {
  node: FileTreeNode;
  selectedPath?: string;
  onSelect: (node: FileTreeNode) => void;
  onToggle?: (node: FileTreeNode) => Promise<void> | void;
  depth?: number;
  // Repository context for IDE integration
  repoUrl: string;
  currentBranch: string;
  gitlabInstance?: string;
};

function FileTreeItem({
  node,
  selectedPath,
  onSelect,
  onToggle,
  depth = 0,
  repoUrl,
  currentBranch,
  gitlabInstance,
}: ItemProps) {
  const [expanded, setExpanded] = useState<boolean>(node.expanded ?? true);
  const isSelected = node.path === selectedPath;

  const handleOpenInIDE = (e: React.MouseEvent) => {
    e.stopPropagation(); // Don't trigger node selection

    const ideUrl = getIdeUrl(repoUrl, currentBranch, node.path, gitlabInstance);

    if (ideUrl) {
      window.open(ideUrl, "_blank", "noopener,noreferrer");
    }
  };

  return (
    <div>
      <div
        className={`flex items-center gap-2 px-2 py-1 text-sm rounded cursor-pointer hover:bg-muted group ${
          isSelected ? "bg-muted" : ""
        }`}
        style={{ paddingLeft: `${(depth + 1) * 12}px` }}
        onClick={async () => {
          if (node.type === "folder") {
            // If folder has children, toggle expand/collapse
            if (node.children && node.children.length > 0) {
              const next = !expanded;
              setExpanded(next);
              if (next && onToggle) {
                await onToggle(node);
              }
            } else {
              // If folder has no children (flat listing), call onSelect to navigate
              onSelect(node);
            }
          } else {
            onSelect(node);
          }
        }}
      >
        {node.type === "folder" ? (
          expanded ? (
            <FolderOpen className="h-4 w-4 text-blue-600" />
          ) : (
            <Folder className="h-4 w-4 text-blue-600" />
          )
        ) : (
          <FileText className="h-4 w-4 text-muted-foreground" />
        )}

        <span className={`flex-1 ${isSelected ? "font-medium" : ""}`}>{node.name}</span>

        {node.branch && (
          <Badge
            variant="outline"
            className="text-xs px-1.5 py-0 h-5 bg-blue-50 dark:bg-blue-950 border-blue-200 dark:border-blue-800"
          >
            {node.branch}
          </Badge>
        )}

        {typeof node.sizeKb === "number" && (
          <span className="text-xs text-muted-foreground">{node.sizeKb.toFixed(1)}K</span>
        )}

        {/* IDE button - only show on hover */}
        <Button
          variant="ghost"
          size="icon"
          className="h-6 w-6 opacity-0 group-hover:opacity-100 transition-opacity"
          onClick={handleOpenInIDE}
          title={`Open in ${getIdeName(repoUrl)}`}
        >
          <ExternalLink className="h-3 w-3" />
        </Button>
      </div>

      {node.type === "folder" && expanded && node.children && node.children.length > 0 && (
        <div>
          {node.children.map((child) => (
            <FileTreeItem
              key={child.path}
              node={child}
              selectedPath={selectedPath}
              onSelect={onSelect}
              onToggle={onToggle}
              depth={depth + 1}
              repoUrl={repoUrl}
              currentBranch={currentBranch}
              gitlabInstance={gitlabInstance}
            />
          ))}
        </div>
      )}
    </div>
  );
}


