"use client";

import { useRef } from "react";
import { Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

export function StringListEditor({
  value,
  onChange,
  placeholder = "Enter value",
}: {
  value: string[];
  onChange: (v: string[]) => void;
  placeholder?: string;
}) {
  const nextId = useRef(0);
  const ids = useRef<number[]>([]);

  // Sync IDs with value length (handles external resets)
  while (ids.current.length < value.length) {
    ids.current.push(nextId.current++);
  }
  ids.current.length = value.length;

  const addItem = () => {
    ids.current.push(nextId.current++);
    onChange([...value, ""]);
  };
  const removeItem = (i: number) => {
    ids.current.splice(i, 1);
    onChange(value.filter((_, j) => j !== i));
  };
  const updateItem = (i: number, v: string) =>
    onChange(value.map((old, j) => (j === i ? v : old)));

  return (
    <div className="space-y-2">
      {value.map((item, i) => (
        <div key={ids.current[i]} className="flex items-center gap-2">
          <Input
            className="font-mono text-xs"
            placeholder={placeholder}
            value={item}
            onChange={(e) => updateItem(i, e.target.value)}
          />
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-8 w-8 shrink-0"
            onClick={() => removeItem(i)}
          >
            <Trash2 className="h-3 w-3" />
          </Button>
        </div>
      ))}
      <Button type="button" variant="outline" size="sm" onClick={addItem}>
        <Plus className="h-3 w-3 mr-1" /> Add
      </Button>
    </div>
  );
}
