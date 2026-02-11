"use client";

import React, { useEffect, useRef } from "react";
import { cn } from "@/lib/utils";

export type AutocompleteItem = {
  id: string;
  label: string;
  detail: string;
  insertText: string;
};

type AutocompletePopupProps = {
  items: AutocompleteItem[];
  selectedIndex: number;
  onSelect: (item: AutocompleteItem) => void;
  onHover: (index: number) => void;
  popupRef: React.RefObject<HTMLDivElement | null>;
  /** The type of autocomplete being shown — used for the empty state message */
  triggerType?: "agent" | "command" | null;
};

export function AutocompletePopup({
  items,
  selectedIndex,
  onSelect,
  onHover,
  popupRef,
  triggerType,
}: AutocompletePopupProps) {
  const selectedRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    selectedRef.current?.scrollIntoView({ block: "nearest" });
  }, [selectedIndex]);

  const emptyLabel =
    triggerType === "command"
      ? "No commands available"
      : triggerType === "agent"
        ? "No agents available"
        : "No suggestions";

  return (
    <div
      ref={popupRef}
      className="absolute z-[100] bg-card border border-border rounded-md shadow-lg max-h-52 overflow-y-auto w-72"
      style={{ bottom: "100%", left: 0, marginBottom: 4 }}
    >
      {items.length === 0 ? (
        <div className="px-3 py-3 text-xs text-muted-foreground text-center">
          {emptyLabel}
        </div>
      ) : (
        items.map((item, idx) => (
          <div
            key={item.id}
            ref={idx === selectedIndex ? selectedRef : undefined}
            className={cn(
              "px-3 py-2 cursor-pointer border-b last:border-b-0 transition-colors",
              idx === selectedIndex
                ? "bg-accent text-accent-foreground"
                : "hover:bg-muted/50",
            )}
            onMouseDown={(e) => {
              e.preventDefault();
              onSelect(item);
            }}
            onMouseEnter={() => onHover(idx)}
          >
            <div className="font-medium text-sm">{item.label}</div>
            {item.detail && (
              <div className="text-xs text-muted-foreground truncate">
                {item.detail}
              </div>
            )}
          </div>
        ))
      )}
    </div>
  );
}
