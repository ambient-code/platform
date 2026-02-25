"use client";

import { useRef, useState } from "react";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Plus, MoreHorizontal, Pencil, Trash2, Server, Zap, Download, Upload, ChevronDown, AlertCircle } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { successToast, errorToast } from "@/hooks/use-toast";
import { useMcpConfig, useUpdateMcpConfig, useTestMcpServer } from "@/services/queries/use-mcp-config";
import { McpServerDialog } from "@/components/mcp-server-dialog";
import type { McpServerConfig } from "@/services/api/mcp-config";

// OpenCode local server format: {type: "local", command: ["cmd", ...args], environment?: {...}}
type OpenCodeServer = {
  type: string;
  command: string[];
  environment?: Record<string, string>;
  enabled?: boolean;
};

function toInternal(servers: Record<string, unknown>): Record<string, McpServerConfig> {
  const result: Record<string, McpServerConfig> = {};
  for (const [name, raw] of Object.entries(servers)) {
    if (!raw || typeof raw !== "object") continue;
    const srv = raw as Record<string, unknown>;
    if (Array.isArray(srv.command) && srv.command.every((c) => typeof c === "string")) {
      // OpenCode format: command is an array of strings
      const [cmd = "", ...args] = srv.command as string[];
      const env = srv.environment && typeof srv.environment === "object" ? (srv.environment as Record<string, string>) : {};
      result[name] = { command: cmd, args, env };
    } else if (typeof srv.command === "string") {
      // Claude Code format: command is a string, args is separate
      const args = Array.isArray(srv.args) ? (srv.args as string[]) : [];
      const env = srv.env && typeof srv.env === "object" ? (srv.env as Record<string, string>) : {};
      result[name] = { command: srv.command, args, env };
    }
  }
  return result;
}

function toOpenCode(servers: Record<string, McpServerConfig>): Record<string, OpenCodeServer> {
  const result: Record<string, OpenCodeServer> = {};
  for (const [name, srv] of Object.entries(servers)) {
    result[name] = {
      type: "local",
      command: [srv.command, ...srv.args],
      ...(Object.keys(srv.env).length > 0 ? { environment: srv.env } : {}),
    };
  }
  return result;
}

function downloadJson(data: unknown, filename: string) {
  const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
  const blobUrl = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = blobUrl;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(blobUrl);
}

type McpServersTabProps = {
  projectName: string;
};

export function McpServersTab({ projectName }: McpServersTabProps) {
  const { data: config, isLoading, error } = useMcpConfig(projectName);
  const updateMutation = useUpdateMcpConfig();
  const testMutation = useTestMcpServer();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingServer, setEditingServer] = useState<{ name: string; config: McpServerConfig } | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const servers = config?.servers ?? {};
  const serverEntries = Object.entries(servers);

  const handleAdd = () => {
    setEditingServer(null);
    setDialogOpen(true);
  };

  const handleEdit = (name: string, serverConfig: McpServerConfig) => {
    setEditingServer({ name, config: serverConfig });
    setDialogOpen(true);
  };

  const handleDelete = (name: string) => {
    const updated = { ...servers };
    delete updated[name];
    updateMutation.mutate(
      { projectName, config: { servers: updated } },
      {
        onSuccess: () => successToast(`Removed MCP server "${name}"`),
        onError: () => errorToast("Failed to remove MCP server"),
      }
    );
  };

  const handleSave = (name: string, serverConfig: McpServerConfig) => {
    const updated = { ...servers, [name]: serverConfig };
    updateMutation.mutate(
      { projectName, config: { servers: updated } },
      {
        onSuccess: () => {
          successToast(editingServer ? `Updated MCP server "${name}"` : `Added MCP server "${name}"`);
          setDialogOpen(false);
          setEditingServer(null);
        },
        onError: () => errorToast("Failed to save MCP server"),
      }
    );
  };

  const handleTest = (name: string, srv: McpServerConfig) => {
    testMutation.mutate(
      { projectName, config: srv },
      {
        onSuccess: (result) => {
          if (result.valid) {
            const info = result.serverInfo;
            const detail = info?.name ? `${info.name}${info.version ? ` v${info.version}` : ''}` : 'OK';
            successToast(`Server "${name}" is working — ${detail}`);
          } else {
            errorToast(`Server "${name}" failed: ${result.error || 'Unknown error'}`);
          }
        },
        onError: (error) => {
          errorToast(`Server "${name}" test error: ${error instanceof Error ? error.message : 'Request failed'}`);
        },
      }
    );
  };

  const handleExportClaudeCode = () => {
    downloadJson({ mcpServers: servers }, "mcp-servers.json");
    successToast(`Exported ${serverEntries.length} server(s) (Claude Code format)`);
  };

  const handleExportOpenCode = () => {
    downloadJson({ mcp: toOpenCode(servers) }, "opencode.json");
    successToast(`Exported ${serverEntries.length} server(s) (OpenCode format)`);
  };

  const handleImportClick = () => {
    fileInputRef.current?.click();
  };

  const handleImportFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    // Reset so the same file can be re-imported
    e.target.value = "";
    try {
      const text = await file.text();
      const data = JSON.parse(text);
      // Accept: Claude Code {"mcpServers": {...}}, native {"servers": {...}}, OpenCode {"mcp": {...}}
      const raw: Record<string, unknown> | undefined = data.mcpServers ?? data.servers ?? data.mcp;
      if (!raw || typeof raw !== "object") {
        errorToast("Invalid MCP config file — must contain 'mcpServers', 'servers', or 'mcp'");
        return;
      }
      const imported = toInternal(raw);
      const merged = { ...servers, ...imported };
      const count = Object.keys(imported).length;
      updateMutation.mutate(
        { projectName, config: { servers: merged } },
        {
          onSuccess: () => successToast(`Imported ${count} server(s)`),
          onError: () => errorToast("Failed to import MCP servers"),
        }
      );
    } catch {
      errorToast("Could not parse the selected file as JSON");
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-3 py-4">
        <Skeleton className="h-8 w-full" />
        <Skeleton className="h-8 w-full" />
        <Skeleton className="h-8 w-3/4" />
      </div>
    );
  }

  if (error) {
    return (
      <Alert variant="destructive">
        <AlertCircle className="h-4 w-4" />
        <AlertDescription>
          Failed to load MCP server configuration: {error instanceof Error ? error.message : "Unknown error"}
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <>
      <div className="flex justify-end gap-2 mb-4">
        <Button variant="outline" size="sm" onClick={handleImportClick}>
          <Upload className="w-4 h-4 mr-2" /> Import
        </Button>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm" disabled={serverEntries.length === 0}>
              <Download className="w-4 h-4 mr-2" /> Export <ChevronDown className="w-3 h-3 ml-1" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuLabel>Export format</DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={handleExportClaudeCode}>
              Claude Code / Desktop
            </DropdownMenuItem>
            <DropdownMenuItem onClick={handleExportOpenCode}>
              OpenCode
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        <Button onClick={handleAdd} size="sm" disabled={updateMutation.isPending}>
          <Plus className="w-4 h-4 mr-2" /> Add Server
        </Button>
        <input ref={fileInputRef} type="file" accept=".json" className="hidden" onChange={handleImportFile} />
      </div>

      {serverEntries.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-12 text-center text-muted-foreground">
          <Server className="w-10 h-10 mb-3" />
          <p className="text-sm font-medium">No MCP servers configured</p>
          <p className="text-xs mt-1">Add an MCP server to extend your session capabilities</p>
        </div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Command</TableHead>
              <TableHead>Args</TableHead>
              <TableHead>Env</TableHead>
              <TableHead className="w-12" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {serverEntries.map(([name, srv]) => (
              <TableRow key={name}>
                <TableCell className="font-medium">{name}</TableCell>
                <TableCell className="font-mono text-xs">{srv.command}</TableCell>
                <TableCell>
                  {srv.args?.length > 0 ? (
                    <span className="font-mono text-xs text-muted-foreground">{srv.args.join(", ")}</span>
                  ) : (
                    <span className="text-xs text-muted-foreground">--</span>
                  )}
                </TableCell>
                <TableCell>
                  {Object.keys(srv.env ?? {}).length > 0 ? (
                    <Badge variant="secondary">{Object.keys(srv.env).length} vars</Badge>
                  ) : (
                    <span className="text-xs text-muted-foreground">--</span>
                  )}
                </TableCell>
                <TableCell>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="ghost" size="icon" className="h-8 w-8">
                        <MoreHorizontal className="w-4 h-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem onClick={() => handleTest(name, srv)}>
                        <Zap className="w-4 h-4 mr-2" /> Test
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => handleEdit(name, srv)}>
                        <Pencil className="w-4 h-4 mr-2" /> Edit
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => handleDelete(name)} className="text-destructive">
                        <Trash2 className="w-4 h-4 mr-2" /> Delete
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      <McpServerDialog
        open={dialogOpen}
        onOpenChange={(open) => {
          setDialogOpen(open);
          if (!open) setEditingServer(null);
        }}
        onSave={handleSave}
        saving={updateMutation.isPending}
        projectName={projectName}
        initialName={editingServer?.name}
        initialConfig={editingServer?.config}
      />
    </>
  );
}
