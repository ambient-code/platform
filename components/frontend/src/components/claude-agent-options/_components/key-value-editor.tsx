"use client";

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
  const entries = Object.entries(value);
  const addEntry = () => onChange({ ...value, "": "" });
  const removeEntry = (key: string) => {
    const next = { ...value };
    delete next[key];
    onChange(next);
  };
  const updateEntry = (oldKey: string, newKey: string, newVal: string | null) => {
    const next: Record<string, string | null> = {};
    for (const [k, v] of Object.entries(value)) {
      if (k === oldKey) {
        next[newKey] = newVal;
      } else {
        next[k] = v;
      }
    }
    onChange(next);
  };

  return (
    <div className="space-y-2">
      {entries.map(([k, v]) => (
        <div key={k} className="flex items-center gap-2">
          <Input
            className="font-mono text-xs w-1/3"
            placeholder={keyPlaceholder}
            value={k}
            onChange={(e) => updateEntry(k, e.target.value, v)}
          />
          <Input
            className="font-mono text-xs flex-1"
            placeholder={valuePlaceholder}
            value={v ?? ""}
            onChange={(e) => updateEntry(k, k, e.target.value || null)}
          />
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-8 w-8 shrink-0"
            onClick={() => removeEntry(k)}
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
