"use client";

import { useState } from "react";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Plus, MoreHorizontal, Pencil, Trash2, Globe } from "lucide-react";
import { successToast, errorToast } from "@/hooks/use-toast";
import { useHttpTools, useUpdateHttpTools } from "@/services/queries/use-http-tools";
import { HttpToolDialog } from "@/components/http-tool-dialog";
import type { HttpToolConfig } from "@/services/api/http-tools";

type HttpToolsTabProps = {
  projectName: string;
};

export function HttpToolsTab({ projectName }: HttpToolsTabProps) {
  const { data } = useHttpTools(projectName);
  const updateMutation = useUpdateHttpTools();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingTool, setEditingTool] = useState<HttpToolConfig | null>(null);

  const tools = data?.tools ?? [];

  const handleAdd = () => {
    setEditingTool(null);
    setDialogOpen(true);
  };

  const handleEdit = (tool: HttpToolConfig) => {
    setEditingTool(tool);
    setDialogOpen(true);
  };

  const handleDelete = (name: string) => {
    const updated = tools.filter((t) => t.name !== name);
    updateMutation.mutate(
      { projectName, data: { tools: updated } },
      {
        onSuccess: () => successToast(`Removed HTTP tool "${name}"`),
        onError: () => errorToast("Failed to remove HTTP tool"),
      }
    );
  };

  const handleSave = (tool: HttpToolConfig) => {
    const updated = editingTool
      ? tools.map((t) => (t.name === editingTool.name ? tool : t))
      : [...tools, tool];
    updateMutation.mutate(
      { projectName, data: { tools: updated } },
      {
        onSuccess: () => {
          successToast(editingTool ? `Updated HTTP tool "${tool.name}"` : `Added HTTP tool "${tool.name}"`);
          setDialogOpen(false);
          setEditingTool(null);
        },
        onError: () => errorToast("Failed to save HTTP tool"),
      }
    );
  };

  return (
    <>
      <div className="flex justify-end mb-4">
        <Button onClick={handleAdd} size="sm" disabled={updateMutation.isPending}>
          <Plus className="w-4 h-4 mr-2" /> Add Tool
        </Button>
      </div>

      {tools.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-12 text-center text-muted-foreground">
          <Globe className="w-10 h-10 mb-3" />
          <p className="text-sm font-medium">No HTTP tools configured</p>
          <p className="text-xs mt-1">Add an HTTP tool to give sessions access to external APIs</p>
        </div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Method</TableHead>
              <TableHead>Endpoint</TableHead>
              <TableHead>Headers</TableHead>
              <TableHead className="w-12" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {tools.map((tool) => (
              <TableRow key={tool.name}>
                <TableCell>
                  <div>
                    <span className="font-medium">{tool.name}</span>
                    {tool.description && (
                      <p className="text-xs text-muted-foreground mt-0.5">{tool.description}</p>
                    )}
                  </div>
                </TableCell>
                <TableCell>
                  <Badge variant="outline">{tool.method}</Badge>
                </TableCell>
                <TableCell className="font-mono text-xs max-w-48 truncate">{tool.endpoint}</TableCell>
                <TableCell>
                  {Object.keys(tool.headers ?? {}).length > 0 ? (
                    <Badge variant="secondary">{Object.keys(tool.headers).length} headers</Badge>
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
                      <DropdownMenuItem onClick={() => handleEdit(tool)}>
                        <Pencil className="w-4 h-4 mr-2" /> Edit
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => handleDelete(tool.name)} className="text-destructive">
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

      <HttpToolDialog
        open={dialogOpen}
        onOpenChange={(open) => {
          setDialogOpen(open);
          if (!open) setEditingTool(null);
        }}
        onSave={handleSave}
        saving={updateMutation.isPending}
        initialTool={editingTool ?? undefined}
      />
    </>
  );
}
