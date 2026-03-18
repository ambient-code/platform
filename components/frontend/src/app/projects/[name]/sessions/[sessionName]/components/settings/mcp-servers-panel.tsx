"use client";

import { useState, useEffect } from "react";
import {
  CheckCircle2,
  XCircle,
  AlertCircle,
  Info,
  Check,
  X,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { CardSkeleton } from "./card-skeleton";
import { useMcpStatus } from "@/services/queries/use-mcp";
import type { McpServer, McpTool } from "@/services/api/sessions";

type McpServersPanelProps = {
  projectName: string;
  sessionName: string;
  sessionPhase?: string;
  onAddServer?: () => void;
};

export function McpServersPanel({
  projectName,
  sessionName,
  sessionPhase,
  onAddServer,
}: McpServersPanelProps) {
  const [placeholderTimedOut, setPlaceholderTimedOut] = useState(false);
  const isRunning = sessionPhase === "Running";
  const { data: mcpStatus, isPending: mcpPending } = useMcpStatus(
    projectName,
    sessionName,
    isRunning
  );
  const mcpServers = mcpStatus?.servers || [];

  const showPlaceholders =
    !isRunning ||
    mcpPending ||
    (mcpServers.length === 0 && !placeholderTimedOut);

  useEffect(() => {
    if (mcpServers.length > 0) {
      setPlaceholderTimedOut(false);
      return;
    }
    if (!isRunning || !mcpStatus) return;
    const t = setTimeout(() => setPlaceholderTimedOut(true), 15 * 1000);
    return () => clearTimeout(t);
  }, [mcpStatus, mcpServers.length, isRunning]);

  return (
    <div>
      <div className="flex items-center justify-between mb-1">
        <h3 className="text-lg font-semibold">MCP Servers</h3>
        {onAddServer && (
          <Button variant="outline" size="sm" onClick={onAddServer}>
            Add server
          </Button>
        )}
      </div>
      <p className="text-sm text-muted-foreground mb-4">
        Model Context Protocol servers connected to this session.
      </p>

      <div className="space-y-2">
        {showPlaceholders ? (
          <>
            <CardSkeleton />
            <CardSkeleton />
          </>
        ) : mcpServers.length > 0 ? (
          mcpServers.map((server) => (
            <ServerCard key={server.name} server={server} />
          ))
        ) : (
          <p className="text-sm text-muted-foreground py-4">
            No MCP servers available for this session.
          </p>
        )}
      </div>
    </div>
  );
}

function ServerCard({ server }: { server: McpServer }) {
  const tools = server.tools ?? [];
  const toolCount = tools.length;

  return (
    <div className="flex items-start justify-between gap-3 p-3 border rounded-lg bg-background/50">
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <div className="flex-shrink-0">
            <StatusIcon status={server.status} />
          </div>
          <h4 className="font-medium text-sm">{server.displayName}</h4>
          <StatusBadgeInline status={server.status} />
        </div>
        <div className="flex items-center gap-2 mt-1 ml-6">
          {server.version && (
            <span className="text-[10px] text-muted-foreground">
              v{server.version}
            </span>
          )}
          {toolCount > 0 && (
            <ToolsPopover server={server} tools={tools} toolCount={toolCount} />
          )}
        </div>
      </div>
    </div>
  );
}

function StatusIcon({ status }: { status: string }) {
  switch (status) {
    case "configured":
    case "connected":
      return <CheckCircle2 className="h-4 w-4 text-green-600" />;
    case "error":
      return <XCircle className="h-4 w-4 text-red-600" />;
    case "disconnected":
    default:
      return <AlertCircle className="h-4 w-4 text-muted-foreground" />;
  }
}

function StatusBadgeInline({ status }: { status: string }) {
  const variants: Record<string, string> = {
    configured: "bg-blue-50 text-blue-700 border-blue-200",
    connected: "bg-green-50 text-green-700 border-green-200",
    error: "bg-red-50 text-red-700 border-red-200",
  };
  const label =
    status === "configured"
      ? "Configured"
      : status === "connected"
        ? "Connected"
        : status === "error"
          ? "Error"
          : "Disconnected";
  const className =
    variants[status] || "bg-muted text-muted-foreground border-border";

  return (
    <Badge variant="outline" className={`text-xs ${className}`}>
      {label}
    </Badge>
  );
}

function ToolsPopover({
  server,
  tools,
  toolCount,
}: {
  server: McpServer;
  tools: McpTool[];
  toolCount: number;
}) {
  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          className="inline-flex items-center gap-1 text-[10px] text-muted-foreground hover:text-foreground transition-colors"
        >
          <Info className="h-3 w-3" />
          <span>
            {toolCount} {toolCount === 1 ? "tool" : "tools"}
          </span>
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-80 p-0">
        <div className="px-3 py-2.5 border-b bg-muted/30">
          <p className="text-xs font-medium">
            {server.displayName} &mdash; {toolCount}{" "}
            {toolCount === 1 ? "tool" : "tools"}
          </p>
        </div>
        <div className="max-h-72 overflow-y-auto">
          {tools.map((tool) => (
            <ToolRow key={tool.name} tool={tool} />
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
}

function ToolRow({ tool }: { tool: McpTool }) {
  const annotations = Object.entries(tool.annotations ?? {}).filter(
    ([, v]) => typeof v === "boolean"
  );
  return (
    <div className="flex items-center justify-between gap-3 px-3 py-2">
      <code className="text-xs truncate">{tool.name}</code>
      {annotations.length > 0 && (
        <div className="flex items-center gap-1 flex-shrink-0">
          {annotations.map(([k, v]) => (
            <Badge
              key={k}
              variant="outline"
              className={`text-[10px] px-1.5 py-0 font-normal gap-0.5 ${
                v
                  ? "bg-green-50 text-green-700 border-green-200 dark:bg-green-950/30 dark:text-green-400 dark:border-green-800"
                  : "bg-red-50 text-red-700 border-red-200 dark:bg-red-950/30 dark:text-red-400 dark:border-red-800"
              }`}
            >
              {v ? (
                <Check className="h-2.5 w-2.5" />
              ) : (
                <X className="h-2.5 w-2.5" />
              )}
              {k}
            </Badge>
          ))}
        </div>
      )}
    </div>
  );
}

