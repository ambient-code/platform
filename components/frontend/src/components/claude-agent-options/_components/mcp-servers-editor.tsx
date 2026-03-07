"use client";

import { Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

import { StringListEditor } from "./string-list-editor";
import { KeyValueEditor } from "./key-value-editor";

// Wider than the discriminated union schema — the editor needs to access all
// fields during editing before the type discriminant narrows them.
export type McpFormServer = {
  type: "stdio" | "sse" | "http";
  command?: string;
  args?: string[];
  env?: Record<string, string>;
  url?: string;
  headers?: Record<string, string>;
};

export function McpServersEditor({ value, onChange }: { value: Record<string, McpFormServer>; onChange: (v: Record<string, McpFormServer>) => void }) {
  const entries = Object.entries(value);

  const addServer = () => {
    onChange({ ...value, [`server-${entries.length + 1}`]: { type: "stdio", command: "", args: [], env: {} } });
  };
  const removeServer = (name: string) => {
    const next = { ...value };
    delete next[name];
    onChange(next);
  };
  const updateServerName = (oldName: string, newName: string) => {
    const next: Record<string, McpFormServer> = {};
    for (const [k, v] of Object.entries(value)) next[k === oldName ? newName : k] = v;
    onChange(next);
  };
  const updateServer = (name: string, server: McpFormServer) => onChange({ ...value, [name]: server });

  return (
    <div className="space-y-3">
      {entries.map(([name, server]) => (
        <div key={name} className="border rounded-md p-3 space-y-3">
          <div className="flex items-center gap-2">
            <Input className="font-mono text-xs w-1/3" value={name} placeholder="server-name" onChange={(e) => updateServerName(name, e.target.value)} />
            <Select value={server.type ?? "stdio"} onValueChange={(t) => {
              if (t === "stdio") updateServer(name, { type: "stdio", command: "", args: [], env: {} });
              else updateServer(name, { type: t as "sse" | "http", url: "", headers: {} });
            }}>
              <SelectTrigger className="w-32"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="stdio">stdio</SelectItem>
                <SelectItem value="sse">SSE</SelectItem>
                <SelectItem value="http">HTTP</SelectItem>
              </SelectContent>
            </Select>
            <Button type="button" variant="ghost" size="icon" className="ml-auto h-8 w-8" onClick={() => removeServer(name)}>
              <Trash2 className="h-3 w-3" />
            </Button>
          </div>
          {(server.type ?? "stdio") === "stdio" ? (
            <>
              <Input className="font-mono text-xs" placeholder="command (e.g. uvx mcp-server-fetch)" value={server.command ?? ""} onChange={(e) => updateServer(name, { ...server, command: e.target.value })} />
              <div>
                <Label className="text-xs text-muted-foreground">Args</Label>
                <StringListEditor value={server.args ?? []} onChange={(a) => updateServer(name, { ...server, args: a })} placeholder="--arg" />
              </div>
              <div>
                <Label className="text-xs text-muted-foreground">Environment</Label>
                <KeyValueEditor value={server.env ?? {}} onChange={(e) => updateServer(name, { ...server, env: e as Record<string, string> })} />
              </div>
            </>
          ) : (
            <>
              <Input className="font-mono text-xs" placeholder={server.type === "sse" ? "https://server.example.com/sse" : "https://server.example.com/mcp"} value={server.url ?? ""} onChange={(e) => updateServer(name, { ...server, url: e.target.value })} />
              <div>
                <Label className="text-xs text-muted-foreground">Headers</Label>
                <KeyValueEditor value={server.headers ?? {}} onChange={(h) => updateServer(name, { ...server, headers: h as Record<string, string> })} keyPlaceholder="Header-Name" valuePlaceholder="Header value" />
              </div>
            </>
          )}
        </div>
      ))}
      <Button type="button" variant="outline" size="sm" onClick={addServer}>
        <Plus className="h-3 w-3 mr-1" /> Add MCP Server
      </Button>
    </div>
  );
}
