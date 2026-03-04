"use client";

import React, { useRef, useEffect } from "react";
import { Users, Terminal } from "lucide-react";
import type { AutocompleteItem, AutocompleteAgent, AutocompleteCommand } from "@/hooks/use-autocomplete";

export type { AutocompleteAgent, AutocompleteCommand };

export type AutocompletePopoverProps = {
  open: boolean;
  type: "agent" | "command" | null;
  filter: string;
  selectedIndex: number;
  items: AutocompleteItem[];
  onSelect: (item: AutocompleteItem) => void;
  onSelectedIndexChange: (index: number) => void;
  onClose: () => void;
};

export const AutocompletePopover: React.FC<AutocompletePopoverProps> = ({
  open,
  type,
  filter,
  selectedIndex,
  items,
  onSelect,
  onSelectedIndexChange,
  onClose,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const selectedItemRef = useRef<HTMLDivElement>(null);

  // Scroll selected item into view
  useEffect(() => {
    if (selectedItemRef.current) {
      selectedItemRef.current.scrollIntoView({
        block: "nearest",
        behavior: "smooth",
      });
    }
  }, [selectedIndex]);

  // Click outside handler
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        onClose();
      }
    };

    if (open) {
      document.addEventListener("mousedown", handleClickOutside);
      return () => document.removeEventListener("mousedown", handleClickOutside);
    }
  }, [open, onClose]);

  if (!open || !type) return null;

  const getShortName = (name: string) => name.split(" - ")[0];
  const typeLabel = type === "agent" ? "agents" : "commands";
  const TypeIcon = type === "agent" ? Users : Terminal;

  return (
    <div
      ref={containerRef}
      className="absolute z-[100] bg-card border-2 border-blue-500 rounded-md shadow-lg max-h-60 overflow-y-auto w-80"
      style={{
        bottom: "100%",
        left: "0px",
        marginBottom: "5px",
      }}
    >
      {/* Header */}
      <div className="sticky top-0 bg-card border-b px-3 py-2 flex items-center gap-2">
        <TypeIcon className="h-4 w-4 text-muted-foreground" />
        <span className="text-xs font-medium text-muted-foreground">
          {type === "agent" ? "Mention an agent" : "Run a command"}
        </span>
        {filter && (
          <span className="text-xs text-blue-500 ml-auto">
            &quot;{filter}&quot;
          </span>
        )}
      </div>

      {items.length === 0 ? (
        <div className="px-3 py-4 text-sm text-muted-foreground text-center">
          No {typeLabel} found
          {filter && <span className="block text-xs mt-1">Try a different search</span>}
        </div>
      ) : (
        <div className="py-1">
          {items.map((item, index) => {
            const isAgent = type === "agent";
            const agent = isAgent ? (item as AutocompleteAgent) : null;
            const cmd = !isAgent ? (item as AutocompleteCommand) : null;
            const isSelected = index === selectedIndex;

            return (
              <div
                key={item.id}
                ref={isSelected ? selectedItemRef : null}
                className={`px-3 py-2 cursor-pointer transition-colors ${
                  isSelected ? "bg-accent text-accent-foreground" : "hover:bg-muted/50"
                }`}
                onClick={() => onSelect(item)}
                onMouseEnter={() => onSelectedIndexChange(index)}
              >
                <div className="flex items-center gap-2">
                  {isAgent && (
                    <div className="w-6 h-6 rounded-full bg-blue-100 dark:bg-blue-900 flex items-center justify-center">
                      <span className="text-[10px] font-bold text-blue-600 dark:text-blue-300">
                        {getShortName(agent!.name).charAt(0).toUpperCase()}
                      </span>
                    </div>
                  )}
                  <div className="flex-1 min-w-0">
                    <div className="font-medium text-sm">
                      {isAgent ? `@${getShortName(agent!.name)}` : cmd!.slashCommand}
                    </div>
                    <div className="text-xs text-muted-foreground truncate">
                      {isAgent ? agent!.name : cmd!.name}
                    </div>
                  </div>
                </div>
                {item.description && (
                  <p className="text-xs text-muted-foreground mt-1 line-clamp-2 pl-8">
                    {item.description}
                  </p>
                )}
              </div>
            );
          })}
        </div>
      )}

      {/* Footer hint */}
      <div className="sticky bottom-0 bg-card border-t px-3 py-1.5 text-[10px] text-muted-foreground flex gap-3">
        <span>
          <kbd className="px-1 bg-muted rounded">↑↓</kbd> navigate
        </span>
        <span>
          <kbd className="px-1 bg-muted rounded">Tab</kbd> select
        </span>
        <span>
          <kbd className="px-1 bg-muted rounded">Esc</kbd> close
        </span>
      </div>
    </div>
  );
};

export default AutocompletePopover;
