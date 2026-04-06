"use client";

import { useRef } from "react";
import { Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

export function KeyValueEditor({
  value,
  onChange,
  keyPlaceholder = "KEY",
  valuePlaceholder = "value",
}: {
  value: Record<string, string | null>;
  onChange: (v: Record<string, string | null>) => void;
  keyPlaceholder?: string;
  valuePlaceholder?: string;
}) {
  const nextId = useRef(0);
  const ids = useRef<number[]>([]);

  const entries = Object.entries(value);

  // Sync IDs with entries length (handles external resets)
  while (ids.current.length < entries.length) {
    ids.current.push(nextId.current++);
  }
  ids.current.length = entries.length;

  const addEntry = () => {
    // Use a unique placeholder key to avoid collisions
    let key = "";
    let suffix = 0;
    while (key in value) {
      suffix++;
      key = `key_${suffix}`;
    }
    ids.current.push(nextId.current++);
    onChange({ ...value, [key]: "" });
  };
  const removeEntry = (index: number) => {
    const key = entries[index][0];
    ids.current.splice(index, 1);
    const next = { ...value };
    delete next[key];
    onChange(next);
  };
  const updateEntry = (index: number, newKey: string, newVal: string | null) => {
    const oldKey = entries[index][0];
    if (newKey !== oldKey && newKey in value) return;
    const next: Record<string, string | null> = {};
    for (let i = 0; i < entries.length; i++) {
      if (i === index) {
        next[newKey] = newVal;
      } else {
        next[entries[i][0]] = entries[i][1];
      }
    }
    onChange(next);
  };

  return (
    <div className="space-y-2">
      {entries.map(([k, v], i) => (
        <div key={ids.current[i]} className="flex items-center gap-2">
          <Input
            className="font-mono text-xs w-1/3"
            placeholder={keyPlaceholder}
            value={k}
            onChange={(e) => updateEntry(i, e.target.value, v)}
          />
          <Input
            className="font-mono text-xs flex-1"
            placeholder={valuePlaceholder}
            value={v ?? ""}
            onChange={(e) => updateEntry(i, k, e.target.value)}
          />
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-8 w-8 shrink-0"
            aria-label={`Remove ${k || "new"} entry`}
            onClick={() => removeEntry(i)}
          >
            <Trash2 className="h-3 w-3" />
          </Button>
        </div>
      ))}
      <Button type="button" variant="outline" size="sm" onClick={addEntry}>
        <Plus className="h-3 w-3 mr-1" /> Add
      </Button>
    </div>
  );
}
