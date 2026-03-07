"use client";

import { useState } from "react";

import {
  FormControl,
  FormDescription,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Textarea } from "@/components/ui/textarea";

export function OutputFormatField({
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
