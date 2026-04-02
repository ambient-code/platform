"use client";

import { useState } from "react";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ChevronsUpDown, Settings, Code, X } from "lucide-react";
import type { SdkOptions } from "@/types/api/sessions";

const DEFAULT_TOOLS = [
  "Read",
  "Write",
  "Edit",
  "MultiEdit",
  "Bash",
  "Glob",
  "Grep",
  "WebSearch",
  "WebFetch",
  "NotebookEdit",
  "TodoRead",
  "TodoWrite",
  "Agent",
];

type ModelOption = {
  id: string;
  name: string;
};

type AdvancedSdkOptionsProps = {
  value: SdkOptions;
  onChange: (opts: SdkOptions) => void;
  models?: ModelOption[];
};

export function AdvancedSdkOptions({
  value,
  onChange,
  models = [],
}: AdvancedSdkOptionsProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [showJsonPreview, setShowJsonPreview] = useState(false);
  const [betaInput, setBetaInput] = useState("");
  const [customToolInput, setCustomToolInput] = useState("");

  const update = (partial: Partial<SdkOptions>) => {
    onChange({ ...value, ...partial });
  };

  const toggleTool = (tool: string) => {
    const current = value.allowed_tools ?? [];
    const next = current.includes(tool)
      ? current.filter((t) => t !== tool)
      : [...current, tool];
    update({ allowed_tools: next.length > 0 ? next : undefined });
  };

  const addBeta = () => {
    const trimmed = betaInput.trim();
    if (!trimmed) return;
    const current = value.betas ?? [];
    if (!current.includes(trimmed)) {
      update({ betas: [...current, trimmed] });
    }
    setBetaInput("");
  };

  const removeBeta = (beta: string) => {
    const next = (value.betas ?? []).filter((b) => b !== beta);
    update({ betas: next.length > 0 ? next : undefined });
  };

  const addCustomTool = () => {
    const trimmed = customToolInput.trim();
    if (!trimmed) return;
    const current = value.allowed_tools ?? [];
    if (!current.includes(trimmed)) {
      update({ allowed_tools: [...current, trimmed] });
    }
    setCustomToolInput("");
  };

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <CollapsibleTrigger asChild>
        <Button
          variant="ghost"
          size="sm"
          className="gap-1.5 text-muted-foreground hover:text-foreground"
        >
          <Settings className="h-4 w-4" />
          Advanced SDK Options
          <ChevronsUpDown className="h-3.5 w-3.5" />
        </Button>
      </CollapsibleTrigger>
      <CollapsibleContent className="mt-3 space-y-4 border rounded-lg p-4 bg-muted/30">
        {/* Model & Generation */}
        <fieldset className="space-y-3">
          <legend className="text-sm font-medium text-muted-foreground">
            Model &amp; Generation
          </legend>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <Label htmlFor="sdk-temperature" className="text-xs">
                Temperature
              </Label>
              <Input
                id="sdk-temperature"
                type="number"
                step={0.1}
                min={0}
                max={2}
                placeholder="1.0"
                value={value.temperature ?? ""}
                onChange={(e) =>
                  update({
                    temperature:
                      e.target.value === ""
                        ? undefined
                        : parseFloat(e.target.value),
                  })
                }
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="sdk-max-tokens" className="text-xs">
                Max Tokens
              </Label>
              <Input
                id="sdk-max-tokens"
                type="number"
                min={1}
                max={200000}
                placeholder="4096"
                value={value.max_tokens ?? ""}
                onChange={(e) =>
                  update({
                    max_tokens:
                      e.target.value === ""
                        ? undefined
                        : parseInt(e.target.value),
                  })
                }
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="sdk-max-thinking" className="text-xs">
                Max Thinking Tokens
              </Label>
              <Input
                id="sdk-max-thinking"
                type="number"
                min={0}
                max={128000}
                placeholder="(default)"
                value={value.max_thinking_tokens ?? ""}
                onChange={(e) =>
                  update({
                    max_thinking_tokens:
                      e.target.value === ""
                        ? undefined
                        : parseInt(e.target.value),
                  })
                }
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="sdk-max-turns" className="text-xs">
                Max Turns
              </Label>
              <Input
                id="sdk-max-turns"
                type="number"
                min={1}
                max={1000}
                placeholder="(unlimited)"
                value={value.max_turns ?? ""}
                onChange={(e) =>
                  update({
                    max_turns:
                      e.target.value === ""
                        ? undefined
                        : parseInt(e.target.value),
                  })
                }
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="sdk-max-budget" className="text-xs">
                Max Budget (USD)
              </Label>
              <Input
                id="sdk-max-budget"
                type="number"
                min={0}
                step={0.01}
                placeholder="(unlimited)"
                value={value.max_budget_usd ?? ""}
                onChange={(e) =>
                  update({
                    max_budget_usd:
                      e.target.value === ""
                        ? undefined
                        : parseFloat(e.target.value),
                  })
                }
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="sdk-fallback-model" className="text-xs">
                Fallback Model
              </Label>
              <Select
                value={value.fallback_model ?? "__none__"}
                onValueChange={(v) =>
                  update({ fallback_model: v === "__none__" ? undefined : v })
                }
              >
                <SelectTrigger id="sdk-fallback-model">
                  <SelectValue placeholder="None" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__none__">None</SelectItem>
                  {models.map((m) => (
                    <SelectItem key={m.id} value={m.id}>
                      {m.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
        </fieldset>

        {/* Execution & Control */}
        <fieldset className="space-y-3">
          <legend className="text-sm font-medium text-muted-foreground">
            Execution &amp; Control
          </legend>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1 col-span-2">
              <Label htmlFor="sdk-permission-mode" className="text-xs">
                Permission Mode
              </Label>
              <Select
                value={value.permission_mode ?? "__default__"}
                onValueChange={(v) =>
                  update({
                    permission_mode:
                      v === "__default__"
                        ? undefined
                        : (v as SdkOptions["permission_mode"]),
                  })
                }
              >
                <SelectTrigger id="sdk-permission-mode">
                  <SelectValue placeholder="Default (acceptEdits)" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__default__">Default (acceptEdits)</SelectItem>
                  <SelectItem value="default">Prompt</SelectItem>
                  <SelectItem value="acceptEdits">Accept Edits</SelectItem>
                  <SelectItem value="bypassPermissions">
                    Bypass Permissions
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label htmlFor="sdk-partial-messages" className="text-xs">
                Streaming (Partial Messages)
              </Label>
              <Switch
                id="sdk-partial-messages"
                checked={value.include_partial_messages ?? true}
                onCheckedChange={(checked) =>
                  update({ include_partial_messages: checked })
                }
              />
            </div>
            <div className="flex items-center justify-between">
              <Label htmlFor="sdk-file-checkpointing" className="text-xs">
                File Checkpointing
              </Label>
              <Switch
                id="sdk-file-checkpointing"
                checked={value.enable_file_checkpointing ?? false}
                onCheckedChange={(checked) =>
                  update({ enable_file_checkpointing: checked })
                }
              />
            </div>
            <div className="flex items-center justify-between">
              <Label htmlFor="sdk-strict-mcp" className="text-xs">
                Strict MCP Config
              </Label>
              <Switch
                id="sdk-strict-mcp"
                checked={value.strict_mcp_config ?? false}
                onCheckedChange={(checked) =>
                  update({ strict_mcp_config: checked })
                }
              />
            </div>
          </div>
          <div className="space-y-1">
            <Label htmlFor="sdk-output-format" className="text-xs">
              Output Format (JSON Schema)
            </Label>
            <Textarea
              id="sdk-output-format"
              placeholder='{"type": "object", "properties": {...}}'
              className="font-mono text-xs min-h-[60px]"
              value={value.output_format ?? ""}
              onChange={(e) =>
                update({
                  output_format:
                    e.target.value === "" ? undefined : e.target.value,
                })
              }
            />
          </div>
        </fieldset>

        {/* Tools */}
        <fieldset className="space-y-3">
          <legend className="text-sm font-medium text-muted-foreground">
            Allowed Tools
          </legend>
          <div className="grid grid-cols-3 gap-2">
            {DEFAULT_TOOLS.map((tool) => (
              <div
                key={tool}
                className="flex items-center justify-between px-2 py-1.5 border rounded text-xs"
              >
                <span>{tool}</span>
                <Switch
                  checked={(value.allowed_tools ?? []).includes(tool)}
                  onCheckedChange={() => toggleTool(tool)}
                  className="scale-75"
                />
              </div>
            ))}
          </div>
          {/* Custom tools not in default list */}
          {(value.allowed_tools ?? [])
            .filter((t) => !DEFAULT_TOOLS.includes(t))
            .map((tool) => (
              <Badge key={tool} variant="secondary" className="gap-1 mr-1">
                {tool}
                <X
                  className="h-3 w-3 cursor-pointer"
                  onClick={() => toggleTool(tool)}
                />
              </Badge>
            ))}
          <div className="flex gap-2">
            <Input
              placeholder="Add custom tool..."
              className="text-xs h-8"
              value={customToolInput}
              onChange={(e) => setCustomToolInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  addCustomTool();
                }
              }}
            />
            <Button
              variant="outline"
              size="sm"
              className="h-8"
              onClick={addCustomTool}
              disabled={!customToolInput.trim()}
            >
              Add
            </Button>
          </div>
        </fieldset>

        {/* System Prompt */}
        <fieldset className="space-y-2">
          <legend className="text-sm font-medium text-muted-foreground">
            System Prompt
          </legend>
          <Textarea
            placeholder="Custom instructions appended to the platform system prompt..."
            className="min-h-[80px] text-sm"
            maxLength={10000}
            value={value.system_prompt ?? ""}
            onChange={(e) =>
              update({
                system_prompt:
                  e.target.value === "" ? undefined : e.target.value,
              })
            }
          />
          <p className="text-xs text-muted-foreground">
            Merged with workspace context. Your prompt is appended after the
            platform system prompt.
          </p>
        </fieldset>

        {/* Beta Flags */}
        <fieldset className="space-y-2">
          <legend className="text-sm font-medium text-muted-foreground">
            Beta Feature Flags
          </legend>
          <div className="flex flex-wrap gap-1">
            {(value.betas ?? []).map((beta) => (
              <Badge key={beta} variant="secondary" className="gap-1">
                {beta}
                <X
                  className="h-3 w-3 cursor-pointer"
                  onClick={() => removeBeta(beta)}
                />
              </Badge>
            ))}
          </div>
          <div className="flex gap-2">
            <Input
              placeholder="Add beta flag..."
              className="text-xs h-8"
              value={betaInput}
              onChange={(e) => setBetaInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  addBeta();
                }
              }}
            />
            <Button
              variant="outline"
              size="sm"
              className="h-8"
              onClick={addBeta}
              disabled={!betaInput.trim()}
            >
              Add
            </Button>
          </div>
        </fieldset>

        {/* JSON Preview */}
        <div className="space-y-2">
          <Button
            variant="ghost"
            size="sm"
            className="gap-1.5 text-xs text-muted-foreground"
            onClick={() => setShowJsonPreview(!showJsonPreview)}
          >
            <Code className="h-3.5 w-3.5" />
            {showJsonPreview ? "Hide" : "Show"} JSON Preview
          </Button>
          {showJsonPreview && (() => {
            const previewObj = Object.fromEntries(
              Object.entries(value).filter(([, v]) => v !== undefined)
            );
            return (
              <pre className="bg-muted p-3 rounded-md text-xs font-mono overflow-auto max-h-[200px]">
                {Object.keys(previewObj).length > 0
                  ? JSON.stringify(previewObj, null, 2)
                  : "// No options set — using defaults"}
              </pre>
            );
          })()}
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}
