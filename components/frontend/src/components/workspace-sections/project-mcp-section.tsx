"use client";

import { useState } from "react";
import {
  Plus,
  Trash2,
  Plug,
  Loader2,
} from "lucide-react";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { toast } from "sonner";
import {
  useProjectMcpServers,
  useUpdateProjectMcpServers,
} from "@/services/queries/use-projects";
import type { MCPServersConfig } from "@/types/agentic-session";

type ProjectMcpSectionProps = {
  projectName: string;
};

type ServerEntry = {
  name: string;
  type: "http" | "sse" | "stdio";
  url?: string;
  command?: string;
  args?: string;
};

export function ProjectMcpSection({ projectName }: ProjectMcpSectionProps) {
  const { data: mcpConfig, isLoading } = useProjectMcpServers(projectName);
  const updateMcp = useUpdateProjectMcpServers(projectName);

  const [newServer, setNewServer] = useState<ServerEntry>({
    name: "",
    type: "http",
    url: "",
    command: "",
    args: "",
  });
  const [disabledInput, setDisabledInput] = useState("");

  const customServers = mcpConfig?.custom || {};
  const disabledServers = mcpConfig?.disabled || [];

  const handleAddServer = () => {
    if (!newServer.name.trim()) return;
    if (customServers[newServer.name.trim()]) {
      toast.error("A server with this name already exists");
      return;
    }

    let config: Record<string, unknown>;
    if (newServer.type === "http" || newServer.type === "sse") {
      if (!newServer.url?.trim()) return;
      config = { type: newServer.type, url: newServer.url.trim() };
    } else {
      if (!newServer.command?.trim()) return;
      config = {
        type: "stdio",
        command: newServer.command.trim(),
        args: (newServer.args || "").split(/\s+/).filter(Boolean),
      };
    }

    const updated: MCPServersConfig = {
      ...mcpConfig,
      custom: { ...customServers, [newServer.name.trim()]: config },
    };
    updateMcp.mutate(updated, {
      onSuccess: () => {
        toast.success(`Added MCP server: ${newServer.name}`);
        setNewServer({ name: "", type: "http", url: "", command: "", args: "" });
      },
      onError: () => toast.error("Failed to add MCP server"),
    });
  };

  const handleRemoveServer = (name: string) => {
    const newCustom = { ...customServers };
    delete newCustom[name];
    updateMcp.mutate(
      { ...mcpConfig, custom: newCustom },
      {
        onSuccess: () => toast.success(`Removed MCP server: ${name}`),
        onError: () => toast.error("Failed to remove MCP server"),
      }
    );
  };

  const handleAddDisabled = () => {
    if (!disabledInput.trim()) return;
    if (disabledServers.includes(disabledInput.trim())) return;
    updateMcp.mutate(
      {
        ...mcpConfig,
        disabled: [...disabledServers, disabledInput.trim()],
      },
      {
        onSuccess: () => {
          toast.success(`Disabled default server: ${disabledInput}`);
          setDisabledInput("");
        },
        onError: () => toast.error("Failed to update disabled servers"),
      }
    );
  };

  const handleRemoveDisabled = (name: string) => {
    updateMcp.mutate(
      {
        ...mcpConfig,
        disabled: disabledServers.filter((n) => n !== name),
      },
      {
        onSuccess: () => toast.success(`Re-enabled server: ${name}`),
        onError: () => toast.error("Failed to update disabled servers"),
      }
    );
  };

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Plug className="h-5 w-5" />
            MCP Servers
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin" />
            Loading...
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Plug className="h-5 w-5" />
          MCP Servers
        </CardTitle>
        <CardDescription>
          Configure default MCP servers for all sessions in this project.
          Session-level configuration takes precedence.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Custom Servers */}
        <div className="space-y-3">
          <Label className="text-base font-semibold">Custom Servers</Label>
          <div className="text-xs text-muted-foreground">
            These servers will be available in all new sessions.
          </div>

          {Object.entries(customServers).length > 0 && (
            <div className="space-y-2">
              {Object.entries(customServers).map(([name, config]) => {
                const cfg = config as Record<string, unknown>;
                const serverType = (cfg.type as string) || "http";
                return (
                  <div
                    key={name}
                    className="flex items-center justify-between gap-2 p-3 border rounded-lg"
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      <span className="font-medium text-sm">{name}</span>
                      <Badge variant="outline" className="text-xs">
                        {serverType}
                      </Badge>
                      <span className="text-xs text-muted-foreground truncate">
                        {serverType === "stdio"
                          ? `${cfg.command}`
                          : `${cfg.url}`}
                      </span>
                    </div>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7 text-muted-foreground hover:text-destructive"
                      onClick={() => handleRemoveServer(name)}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                );
              })}
            </div>
          )}

          {/* Add server form */}
          <div className="space-y-3 p-3 border rounded-lg bg-muted/30">
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <Label className="text-xs">Name</Label>
                <Input
                  placeholder="my-server"
                  value={newServer.name}
                  onChange={(e) =>
                    setNewServer((prev) => ({ ...prev, name: e.target.value }))
                  }
                />
              </div>
              <div className="space-y-1">
                <Label className="text-xs">Type</Label>
                <Select
                  value={newServer.type}
                  onValueChange={(v) =>
                    setNewServer((prev) => ({
                      ...prev,
                      type: v as "http" | "sse" | "stdio",
                    }))
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="http">HTTP</SelectItem>
                    <SelectItem value="sse">SSE</SelectItem>
                    <SelectItem value="stdio">Stdio</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            {newServer.type === "http" || newServer.type === "sse" ? (
              <div className="space-y-1">
                <Label className="text-xs">URL</Label>
                <Input
                  placeholder="https://example.com/mcp"
                  value={newServer.url}
                  onChange={(e) =>
                    setNewServer((prev) => ({ ...prev, url: e.target.value }))
                  }
                />
              </div>
            ) : (
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <Label className="text-xs">Command</Label>
                  <Input
                    placeholder="uvx"
                    value={newServer.command}
                    onChange={(e) =>
                      setNewServer((prev) => ({
                        ...prev,
                        command: e.target.value,
                      }))
                    }
                  />
                </div>
                <div className="space-y-1">
                  <Label className="text-xs">Arguments</Label>
                  <Input
                    placeholder="mcp-server-fetch"
                    value={newServer.args}
                    onChange={(e) =>
                      setNewServer((prev) => ({
                        ...prev,
                        args: e.target.value,
                      }))
                    }
                  />
                </div>
              </div>
            )}
            <Button
              variant="outline"
              size="sm"
              onClick={handleAddServer}
              disabled={
                !newServer.name.trim() ||
                updateMcp.isPending ||
                (newServer.type !== "stdio" && !newServer.url?.trim()) ||
                (newServer.type === "stdio" && !newServer.command?.trim())
              }
            >
              <Plus className="h-3.5 w-3.5 mr-1" />
              Add Server
            </Button>
          </div>
        </div>

        {/* Disabled Default Servers */}
        <div className="space-y-3">
          <Label className="text-base font-semibold">
            Disabled Default Servers
          </Label>
          <div className="text-xs text-muted-foreground">
            Platform default servers (context7, deepwiki, webfetch, etc.) to
            disable for all sessions.
          </div>

          {disabledServers.length > 0 && (
            <div className="flex flex-wrap gap-2">
              {disabledServers.map((name) => (
                <Badge
                  key={name}
                  variant="secondary"
                  className="gap-1 pr-1"
                >
                  {name}
                  <button
                    type="button"
                    onClick={() => handleRemoveDisabled(name)}
                    className="ml-1 rounded-full p-0.5 hover:bg-muted"
                  >
                    <Trash2 className="h-3 w-3" />
                  </button>
                </Badge>
              ))}
            </div>
          )}

          <div className="flex gap-2">
            <Input
              placeholder="e.g. context7"
              value={disabledInput}
              onChange={(e) => setDisabledInput(e.target.value)}
              className="max-w-[200px]"
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  handleAddDisabled();
                }
              }}
            />
            <Button
              variant="outline"
              size="sm"
              onClick={handleAddDisabled}
              disabled={!disabledInput.trim() || updateMcp.isPending}
            >
              <Plus className="h-3.5 w-3.5 mr-1" />
              Disable
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
