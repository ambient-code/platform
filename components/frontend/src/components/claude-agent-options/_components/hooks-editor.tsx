"use client";

import { useRef } from "react";
import { Plus, Trash2 } from "lucide-react";
import type { z } from "zod";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

import type { hookMatcherFormSchema } from "../schema";

const HOOK_EVENTS = ["PreToolUse", "PostToolUse", "PostToolUseFailure", "UserPromptSubmit", "Stop", "SubagentStop", "PreCompact", "Notification", "SubagentStart", "PermissionRequest"] as const;
type HookMatcherFormValue = z.infer<typeof hookMatcherFormSchema>;

export function HooksEditor({ value, onChange }: { value: Record<string, HookMatcherFormValue[]>; onChange: (v: Record<string, HookMatcherFormValue[]>) => void }) {
  const nextId = useRef(0);
  const idsMap = useRef<Record<string, number[]>>({});

  const getIds = (event: string, length: number) => {
    if (!idsMap.current[event]) idsMap.current[event] = [];
    const ids = idsMap.current[event];
    while (ids.length < length) ids.push(nextId.current++);
    ids.length = length;
    return ids;
  };

  const addHook = (event: string) => {
    getIds(event, (value[event] ?? []).length).push(nextId.current++);
    onChange({ ...value, [event]: [...(value[event] ?? []), {}] });
  };
  const removeHook = (event: string, index: number) => {
    getIds(event, (value[event] ?? []).length).splice(index, 1);
    const existing = [...(value[event] ?? [])];
    existing.splice(index, 1);
    if (existing.length === 0) { const next = { ...value }; delete next[event]; onChange(next); }
    else onChange({ ...value, [event]: existing });
  };
  const updateHook = (event: string, index: number, hook: HookMatcherFormValue) => {
    const existing = [...(value[event] ?? [])];
    existing[index] = hook;
    onChange({ ...value, [event]: existing });
  };

  return (
    <div className="space-y-4">
      <p className="text-xs text-muted-foreground">Hooks fire Python callbacks at lifecycle events. Matcher patterns filter tool names (e.g. &quot;Bash&quot;, &quot;Write|Edit&quot;).</p>
      {HOOK_EVENTS.map((event) => {
        const hooks = value[event] ?? [];
        const ids = getIds(event, hooks.length);
        return (
          <div key={event} className="space-y-2">
            <div className="flex items-center justify-between">
              <Label className="font-mono">{event}</Label>
              <Button type="button" variant="outline" size="sm" onClick={() => addHook(event)}><Plus className="h-3 w-3 mr-1" /> Add</Button>
            </div>
            {hooks.map((hook, i) => (
              <div key={ids[i]} className="flex items-center gap-2">
                <Input className="font-mono text-xs flex-1" placeholder="matcher (e.g. Bash)" value={hook.matcher ?? ""} onChange={(e) => updateHook(event, i, { ...hook, matcher: e.target.value || null })} />
                <Input className="font-mono text-xs w-24" type="number" placeholder="timeout" value={hook.timeout ?? ""} onChange={(e) => updateHook(event, i, { ...hook, timeout: e.target.value ? Number(e.target.value) : undefined })} />
                <Button type="button" variant="ghost" size="icon" className="h-8 w-8 shrink-0" onClick={() => removeHook(event, i)}><Trash2 className="h-3 w-3" /></Button>
              </div>
            ))}
          </div>
        );
      })}
    </div>
  );
}
