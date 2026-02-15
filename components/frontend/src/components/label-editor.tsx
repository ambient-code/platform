"use client";

import { useState, useCallback } from "react";
import { X, Plus, ChevronDown } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";

type LabelEditorProps = {
  labels: Record<string, string>;
  onChange: (labels: Record<string, string>) => void;
  disabled?: boolean;
  suggestions?: string[];
};

const DEFAULT_SUGGESTIONS = ["team", "type", "priority", "feature"];

export function LabelEditor({
  labels,
  onChange,
  disabled = false,
  suggestions = DEFAULT_SUGGESTIONS,
}: LabelEditorProps) {
  const [inputValue, setInputValue] = useState("");
  const [suggestionsOpen, setSuggestionsOpen] = useState(false);

  const handleRemove = useCallback(
    (key: string) => {
      const next = { ...labels };
      delete next[key];
      onChange(next);
    },
    [labels, onChange]
  );

  const handleAdd = useCallback(() => {
    const trimmed = inputValue.trim();
    if (!trimmed) return;

    const colonIdx = trimmed.indexOf(":");
    if (colonIdx <= 0 || colonIdx === trimmed.length - 1) return;

    const key = trimmed.slice(0, colonIdx).trim();
    const value = trimmed.slice(colonIdx + 1).trim();
    if (!key || !value) return;

    onChange({ ...labels, [key]: value });
    setInputValue("");
  }, [inputValue, labels, onChange]);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault();
      handleAdd();
    }
  };

  const handleSuggestionClick = (suggestion: string) => {
    setInputValue(`${suggestion}:`);
    setSuggestionsOpen(false);
  };

  const entries = Object.entries(labels);

  return (
    <div className="space-y-2">
      {/* Existing labels */}
      {entries.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {entries.map(([key, value]) => (
            <Badge key={key} variant="secondary" className="gap-1 pr-1">
              <span className="font-semibold">{key}</span>
              <span className="text-muted-foreground">=</span>
              <span>{value}</span>
              {!disabled && (
                <button
                  type="button"
                  onClick={() => handleRemove(key)}
                  className="ml-0.5 rounded-sm hover:bg-muted p-0.5"
                  aria-label={`Remove label ${key}`}
                >
                  <X className="h-3 w-3" />
                </button>
              )}
            </Badge>
          ))}
        </div>
      )}

      {/* Input row */}
      {!disabled && (
        <div className="flex gap-2">
          <div className="flex-1 relative">
            <Input
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="key:value"
              disabled={disabled}
              className="pr-8"
            />
          </div>
          <Popover open={suggestionsOpen} onOpenChange={setSuggestionsOpen}>
            <PopoverTrigger asChild>
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="h-9 px-2"
                disabled={disabled}
              >
                <ChevronDown className="h-4 w-4" />
              </Button>
            </PopoverTrigger>
            <PopoverContent align="end" className="w-40 p-1">
              {suggestions.map((s) => (
                <button
                  key={s}
                  type="button"
                  onClick={() => handleSuggestionClick(s)}
                  className="w-full text-left text-sm px-2 py-1.5 rounded hover:bg-accent"
                >
                  {s}
                </button>
              ))}
            </PopoverContent>
          </Popover>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="h-9"
            onClick={handleAdd}
            disabled={disabled || !inputValue.includes(":")}
          >
            <Plus className="h-4 w-4 mr-1" />
            Add
          </Button>
        </div>
      )}

      {!disabled && (
        <p className="text-xs text-muted-foreground">
          Add labels as key:value pairs. Use the dropdown for common keys.
        </p>
      )}
    </div>
  );
}
