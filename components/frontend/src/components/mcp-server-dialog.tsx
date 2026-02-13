"use client";

import { useState, useEffect } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Plus, Trash2, Loader2 } from "lucide-react";
import type { McpServerConfig } from "@/services/api/mcp-config";

type EnvEntry = { key: string; value: string };

type McpServerDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSave: (name: string, config: McpServerConfig) => void;
  saving: boolean;
  initialName?: string;
  initialConfig?: McpServerConfig;
};

export function McpServerDialog({ open, onOpenChange, onSave, saving, initialName, initialConfig }: McpServerDialogProps) {
  const [name, setName] = useState("");
  const [command, setCommand] = useState("");
  const [args, setArgs] = useState("");
  const [envEntries, setEnvEntries] = useState<EnvEntry[]>([]);
  const isEditing = !!initialName;

  useEffect(() => {
    if (open) {
      setName(initialName ?? "");
      setCommand(initialConfig?.command ?? "");
      setArgs(initialConfig?.args?.join(", ") ?? "");
      const entries = initialConfig?.env
        ? Object.entries(initialConfig.env).map(([key, value]) => ({ key, value }))
        : [];
      setEnvEntries(entries);
    }
  }, [open, initialName, initialConfig]);

  const handleSubmit = () => {
    if (!name.trim() || !command.trim()) return;
    const parsedArgs = args
      .split(",")
      .map((a) => a.trim())
      .filter(Boolean);
    const env: Record<string, string> = {};
    for (const entry of envEntries) {
      if (entry.key.trim()) {
        env[entry.key.trim()] = entry.value;
      }
    }
    onSave(name.trim(), { command: command.trim(), args: parsedArgs, env });
  };

  const addEnvEntry = () => setEnvEntries([...envEntries, { key: "", value: "" }]);

  const removeEnvEntry = (index: number) => {
    setEnvEntries(envEntries.filter((_, i) => i !== index));
  };

  const updateEnvEntry = (index: number, field: "key" | "value", val: string) => {
    setEnvEntries(envEntries.map((e, i) => (i === index ? { ...e, [field]: val } : e)));
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{isEditing ? "Edit MCP Server" : "Add MCP Server"}</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 py-2">
          <div className="space-y-2">
            <Label htmlFor="server-name">Server Name</Label>
            <Input id="server-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. my-mcp-server" disabled={isEditing} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="server-command">Command</Label>
            <Input id="server-command" value={command} onChange={(e) => setCommand(e.target.value)} placeholder="e.g. npx" />
          </div>
          <div className="space-y-2">
            <Label htmlFor="server-args">Arguments (comma-separated)</Label>
            <Input id="server-args" value={args} onChange={(e) => setArgs(e.target.value)} placeholder="e.g. -y, @modelcontextprotocol/server-filesystem, /path" />
          </div>
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label>Environment Variables</Label>
              <Button type="button" variant="ghost" size="sm" onClick={addEnvEntry}>
                <Plus className="w-3 h-3 mr-1" /> Add
              </Button>
            </div>
            {envEntries.map((entry, i) => (
              <div key={i} className="flex gap-2 items-center">
                <Input value={entry.key} onChange={(e) => updateEnvEntry(i, "key", e.target.value)} placeholder="KEY" className="flex-1" />
                <Input value={entry.value} onChange={(e) => updateEnvEntry(i, "value", e.target.value)} placeholder="value" className="flex-1" />
                <Button type="button" variant="ghost" size="icon" onClick={() => removeEnvEntry(i)}>
                  <Trash2 className="w-4 h-4 text-destructive" />
                </Button>
              </div>
            ))}
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={saving}>Cancel</Button>
          <Button onClick={handleSubmit} disabled={saving || !name.trim() || !command.trim()}>
            {saving && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
            {isEditing ? "Update" : "Add"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
