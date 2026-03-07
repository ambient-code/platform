"use client";

import { useRef } from "react";
import { Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

export function PluginsEditor({ value, onChange }: { value: { type: "local"; path: string }[]; onChange: (v: { type: "local"; path: string }[]) => void }) {
  const nextId = useRef(0);
  const ids = useRef<number[]>([]);

  while (ids.current.length < value.length) {
    ids.current.push(nextId.current++);
  }
  ids.current.length = value.length;

  return (
    <div className="space-y-2">
      <p className="text-xs text-muted-foreground">Local SDK plugins loaded from filesystem paths.</p>
      {value.map((plugin, i) => (
        <div key={ids.current[i]} className="flex items-center gap-2">
          <Input className="font-mono text-xs" placeholder="/path/to/plugin" value={plugin.path} onChange={(e) => { const next = [...value]; next[i] = { type: "local", path: e.target.value }; onChange(next); }} />
          <Button type="button" variant="ghost" size="icon" className="h-8 w-8 shrink-0" onClick={() => { ids.current.splice(i, 1); onChange(value.filter((_, j) => j !== i)); }}><Trash2 className="h-3 w-3" /></Button>
        </div>
      ))}
      <Button type="button" variant="outline" size="sm" onClick={() => { ids.current.push(nextId.current++); onChange([...value, { type: "local", path: "" }]); }}><Plus className="h-3 w-3 mr-1" /> Add Plugin</Button>
    </div>
  );
}
