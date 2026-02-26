"use client";

import { useState, useCallback, useRef, useImperativeHandle, forwardRef } from "react";
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

export type LabelEditorHandle = {
  /** Auto-add any valid pending input. Returns true if input was empty or added, false if invalid/partial text remains. */
  flush: () => boolean;
  /** Whether the input field has non-empty text. */
  hasPendingInput: () => boolean;
};

const DEFAULT_SUGGESTIONS = ["issue", "research", "team", "type", "other"];

// K8s label segment: 1-63 chars, alphanumeric start/end, dashes/dots/underscores allowed
const K8S_LABEL_REGEX = /^[a-zA-Z0-9]([a-zA-Z0-9._-]{0,61}[a-zA-Z0-9])?$/;

function isValidLabelSegment(s: string): boolean {
  return s.length > 0 && s.length <= 63 && K8S_LABEL_REGEX.test(s);
}

function parseLabel(input: string): { key: string; value: string } | { error: string } | null {
  const trimmed = input.trim();
  if (!trimmed) return null;
  const colonIdx = trimmed.indexOf(":");
  if (colonIdx <= 0 || colonIdx === trimmed.length - 1) return { error: "Use key:value format" };
  const key = trimmed.slice(0, colonIdx).trim();
  const value = trimmed.slice(colonIdx + 1).trim();
  if (!key || !value) return { error: "Use key:value format" };
  if (!isValidLabelSegment(key)) return { error: `Key "${key}" must be 1-63 alphanumeric chars (dashes, dots, underscores allowed)` };
  if (!isValidLabelSegment(value)) return { error: `Value "${value}" must be 1-63 alphanumeric chars (dashes, dots, underscores allowed)` };
  return { key, value };
}

export const LabelEditor = forwardRef<LabelEditorHandle, LabelEditorProps>(
  function LabelEditor({ labels, onChange, disabled = false, suggestions = DEFAULT_SUGGESTIONS }, ref) {
    const [inputValue, setInputValue] = useState("");
    const [suggestionsOpen, setSuggestionsOpen] = useState(false);
    const [validationError, setValidationError] = useState<string | null>(null);
    const inputRef = useRef<HTMLInputElement>(null);

    const tryAdd = useCallback((): boolean => {
      const result = parseLabel(inputValue);
      if (result === null) return true; // empty input, nothing to add
      if ("error" in result) {
        setValidationError(result.error);
        return false;
      }
      setValidationError(null);
      onChange({ ...labels, [result.key]: result.value });
      setInputValue("");
      return true;
    }, [inputValue, labels, onChange]);

    useImperativeHandle(ref, () => ({
      flush: tryAdd,
      hasPendingInput: () => inputValue.trim().length > 0,
    }), [tryAdd, inputValue]);

    const handleRemove = useCallback(
      (key: string) => {
        const next = { ...labels };
        delete next[key];
        onChange(next);
      },
      [labels, onChange]
    );

    const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === "Enter") {
        e.preventDefault();
        tryAdd();
      }
    };

    const handleSuggestionClick = (suggestion: string) => {
      setInputValue(`${suggestion}:`);
      setSuggestionsOpen(false);
      setValidationError(null);
      inputRef.current?.focus();
    };

    const entries = Object.entries(labels);

    return (
      <div className="space-y-2">
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

        {!disabled && (
          <div className="flex gap-2">
            <div className="flex-1">
              <Input
                ref={inputRef}
                value={inputValue}
                onChange={(e) => { setInputValue(e.target.value); setValidationError(null); }}
                onKeyDown={handleKeyDown}
                placeholder="key:value"
                disabled={disabled}
              />
            </div>
            <Popover open={suggestionsOpen} onOpenChange={setSuggestionsOpen}>
              <PopoverTrigger asChild>
                <Button type="button" variant="outline" size="sm" className="h-9 px-2" disabled={disabled}>
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
              onClick={tryAdd}
              disabled={disabled || !inputValue.includes(":")}
            >
              <Plus className="h-4 w-4 mr-1" />
              Add
            </Button>
          </div>
        )}

        {validationError && (
          <p className="text-xs text-destructive">{validationError}</p>
        )}

        {!disabled && !validationError && (
          <p className="text-xs text-muted-foreground">
            Add labels as key:value pairs. Press Enter or click Add.
          </p>
        )}
      </div>
    );
  }
);
