"use client";

import { useState } from "react";
import type { UseFormReturn } from "react-hook-form";
import {
  Plus,
  Trash2,
  ChevronsUpDown,
  Code2,
  Settings2,
  Terminal,
  Brain,
  Shield,
  Layers,
  Wrench,
  Box,
  Webhook,
  Users,
  Puzzle,
  FileOutput,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Separator } from "@/components/ui/separator";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";

import type { ClaudeAgentOptionsForm } from "./schema";
import type { z } from "zod";
import type {
  agentDefinitionSchema,
  hookMatcherFormSchema,
} from "./schema";

// ---------------------------------------------------------------------------
// Section wrapper — collapsible group with icon
// ---------------------------------------------------------------------------
function Section({
  title,
  icon: Icon,
  defaultOpen = false,
  badge,
  children,
}: {
  title: string;
  icon: React.ElementType;
  defaultOpen?: boolean;
  badge?: string;
  children: React.ReactNode;
}) {
  const [open, setOpen] = useState(defaultOpen);
  return (
    <Collapsible open={open} onOpenChange={setOpen} className="border rounded-lg">
      <CollapsibleTrigger className="flex w-full items-center justify-between p-4 hover:bg-muted/50 transition-colors">
        <div className="flex items-center gap-2">
          <Icon className="h-4 w-4 text-muted-foreground" />
          <span className="font-medium text-sm">{title}</span>
          {badge && (
            <Badge variant="secondary" className="text-xs">
              {badge}
            </Badge>
          )}
        </div>
        <ChevronsUpDown className="h-4 w-4 text-muted-foreground" />
      </CollapsibleTrigger>
      <CollapsibleContent>
        <Separator />
        <div className="p-4 space-y-4">{children}</div>
      </CollapsibleContent>
    </Collapsible>
  );
}

// ---------------------------------------------------------------------------
// Key-value pair editor
// ---------------------------------------------------------------------------
function KeyValueEditor({
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
      {entries.map(([k, v], i) => (
        <div key={i} className="flex items-center gap-2">
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

// ---------------------------------------------------------------------------
// String list editor
// ---------------------------------------------------------------------------
function StringListEditor({
  value,
  onChange,
  placeholder = "Enter value",
}: {
  value: string[];
  onChange: (v: string[]) => void;
  placeholder?: string;
}) {
  const addItem = () => onChange([...value, ""]);
  const removeItem = (i: number) => onChange(value.filter((_, j) => j !== i));
  const updateItem = (i: number, v: string) =>
    onChange(value.map((old, j) => (j === i ? v : old)));

  return (
    <div className="space-y-2">
      {value.map((item, i) => (
        <div key={i} className="flex items-center gap-2">
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

// ---------------------------------------------------------------------------
// Exported form sections — rendered inside the parent <Form> provider
// ---------------------------------------------------------------------------

type AgentOptionsFieldsProps = {
  form: UseFormReturn<ClaudeAgentOptionsForm>;
  disabled?: boolean;
};

export function AgentOptionsFields({ form, disabled }: AgentOptionsFieldsProps) {
  return (
    <div className="space-y-3">
      {/* Core */}
      <Section title="Agent Options" icon={Settings2} defaultOpen>
        <FormField
          control={form.control}
          name="permission_mode"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Permission Mode</FormLabel>
              <Select onValueChange={field.onChange} value={field.value} disabled={disabled}>
                <FormControl>
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="Select permission mode" />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  <SelectItem value="default">Default — prompt before tool calls</SelectItem>
                  <SelectItem value="acceptEdits">Accept Edits — auto-approve file edits</SelectItem>
                  <SelectItem value="plan">Plan — read-only, no writes</SelectItem>
                  <SelectItem value="bypassPermissions">Bypass — auto-approve everything</SelectItem>
                </SelectContent>
              </Select>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name="effort"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Effort</FormLabel>
              <Select onValueChange={field.onChange} value={field.value} disabled={disabled}>
                <FormControl>
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="Select effort level" />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  <SelectItem value="low">Low</SelectItem>
                  <SelectItem value="medium">Medium</SelectItem>
                  <SelectItem value="high">High</SelectItem>
                  <SelectItem value="max">Max</SelectItem>
                </SelectContent>
              </Select>
              <FormDescription>Controls how much effort the agent puts into responses</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />

        <div className="grid grid-cols-2 gap-4">
          <FormField
            control={form.control}
            name="max_turns"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Max Turns</FormLabel>
                <FormControl>
                  <Input
                    type="number"
                    placeholder="Unlimited"
                    disabled={disabled}
                    value={field.value ?? ""}
                    onChange={(e) => field.onChange(e.target.value ? Number(e.target.value) : undefined)}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="max_budget_usd"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Max Budget (USD)</FormLabel>
                <FormControl>
                  <Input
                    type="number"
                    step="0.01"
                    placeholder="No limit"
                    disabled={disabled}
                    value={field.value ?? ""}
                    onChange={(e) => field.onChange(e.target.value ? Number(e.target.value) : undefined)}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>
      </Section>

      {/* System Prompt */}
      <Section title="System Prompt" icon={Brain}>
        <SystemPromptField form={form} disabled={disabled} />
      </Section>

      {/* Tools */}
      <Section title="Tools" icon={Wrench}>
        <FormField
          control={form.control}
          name="allowed_tools"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Allowed Tools</FormLabel>
              <FormControl>
                <StringListEditor
                  value={field.value ?? []}
                  onChange={field.onChange}
                  placeholder="Tool name pattern (e.g. mcp__*, Edit)"
                />
              </FormControl>
              <FormDescription>Explicitly allow these tools (glob patterns supported)</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="disallowed_tools"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Disallowed Tools</FormLabel>
              <FormControl>
                <StringListEditor
                  value={field.value ?? []}
                  onChange={field.onChange}
                  placeholder="Tool name pattern (e.g. Bash)"
                />
              </FormControl>
              <FormDescription>Prevent the agent from using these tools</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
      </Section>

      {/* MCP Servers */}
      <Section title="MCP Servers" icon={Layers}>
        <FormField
          control={form.control}
          name="mcp_servers"
          render={({ field }) => (
            <McpServersEditor value={field.value ?? {}} onChange={field.onChange} />
          )}
        />
      </Section>

      {/* Thinking */}
      <Section title="Thinking" icon={Brain} badge="Extended">
        <FormField
          control={form.control}
          name="thinking"
          render={({ field }) => <ThinkingField value={field.value} onChange={field.onChange} disabled={disabled} />}
        />
      </Section>

      {/* Session */}
      <Section title="Session" icon={Terminal}>
        <FormField
          control={form.control}
          name="cwd"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Working Directory</FormLabel>
              <FormControl>
                <Input {...field} value={field.value ?? ""} placeholder="/path/to/project" className="font-mono text-xs" disabled={disabled} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="add_dirs"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Additional Directories</FormLabel>
              <FormControl>
                <StringListEditor value={field.value ?? []} onChange={field.onChange} placeholder="/path/to/extra/dir" />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <div className="grid grid-cols-3 gap-4">
          <FormField
            control={form.control}
            name="continue_conversation"
            render={({ field }) => (
              <FormItem className="flex items-center gap-2 space-y-0">
                <FormControl>
                  <Switch checked={field.value} onCheckedChange={field.onChange} disabled={disabled} />
                </FormControl>
                <FormLabel>Continue Conversation</FormLabel>
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="fork_session"
            render={({ field }) => (
              <FormItem className="flex items-center gap-2 space-y-0">
                <FormControl>
                  <Switch checked={field.value} onCheckedChange={field.onChange} disabled={disabled} />
                </FormControl>
                <FormLabel>Fork Session</FormLabel>
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="enable_file_checkpointing"
            render={({ field }) => (
              <FormItem className="flex items-center gap-2 space-y-0">
                <FormControl>
                  <Switch checked={field.value} onCheckedChange={field.onChange} disabled={disabled} />
                </FormControl>
                <FormLabel>File Checkpointing</FormLabel>
              </FormItem>
            )}
          />
        </div>
        <FormField
          control={form.control}
          name="setting_sources"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Setting Sources</FormLabel>
              <div className="flex gap-4 pt-1">
                {(["user", "project", "local"] as const).map((source) => {
                  const checked = (field.value ?? []).includes(source);
                  return (
                    <label key={source} className="flex items-center gap-1.5 text-sm">
                      <Checkbox
                        checked={checked}
                        disabled={disabled}
                        onCheckedChange={(c) => {
                          const current = field.value ?? [];
                          field.onChange(c ? [...current, source] : current.filter((s) => s !== source));
                        }}
                      />
                      {source}
                    </label>
                  );
                })}
              </div>
              <FormMessage />
            </FormItem>
          )}
        />
      </Section>

      {/* Environment */}
      <Section title="Environment" icon={Code2}>
        <FormField
          control={form.control}
          name="env"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Environment Variables</FormLabel>
              <FormControl>
                <KeyValueEditor value={field.value ?? {}} onChange={field.onChange} keyPlaceholder="ENV_VAR" valuePlaceholder="value" />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="extra_args"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Extra CLI Arguments</FormLabel>
              <FormControl>
                <KeyValueEditor value={field.value ?? {}} onChange={field.onChange} keyPlaceholder="--flag" valuePlaceholder="value (empty for boolean)" />
              </FormControl>
              <FormDescription>Arbitrary CLI flags passed to the Claude process</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="cli_path"
          render={({ field }) => (
            <FormItem>
              <FormLabel>CLI Path</FormLabel>
              <FormControl>
                <Input {...field} value={field.value ?? ""} placeholder="Auto-detect" className="font-mono text-xs" disabled={disabled} />
              </FormControl>
              <FormDescription>Path to the Claude CLI binary</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="user"
          render={({ field }) => (
            <FormItem>
              <FormLabel>User Identifier</FormLabel>
              <FormControl>
                <Input {...field} value={field.value ?? ""} placeholder="user@example.com" disabled={disabled} />
              </FormControl>
              <FormDescription>User identifier for SDK tracking</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
      </Section>

      {/* Sandbox */}
      <Section title="Sandbox" icon={Box} badge="Bash Isolation">
        <SandboxField form={form} disabled={disabled} />
      </Section>

      {/* Hooks */}
      <Section title="Hooks" icon={Webhook}>
        <FormField
          control={form.control}
          name="hooks"
          render={({ field }) => <HooksEditor value={field.value ?? {}} onChange={field.onChange} />}
        />
      </Section>

      {/* Agents */}
      <Section title="Agents" icon={Users}>
        <FormField
          control={form.control}
          name="agents"
          render={({ field }) => <AgentsEditor value={field.value ?? {}} onChange={field.onChange} />}
        />
      </Section>

      {/* Plugins */}
      <Section title="Plugins" icon={Puzzle}>
        <FormField
          control={form.control}
          name="plugins"
          render={({ field }) => <PluginsEditor value={field.value ?? []} onChange={field.onChange} />}
        />
      </Section>

      {/* Output Format */}
      <Section title="Output Format" icon={FileOutput}>
        <FormField
          control={form.control}
          name="output_format"
          render={({ field }) => <OutputFormatField value={field.value} onChange={field.onChange} disabled={disabled} />}
        />
      </Section>

      {/* Advanced */}
      <Section title="Advanced" icon={Shield}>
        <FormField
          control={form.control}
          name="max_buffer_size"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Max Buffer Size</FormLabel>
              <FormControl>
                <Input
                  type="number"
                  placeholder="Default"
                  disabled={disabled}
                  value={field.value ?? ""}
                  onChange={(e) => field.onChange(e.target.value ? Number(e.target.value) : undefined)}
                />
              </FormControl>
              <FormDescription>Max bytes for CLI stdout buffer (min 1024)</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="include_partial_messages"
          render={({ field }) => (
            <FormItem className="flex items-center gap-2 space-y-0">
              <FormControl>
                <Switch checked={field.value} onCheckedChange={field.onChange} disabled={disabled} />
              </FormControl>
              <div>
                <FormLabel>Include Partial Messages</FormLabel>
                <FormDescription>Stream partial message chunks as they arrive</FormDescription>
              </div>
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="betas"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Beta Features</FormLabel>
              <div className="flex gap-4 pt-1">
                {(["context-1m-2025-08-07"] as const).map((beta) => {
                  const checked = (field.value ?? []).includes(beta);
                  return (
                    <label key={beta} className="flex items-center gap-1.5 text-sm font-mono">
                      <Checkbox
                        checked={checked}
                        disabled={disabled}
                        onCheckedChange={(c) =>
                          field.onChange(c ? [...(field.value ?? []), beta] : (field.value ?? []).filter((b) => b !== beta))
                        }
                      />
                      {beta}
                    </label>
                  );
                })}
              </div>
              <FormMessage />
            </FormItem>
          )}
        />
      </Section>
    </div>
  );
}

// ---------------------------------------------------------------------------
// SystemPromptField
// ---------------------------------------------------------------------------
function SystemPromptField({ form, disabled }: { form: UseFormReturn<ClaudeAgentOptionsForm>; disabled?: boolean }) {
  const value = form.watch("system_prompt");
  const isPreset = typeof value === "object" && value !== null;

  return (
    <FormField
      control={form.control}
      name="system_prompt"
      render={({ field }) => (
        <FormItem className="space-y-4">
          <div className="flex items-center gap-4">
            <FormLabel>Mode</FormLabel>
            <Select
              value={isPreset ? "preset" : "custom"}
              disabled={disabled}
              onValueChange={(v) => {
                if (v === "preset") {
                  field.onChange({ type: "preset" as const, preset: "claude_code" as const });
                } else {
                  field.onChange("");
                }
              }}
            >
              <SelectTrigger className="w-48">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="custom">Custom text</SelectItem>
                <SelectItem value="preset">Preset (claude_code)</SelectItem>
              </SelectContent>
            </Select>
          </div>
          {isPreset ? (
            <FormItem>
              <FormLabel>Append to preset</FormLabel>
              <FormControl>
                <Textarea
                  placeholder="Additional instructions appended after the preset prompt..."
                  rows={4}
                  className="font-mono text-xs"
                  disabled={disabled}
                  value={(value as { append?: string }).append ?? ""}
                  onChange={(e) =>
                    field.onChange({
                      type: "preset" as const,
                      preset: "claude_code" as const,
                      ...(e.target.value ? { append: e.target.value } : {}),
                    })
                  }
                />
              </FormControl>
            </FormItem>
          ) : (
            <FormControl>
              <Textarea
                placeholder="Enter custom system prompt..."
                rows={6}
                className="font-mono text-xs"
                disabled={disabled}
                value={typeof value === "string" ? value : ""}
                onChange={(e) => field.onChange(e.target.value)}
              />
            </FormControl>
          )}
          <FormMessage />
        </FormItem>
      )}
    />
  );
}

// ---------------------------------------------------------------------------
// ThinkingField
// ---------------------------------------------------------------------------
function ThinkingField({
  value,
  onChange,
  disabled,
}: {
  value: { type: "adaptive" } | { type: "enabled"; budget_tokens: number } | { type: "disabled" } | undefined;
  onChange: (v: typeof value) => void;
  disabled?: boolean;
}) {
  const current = value ?? { type: "adaptive" as const };

  return (
    <div className="space-y-4">
      <div>
        <Label>Thinking Mode</Label>
        <Select
          value={current.type}
          disabled={disabled}
          onValueChange={(t) => {
            if (t === "adaptive") onChange({ type: "adaptive" });
            else if (t === "enabled") onChange({ type: "enabled", budget_tokens: 10000 });
            else onChange({ type: "disabled" });
          }}
        >
          <SelectTrigger className="w-full mt-1.5">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="adaptive">Adaptive — model decides when to think</SelectItem>
            <SelectItem value="enabled">Enabled — always think with token budget</SelectItem>
            <SelectItem value="disabled">Disabled — no extended thinking</SelectItem>
          </SelectContent>
        </Select>
      </div>
      {current.type === "enabled" && (
        <div>
          <Label>Budget Tokens</Label>
          <Input
            type="number"
            className="mt-1.5"
            min={1024}
            max={128000}
            disabled={disabled}
            value={"budget_tokens" in current ? current.budget_tokens : 10000}
            onChange={(e) => onChange({ type: "enabled", budget_tokens: Number(e.target.value) || 10000 })}
          />
          <p className="text-xs text-muted-foreground mt-1">Token budget for extended thinking (1,024 — 128,000)</p>
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// SandboxField
// ---------------------------------------------------------------------------
function SandboxField({ form, disabled }: { form: UseFormReturn<ClaudeAgentOptionsForm>; disabled?: boolean }) {
  return (
    <FormField
      control={form.control}
      name="sandbox"
      render={({ field }) => {
        const val = field.value ?? {
          enabled: false,
          autoAllowBashIfSandboxed: true,
          excludedCommands: [],
          allowUnsandboxedCommands: true,
          enableWeakerNestedSandbox: false,
        };
        const update = (patch: Record<string, unknown>) => field.onChange({ ...val, ...patch });

        return (
          <FormItem className="space-y-4">
            <div className="flex items-center gap-2">
              <Switch checked={val.enabled} onCheckedChange={(c) => update({ enabled: c })} disabled={disabled} />
              <FormLabel>Enable Bash Sandboxing</FormLabel>
            </div>
            {val.enabled && (
              <div className="space-y-4 pl-4 border-l-2 border-muted">
                <div className="flex items-center gap-2">
                  <Switch checked={val.autoAllowBashIfSandboxed ?? true} onCheckedChange={(c) => update({ autoAllowBashIfSandboxed: c })} disabled={disabled} />
                  <Label>Auto-approve bash when sandboxed</Label>
                </div>
                <div className="flex items-center gap-2">
                  <Switch checked={val.allowUnsandboxedCommands ?? true} onCheckedChange={(c) => update({ allowUnsandboxedCommands: c })} disabled={disabled} />
                  <Label>Allow unsandboxed commands</Label>
                </div>
                <div className="flex items-center gap-2">
                  <Switch checked={val.enableWeakerNestedSandbox ?? false} onCheckedChange={(c) => update({ enableWeakerNestedSandbox: c })} disabled={disabled} />
                  <Label>Enable weaker nested sandbox (Docker/Linux)</Label>
                </div>
                <div>
                  <Label>Excluded Commands</Label>
                  <p className="text-xs text-muted-foreground mb-2">Commands that run outside the sandbox</p>
                  <StringListEditor value={val.excludedCommands ?? []} onChange={(v) => update({ excludedCommands: v })} placeholder="command name" />
                </div>
                <div className="space-y-3">
                  <Label>Network</Label>
                  <div className="space-y-2 pl-2">
                    <div className="flex items-center gap-2">
                      <Switch checked={val.network?.allowAllUnixSockets ?? false} disabled={disabled} onCheckedChange={(c) => update({ network: { ...(val.network ?? {}), allowAllUnixSockets: c } })} />
                      <Label>Allow all Unix sockets</Label>
                    </div>
                    <div className="flex items-center gap-2">
                      <Switch checked={val.network?.allowLocalBinding ?? false} disabled={disabled} onCheckedChange={(c) => update({ network: { ...(val.network ?? {}), allowLocalBinding: c } })} />
                      <Label>Allow local port binding (macOS)</Label>
                    </div>
                    <div>
                      <label className="text-xs text-muted-foreground">Allowed Unix Sockets</label>
                      <StringListEditor value={val.network?.allowUnixSockets ?? []} onChange={(v) => update({ network: { ...(val.network ?? {}), allowUnixSockets: v } })} placeholder="/var/run/docker.sock" />
                    </div>
                  </div>
                </div>
              </div>
            )}
            <FormMessage />
          </FormItem>
        );
      }}
    />
  );
}

// ---------------------------------------------------------------------------
// MCP Servers
// ---------------------------------------------------------------------------
// Wider than the discriminated union schema — the editor needs to access all
// fields during editing before the type discriminant narrows them.
type McpFormServer = {
  type: "stdio" | "sse" | "http";
  command?: string;
  args?: string[];
  env?: Record<string, string>;
  url?: string;
  headers?: Record<string, string>;
};

function McpServersEditor({ value, onChange }: { value: Record<string, McpFormServer>; onChange: (v: Record<string, McpFormServer>) => void }) {
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
                <label className="text-xs text-muted-foreground">Args</label>
                <StringListEditor value={server.args ?? []} onChange={(a) => updateServer(name, { ...server, args: a })} placeholder="--arg" />
              </div>
              <div>
                <label className="text-xs text-muted-foreground">Environment</label>
                <KeyValueEditor value={server.env ?? {}} onChange={(e) => updateServer(name, { ...server, env: e as Record<string, string> })} />
              </div>
            </>
          ) : (
            <>
              <Input className="font-mono text-xs" placeholder={server.type === "sse" ? "https://server.example.com/sse" : "https://server.example.com/mcp"} value={server.url ?? ""} onChange={(e) => updateServer(name, { ...server, url: e.target.value })} />
              <div>
                <label className="text-xs text-muted-foreground">Headers</label>
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

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------
const HOOK_EVENTS = ["PreToolUse", "PostToolUse", "PostToolUseFailure", "UserPromptSubmit", "Stop", "SubagentStop", "PreCompact", "Notification", "SubagentStart", "PermissionRequest"] as const;
type HookMatcherFormValue = z.infer<typeof hookMatcherFormSchema>;

function HooksEditor({ value, onChange }: { value: Record<string, HookMatcherFormValue[]>; onChange: (v: Record<string, HookMatcherFormValue[]>) => void }) {
  const addHook = (event: string) => onChange({ ...value, [event]: [...(value[event] ?? []), {}] });
  const removeHook = (event: string, index: number) => {
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
        return (
          <div key={event} className="space-y-2">
            <div className="flex items-center justify-between">
              <Label className="font-mono">{event}</Label>
              <Button type="button" variant="outline" size="sm" onClick={() => addHook(event)}><Plus className="h-3 w-3 mr-1" /> Add</Button>
            </div>
            {hooks.map((hook, i) => (
              <div key={i} className="flex items-center gap-2">
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

// ---------------------------------------------------------------------------
// Agents
// ---------------------------------------------------------------------------
type AgentDef = z.infer<typeof agentDefinitionSchema>;

function AgentsEditor({ value, onChange }: { value: Record<string, AgentDef>; onChange: (v: Record<string, AgentDef>) => void }) {
  const entries = Object.entries(value);
  const addAgent = () => onChange({ ...value, [`agent-${entries.length + 1}`]: { description: "", prompt: "" } });
  const removeAgent = (name: string) => { const next = { ...value }; delete next[name]; onChange(next); };
  const updateAgentName = (oldName: string, newName: string) => {
    const next: Record<string, AgentDef> = {};
    for (const [k, v] of Object.entries(value)) next[k === oldName ? newName : k] = v;
    onChange(next);
  };
  const updateAgent = (name: string, agent: AgentDef) => onChange({ ...value, [name]: agent });

  return (
    <div className="space-y-3">
      <p className="text-xs text-muted-foreground">Define custom sub-agents with their own prompt, tools, and model.</p>
      {entries.map(([name, agent]) => (
        <div key={name} className="border rounded-md p-3 space-y-3">
          <div className="flex items-center gap-2">
            <Input className="font-mono text-xs w-1/3" value={name} placeholder="agent-name" onChange={(e) => updateAgentName(name, e.target.value)} />
            <Select value={agent.model ?? "inherit"} onValueChange={(m) => updateAgent(name, { ...agent, model: m === "inherit" ? null : m as AgentDef["model"] })}>
              <SelectTrigger className="w-32"><SelectValue placeholder="Model" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="inherit">Inherit</SelectItem>
                <SelectItem value="sonnet">Sonnet</SelectItem>
                <SelectItem value="opus">Opus</SelectItem>
                <SelectItem value="haiku">Haiku</SelectItem>
              </SelectContent>
            </Select>
            <Button type="button" variant="ghost" size="icon" className="ml-auto h-8 w-8" onClick={() => removeAgent(name)}><Trash2 className="h-3 w-3" /></Button>
          </div>
          <Input className="text-xs" placeholder="Description" value={agent.description} onChange={(e) => updateAgent(name, { ...agent, description: e.target.value })} />
          <Textarea className="font-mono text-xs" placeholder="Agent prompt..." rows={3} value={agent.prompt} onChange={(e) => updateAgent(name, { ...agent, prompt: e.target.value })} />
          <div>
            <label className="text-xs text-muted-foreground">Tools</label>
            <StringListEditor value={agent.tools ?? []} onChange={(t) => updateAgent(name, { ...agent, tools: t })} placeholder="Tool name" />
          </div>
        </div>
      ))}
      <Button type="button" variant="outline" size="sm" onClick={addAgent}><Plus className="h-3 w-3 mr-1" /> Add Agent</Button>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Plugins
// ---------------------------------------------------------------------------
function PluginsEditor({ value, onChange }: { value: { type: "local"; path: string }[]; onChange: (v: { type: "local"; path: string }[]) => void }) {
  return (
    <div className="space-y-2">
      <p className="text-xs text-muted-foreground">Local SDK plugins loaded from filesystem paths.</p>
      {value.map((plugin, i) => (
        <div key={i} className="flex items-center gap-2">
          <Input className="font-mono text-xs" placeholder="/path/to/plugin" value={plugin.path} onChange={(e) => { const next = [...value]; next[i] = { type: "local", path: e.target.value }; onChange(next); }} />
          <Button type="button" variant="ghost" size="icon" className="h-8 w-8 shrink-0" onClick={() => onChange(value.filter((_, j) => j !== i))}><Trash2 className="h-3 w-3" /></Button>
        </div>
      ))}
      <Button type="button" variant="outline" size="sm" onClick={() => onChange([...value, { type: "local", path: "" }])}><Plus className="h-3 w-3 mr-1" /> Add Plugin</Button>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Output Format (JSON editor with parse error feedback)
// ---------------------------------------------------------------------------
function OutputFormatField({
  value,
  onChange,
  disabled,
}: {
  value: { type?: string; schema?: Record<string, unknown> } | undefined;
  onChange: (v: typeof value) => void;
  disabled?: boolean;
}) {
  const [rawJson, setRawJson] = useState(value ? JSON.stringify(value, null, 2) : "");
  const [jsonError, setJsonError] = useState<string | null>(null);

  const handleChange = (text: string) => {
    setRawJson(text);
    if (!text.trim()) {
      setJsonError(null);
      onChange(undefined);
      return;
    }
    try {
      onChange(JSON.parse(text));
      setJsonError(null);
    } catch (e) {
      setJsonError(e instanceof Error ? e.message : "Invalid JSON");
    }
  };

  return (
    <FormItem>
      <FormLabel>JSON Schema</FormLabel>
      <FormControl>
        <Textarea
          placeholder='{"type": "json_schema", "schema": {"type": "object", ...}}'
          className={`font-mono text-xs ${jsonError ? "border-destructive" : ""}`}
          rows={6}
          disabled={disabled}
          value={rawJson}
          onChange={(e) => handleChange(e.target.value)}
        />
      </FormControl>
      {jsonError && <p className="text-xs text-destructive">{jsonError}</p>}
      <FormDescription>Structured output format (Messages API schema)</FormDescription>
      <FormMessage />
    </FormItem>
  );
}
