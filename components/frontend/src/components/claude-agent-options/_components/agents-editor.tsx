"use client";

import { useRef } from "react";
import { Plus, Trash2 } from "lucide-react";
import type { z } from "zod";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

import type { agentDefinitionSchema } from "../schema";
import { StringListEditor } from "./string-list-editor";

type AgentDef = z.infer<typeof agentDefinitionSchema>;

export function AgentsEditor({ value, onChange }: { value: Record<string, AgentDef>; onChange: (v: Record<string, AgentDef>) => void }) {
  const nextId = useRef(0);
  const ids = useRef<number[]>([]);

  const entries = Object.entries(value);

  // Sync IDs with entries length (handles external resets)
  while (ids.current.length < entries.length) {
    ids.current.push(nextId.current++);
  }
  ids.current.length = entries.length;

  const addAgent = () => {
    let i = 1;
    while (`agent-${i}` in value) i++;
    ids.current.push(nextId.current++);
    onChange({ ...value, [`agent-${i}`]: { description: "", prompt: "" } });
  };
  const removeAgent = (index: number) => {
    const name = entries[index][0];
    ids.current.splice(index, 1);
    const next = { ...value };
    delete next[name];
    onChange(next);
  };
  const updateAgentName = (index: number, newName: string) => {
    const oldName = entries[index][0];
    if (newName !== oldName && newName in value) return;
    const next: Record<string, AgentDef> = {};
    for (let i = 0; i < entries.length; i++) {
      next[i === index ? newName : entries[i][0]] = entries[i][1];
    }
    onChange(next);
  };
  const updateAgent = (name: string, agent: AgentDef) => onChange({ ...value, [name]: agent });

  return (
    <div className="space-y-3">
      <p className="text-xs text-muted-foreground">Define custom sub-agents with their own prompt, tools, and model.</p>
      {entries.map(([name, agent], i) => (
        <div key={ids.current[i]} className="border rounded-md p-3 space-y-3">
          <div className="flex items-center gap-2">
            <Input className="font-mono text-xs w-1/3" value={name} placeholder="agent-name" onChange={(e) => updateAgentName(i, e.target.value)} />
            <Select value={agent.model ?? "inherit"} onValueChange={(m) => updateAgent(name, { ...agent, model: m === "inherit" ? null : m as AgentDef["model"] })}>
              <SelectTrigger className="w-32"><SelectValue placeholder="Model" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="inherit">Inherit</SelectItem>
                <SelectItem value="sonnet">Sonnet</SelectItem>
                <SelectItem value="opus">Opus</SelectItem>
                <SelectItem value="haiku">Haiku</SelectItem>
              </SelectContent>
            </Select>
            <Button type="button" variant="ghost" size="icon" className="ml-auto h-8 w-8" aria-label={`Remove ${name}`} onClick={() => removeAgent(i)}><Trash2 className="h-3 w-3" /></Button>
          </div>
          <Input className="text-xs" placeholder="Description" value={agent.description} onChange={(e) => updateAgent(name, { ...agent, description: e.target.value })} />
          <Textarea className="font-mono text-xs" placeholder="Agent prompt..." rows={3} value={agent.prompt} onChange={(e) => updateAgent(name, { ...agent, prompt: e.target.value })} />
          <div>
            <Label className="text-xs text-muted-foreground">Tools</Label>
            <StringListEditor value={agent.tools ?? []} onChange={(t) => updateAgent(name, { ...agent, tools: t })} placeholder="Tool name" />
          </div>
        </div>
      ))}
      <Button type="button" variant="outline" size="sm" onClick={addAgent}><Plus className="h-3 w-3 mr-1" /> Add Agent</Button>
    </div>
  );
}
