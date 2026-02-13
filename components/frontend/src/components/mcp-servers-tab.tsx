"use client";

import { useState } from "react";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Plus, MoreHorizontal, Pencil, Trash2, Server } from "lucide-react";
import { successToast, errorToast } from "@/hooks/use-toast";
import { useMcpConfig, useUpdateMcpConfig } from "@/services/queries/use-mcp-config";
import { McpServerDialog } from "@/components/mcp-server-dialog";
import type { McpServerConfig } from "@/services/api/mcp-config";

type McpServersTabProps = {
  projectName: string;
};

export function McpServersTab({ projectName }: McpServersTabProps) {
  const { data: config } = useMcpConfig(projectName);
  const updateMutation = useUpdateMcpConfig();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingServer, setEditingServer] = useState<{ name: string; config: McpServerConfig } | null>(null);

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

  return (
    <>
      <div className="flex justify-end mb-4">
        <Button onClick={handleAdd} size="sm" disabled={updateMutation.isPending}>
          <Plus className="w-4 h-4 mr-2" /> Add Server
        </Button>
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
        initialName={editingServer?.name}
        initialConfig={editingServer?.config}
      />
    </>
  );
}
