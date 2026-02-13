"use client";

import { useState, useEffect } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Plus, Trash2, Loader2 } from "lucide-react";
import type { HttpToolConfig, HttpMethod } from "@/services/api/http-tools";

type KVEntry = { key: string; value: string };

const HTTP_METHODS: HttpMethod[] = ["GET", "POST", "PUT", "PATCH", "DELETE"];

type HttpToolDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSave: (tool: HttpToolConfig) => void;
  saving: boolean;
  initialTool?: HttpToolConfig;
};

export function HttpToolDialog({ open, onOpenChange, onSave, saving, initialTool }: HttpToolDialogProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [method, setMethod] = useState<HttpMethod>("GET");
  const [endpoint, setEndpoint] = useState("");
  const [headers, setHeaders] = useState<KVEntry[]>([]);
  const [params, setParams] = useState<KVEntry[]>([]);
  const isEditing = !!initialTool;

  useEffect(() => {
    if (open) {
      setName(initialTool?.name ?? "");
      setDescription(initialTool?.description ?? "");
      setMethod(initialTool?.method ?? "GET");
      setEndpoint(initialTool?.endpoint ?? "");
      setHeaders(
        initialTool?.headers
          ? Object.entries(initialTool.headers).map(([key, value]) => ({ key, value }))
          : []
      );
      setParams(
        initialTool?.params
          ? Object.entries(initialTool.params).map(([key, value]) => ({ key, value }))
          : []
      );
    }
  }, [open, initialTool]);

  const handleSubmit = () => {
    if (!name.trim() || !endpoint.trim()) return;
    const toRecord = (entries: KVEntry[]) => {
      const rec: Record<string, string> = {};
      for (const e of entries) {
        if (e.key.trim()) rec[e.key.trim()] = e.value;
      }
      return rec;
    };
    onSave({
      name: name.trim(),
      description: description.trim(),
      method,
      endpoint: endpoint.trim(),
      headers: toRecord(headers),
      params: toRecord(params),
    });
  };

  const addEntry = (setter: typeof setHeaders) => setter((prev) => [...prev, { key: "", value: "" }]);
  const removeEntry = (setter: typeof setHeaders, index: number) => setter((prev) => prev.filter((_, i) => i !== index));
  const updateEntry = (setter: typeof setHeaders, index: number, field: "key" | "value", val: string) =>
    setter((prev) => prev.map((e, i) => (i === index ? { ...e, [field]: val } : e)));

  const renderKVSection = (label: string, entries: KVEntry[], setter: typeof setHeaders) => (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <Label>{label}</Label>
        <Button type="button" variant="ghost" size="sm" onClick={() => addEntry(setter)}>
          <Plus className="w-3 h-3 mr-1" /> Add
        </Button>
      </div>
      {entries.map((entry, i) => (
        <div key={i} className="flex gap-2 items-center">
          <Input value={entry.key} onChange={(e) => updateEntry(setter, i, "key", e.target.value)} placeholder="Key" className="flex-1" />
          <Input value={entry.value} onChange={(e) => updateEntry(setter, i, "value", e.target.value)} placeholder="Value" className="flex-1" />
          <Button type="button" variant="ghost" size="icon" onClick={() => removeEntry(setter, i)}>
            <Trash2 className="w-4 h-4 text-destructive" />
          </Button>
        </div>
      ))}
    </div>
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEditing ? "Edit HTTP Tool" : "Add HTTP Tool"}</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 py-2">
          <div className="space-y-2">
            <Label htmlFor="tool-name">Name</Label>
            <Input id="tool-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. fetch-weather" disabled={isEditing} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="tool-description">Description</Label>
            <Input id="tool-description" value={description} onChange={(e) => setDescription(e.target.value)} placeholder="What does this tool do?" />
          </div>
          <div className="grid grid-cols-3 gap-4">
            <div className="space-y-2">
              <Label>Method</Label>
              <Select value={method} onValueChange={(v) => setMethod(v as HttpMethod)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {HTTP_METHODS.map((m) => (
                    <SelectItem key={m} value={m}>{m}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="col-span-2 space-y-2">
              <Label htmlFor="tool-endpoint">Endpoint URL</Label>
              <Input id="tool-endpoint" value={endpoint} onChange={(e) => setEndpoint(e.target.value)} placeholder="https://api.example.com/data" />
            </div>
          </div>
          {renderKVSection("Headers", headers, setHeaders)}
          {renderKVSection("Parameters", params, setParams)}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={saving}>Cancel</Button>
          <Button onClick={handleSubmit} disabled={saving || !name.trim() || !endpoint.trim()}>
            {saving && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
            {isEditing ? "Update" : "Add"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
