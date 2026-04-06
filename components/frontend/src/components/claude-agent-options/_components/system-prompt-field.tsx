"use client";

import type { UseFormReturn } from "react-hook-form";

import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

import type { ClaudeAgentOptionsForm } from "../schema";

export function SystemPromptField({ form, disabled }: { form: UseFormReturn<ClaudeAgentOptionsForm>; disabled?: boolean }) {
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
