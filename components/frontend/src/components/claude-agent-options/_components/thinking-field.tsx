"use client";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

export function ThinkingField({
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
