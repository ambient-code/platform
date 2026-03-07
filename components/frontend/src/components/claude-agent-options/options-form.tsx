"use client";

import type { UseFormReturn } from "react-hook-form";
import {
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
  Code2,
} from "lucide-react";

import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";

import type { ClaudeAgentOptionsForm } from "./schema";
import { Section } from "./_components/section";
import { KeyValueEditor } from "./_components/key-value-editor";
import { StringListEditor } from "./_components/string-list-editor";
import { SystemPromptField } from "./_components/system-prompt-field";
import { ThinkingField } from "./_components/thinking-field";
import { SandboxField } from "./_components/sandbox-field";
import { McpServersEditor } from "./_components/mcp-servers-editor";
import { HooksEditor } from "./_components/hooks-editor";
import { AgentsEditor } from "./_components/agents-editor";
import { PluginsEditor } from "./_components/plugins-editor";
import { OutputFormatField } from "./_components/output-format-field";

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
                    <Label key={source} className="flex items-center gap-1.5 text-sm">
                      <Checkbox
                        checked={checked}
                        disabled={disabled}
                        onCheckedChange={(c) => {
                          const current = field.value ?? [];
                          field.onChange(c ? [...current, source] : current.filter((s) => s !== source));
                        }}
                      />
                      {source}
                    </Label>
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
                    <Label key={beta} className="flex items-center gap-1.5 text-sm font-mono">
                      <Checkbox
                        checked={checked}
                        disabled={disabled}
                        onCheckedChange={(c) =>
                          field.onChange(c ? [...(field.value ?? []), beta] : (field.value ?? []).filter((b) => b !== beta))
                        }
                      />
                      {beta}
                    </Label>
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
