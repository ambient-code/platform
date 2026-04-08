"use client";

import { useState, useEffect } from "react";
import {
  CheckCircle2,
  XCircle,
  AlertCircle,
  Info,
  Check,
  X,
  Plus,
  Trash2,
  Power,
  PowerOff,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { CardSkeleton } from "./card-skeleton";
import { useMcpStatus, useUpdateSessionMcpServers } from "@/services/queries/use-mcp";
import type { McpServer, McpTool } from "@/services/api/sessions";
import type { AgenticSession, MCPServersConfig } from "@/types/agentic-session";
import { toast } from "sonner";

type McpServersPanelProps = {
  projectName: string;
  sessionName: string;
  sessionPhase?: string;
  session?: AgenticSession;
};

export function McpServersPanel({
  projectName,
  sessionName,
  sessionPhase,
  session,
}: McpServersPanelProps) {
  const [placeholderTimedOut, setPlaceholderTimedOut] = useState(false);
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const isRunning = sessionPhase === "Running";
  const { data: mcpStatus, isPending: mcpPending } = useMcpStatus(
    projectName,
    sessionName,
    isRunning
  );
  const updateMcp = useUpdateSessionMcpServers(projectName, sessionName);
  const mcpServers = mcpStatus?.servers || [];

  // Current MCP config from session spec
  const currentConfig: MCPServersConfig = session?.spec?.mcpServers || {};
  const customServers = currentConfig.custom || {};
  const disabledServers = currentConfig.disabled || [];

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

  const handleToggleServer = (serverName: string, enabled: boolean) => {
    const newDisabled = enabled
      ? disabledServers.filter((n) => n !== serverName)
      : [...disabledServers, serverName];
    updateMcp.mutate(
      { ...currentConfig, disabled: newDisabled },
      {
        onSuccess: () =>
          toast.success(
            `${serverName} will be ${enabled ? "enabled" : "disabled"} on next restart`
          ),
        onError: () => toast.error("Failed to update MCP configuration"),
      }
    );
  };

  const handleRemoveServer = (serverName: string) => {
    const newCustom = { ...customServers };
    delete newCustom[serverName];
    updateMcp.mutate(
      { ...currentConfig, custom: newCustom },
      {
        onSuccess: () =>
          toast.success(
            `${serverName} removed (takes effect on next restart)`
          ),
        onError: () => toast.error("Failed to remove MCP server"),
      }
    );
  };

  const handleAddServer = (
    name: string,
    config: Record<string, unknown>
  ) => {
    const newCustom = { ...customServers, [name]: config };
    updateMcp.mutate(
      { ...currentConfig, custom: newCustom },
      {
        onSuccess: () => {
          toast.success(
            `${name} added (takes effect on next restart)`
          );
          setAddDialogOpen(false);
        },
        onError: () => toast.error("Failed to add MCP server"),
      }
    );
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-1">
        <h3 className="text-lg font-semibold">MCP Servers</h3>
        <Button
          variant="outline"
          size="sm"
          onClick={() => setAddDialogOpen(true)}
        >
          <Plus className="h-3.5 w-3.5 mr-1" />
          Add server
        </Button>
      </div>
      <p className="text-sm text-muted-foreground mb-4">
        Model Context Protocol servers connected to this session.
        {isRunning && " Changes take effect on session restart."}
      </p>

      {/* Custom servers configured but not yet active */}
      {Object.keys(customServers).length > 0 && (
        <div className="mb-4">
          <h4 className="text-xs uppercase tracking-wide text-muted-foreground mb-2">
            Custom Servers
          </h4>
          <div className="space-y-2">
            {Object.entries(customServers).map(([name, config]) => {
              const liveServer = mcpServers.find((s) => s.name === name);
              return (
                <CustomServerCard
                  key={name}
                  name={name}
                  config={config}
                  liveServer={liveServer}
                  onRemove={() => handleRemoveServer(name)}
                />
              );
            })}
          </div>
        </div>
      )}

      {/* Disabled servers */}
      {disabledServers.length > 0 && (
        <div className="mb-4">
          <h4 className="text-xs uppercase tracking-wide text-muted-foreground mb-2">
            Disabled Servers
          </h4>
          <div className="space-y-2">
            {disabledServers.map((name) => (
              <div
                key={name}
                className="flex items-center justify-between gap-3 p-3 border rounded-lg bg-muted/30 opacity-60"
              >
                <div className="flex items-center gap-2">
                  <PowerOff className="h-4 w-4 text-muted-foreground" />
                  <span className="font-medium text-sm">{name}</span>
                  <Badge variant="outline" className="text-xs">
                    Disabled
                  </Badge>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleToggleServer(name, true)}
                >
                  <Power className="h-3.5 w-3.5 mr-1" />
                  Enable
                </Button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Live active servers */}
      <div>
        {(Object.keys(customServers).length > 0 || disabledServers.length > 0) && (
          <h4 className="text-xs uppercase tracking-wide text-muted-foreground mb-2">
            Active Servers
          </h4>
        )}
        <div className="space-y-2">
          {showPlaceholders ? (
            <>
              <CardSkeleton />
              <CardSkeleton />
            </>
          ) : mcpServers.length > 0 ? (
            mcpServers
              .filter((s) => !customServers[s.name]) // Don't duplicate custom servers shown above
              .map((server) => (
                <ServerCard
                  key={server.name}
                  server={server}
                  isDisableable={!["session", "corrections", "acp", "rubric"].includes(server.name)}
                  onDisable={() => handleToggleServer(server.name, false)}
                />
              ))
          ) : (
            <p className="text-sm text-muted-foreground py-4">
              No MCP servers available for this session.
            </p>
          )}
        </div>
      </div>

      <AddServerDialog
        open={addDialogOpen}
        onOpenChange={setAddDialogOpen}
        onAdd={handleAddServer}
        existingNames={[...mcpServers.map((s) => s.name), ...Object.keys(customServers)]}
      />
    </div>
  );
}

function CustomServerCard({
  name,
  config,
  liveServer,
  onRemove,
}: {
  name: string;
  config: Record<string, unknown>;
  liveServer?: McpServer;
  onRemove: () => void;
}) {
  const serverType = (config.type as string) || (config.url ? "http" : "stdio");
  return (
    <div className="flex items-start justify-between gap-3 p-3 border rounded-lg bg-background/50">
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <div className="flex-shrink-0">
            {liveServer ? (
              <StatusIcon status={liveServer.status} />
            ) : (
              <AlertCircle className="h-4 w-4 text-amber-500" />
            )}
          </div>
          <h4 className="font-medium text-sm">{name}</h4>
          <Badge variant="outline" className="text-xs bg-blue-50 text-blue-700 border-blue-200">
            Custom
          </Badge>
          <Badge variant="outline" className="text-xs">
            {serverType}
          </Badge>
        </div>
        <div className="flex items-center gap-2 mt-1 ml-6">
          {serverType === "http" || serverType === "sse" ? (
            <span className="text-[10px] text-muted-foreground truncate max-w-[250px]">
              {config.url as string}
            </span>
          ) : (
            <span className="text-[10px] text-muted-foreground truncate max-w-[250px]">
              {config.command as string}{" "}
              {Array.isArray(config.args) ? (config.args as string[]).join(" ") : ""}
            </span>
          )}
          {liveServer?.tools && liveServer.tools.length > 0 && (
            <ToolsPopover
              server={liveServer}
              tools={liveServer.tools}
              toolCount={liveServer.tools.length}
            />
          )}
        </div>
      </div>
      <Button
        variant="ghost"
        size="icon"
        className="h-7 w-7 text-muted-foreground hover:text-destructive"
        onClick={onRemove}
      >
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  );
}

function ServerCard({
  server,
  isDisableable,
  onDisable,
}: {
  server: McpServer;
  isDisableable: boolean;
  onDisable: () => void;
}) {
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
      {isDisableable && (
        <Button
          variant="ghost"
          size="sm"
          className="text-muted-foreground hover:text-destructive text-xs"
          onClick={onDisable}
        >
          <PowerOff className="h-3 w-3 mr-1" />
          Disable
        </Button>
      )}
    </div>
  );
}

function AddServerDialog({
  open,
  onOpenChange,
  onAdd,
  existingNames,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onAdd: (name: string, config: Record<string, unknown>) => void;
  existingNames: string[];
}) {
  const [name, setName] = useState("");
  const [type, setType] = useState<"http" | "sse" | "stdio">("http");
  const [url, setUrl] = useState("");
  const [command, setCommand] = useState("");
  const [args, setArgs] = useState("");

  const resetForm = () => {
    setName("");
    setType("http");
    setUrl("");
    setCommand("");
    setArgs("");
  };

  const handleSubmit = () => {
    if (!name.trim()) return;
    if (existingNames.includes(name.trim())) {
      toast.error("A server with this name already exists");
      return;
    }

    let config: Record<string, unknown>;
    if (type === "http" || type === "sse") {
      if (!url.trim()) return;
      config = { type, url: url.trim() };
    } else {
      if (!command.trim()) return;
      config = {
        type: "stdio",
        command: command.trim(),
        args: args
          .split(/\s+/)
          .filter(Boolean),
      };
    }
    onAdd(name.trim(), config);
    resetForm();
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        onOpenChange(v);
        if (!v) resetForm();
      }}
    >
      <DialogContent className="sm:max-w-[480px]">
        <DialogHeader>
          <DialogTitle>Add MCP Server</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 py-2">
          <div className="space-y-2">
            <Label htmlFor="server-name">Server Name</Label>
            <Input
              id="server-name"
              placeholder="e.g. my-custom-server"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label>Type</Label>
            <Select value={type} onValueChange={(v) => setType(v as "http" | "sse" | "stdio")}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="http">HTTP</SelectItem>
                <SelectItem value="sse">SSE</SelectItem>
                <SelectItem value="stdio">Stdio (command)</SelectItem>
              </SelectContent>
            </Select>
          </div>
          {type === "http" || type === "sse" ? (
            <div className="space-y-2">
              <Label htmlFor="server-url">URL</Label>
              <Input
                id="server-url"
                placeholder="https://example.com/mcp"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
              />
            </div>
          ) : (
            <>
              <div className="space-y-2">
                <Label htmlFor="server-command">Command</Label>
                <Input
                  id="server-command"
                  placeholder="e.g. uvx"
                  value={command}
                  onChange={(e) => setCommand(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="server-args">
                  Arguments <span className="text-muted-foreground">(space-separated)</span>
                </Label>
                <Input
                  id="server-args"
                  placeholder="e.g. mcp-server-fetch"
                  value={args}
                  onChange={(e) => setArgs(e.target.value)}
                />
              </div>
            </>
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={
              !name.trim() ||
              (type !== "stdio" && !url.trim()) ||
              (type === "stdio" && !command.trim())
            }
          >
            Add Server
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
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
