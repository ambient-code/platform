"use client";

import { useState, useMemo, useCallback } from "react";

export type AutocompleteAgent = {
  id: string;
  name: string;
  description?: string;
};

export type AutocompleteCommand = {
  id: string;
  name: string;
  slashCommand: string;
  description?: string;
};

export type AutocompleteItem = AutocompleteAgent | AutocompleteCommand;
type AutocompleteType = "agent" | "command";

type UseAutocompleteOptions = {
  agents: AutocompleteAgent[];
  commands: AutocompleteCommand[];
};

export function useAutocomplete({ agents, commands }: UseAutocompleteOptions) {
  const [isOpen, setIsOpen] = useState(false);
  const [type, setType] = useState<AutocompleteType | null>(null);
  const [filter, setFilter] = useState("");
  const [triggerPos, setTriggerPos] = useState(0);
  const [selectedIndex, setSelectedIndex] = useState(0);

  const filteredItems = useMemo<AutocompleteItem[]>(() => {
    if (!type) return [];
    const filterLower = filter.toLowerCase();

    if (type === "agent") {
      return agents.filter(
        (a) =>
          a.name.toLowerCase().includes(filterLower) ||
          a.description?.toLowerCase().includes(filterLower)
      );
    }

    return commands.filter(
      (c) =>
        c.name.toLowerCase().includes(filterLower) ||
        c.slashCommand.toLowerCase().includes(filterLower) ||
        c.description?.toLowerCase().includes(filterLower)
    );
  }, [type, filter, agents, commands]);

  const close = useCallback(() => {
    setIsOpen(false);
    setType(null);
    setFilter("");
    setSelectedIndex(0);
  }, []);

  const open = useCallback((acType: AutocompleteType, position: number) => {
    setTriggerPos(position);
    setType(acType);
    setFilter("");
    setSelectedIndex(0);
    setIsOpen(true);
  }, []);

  /** Insert selected item into the text value. Returns new cursor position. */
  const select = useCallback(
    (
      item: AutocompleteItem,
      currentValue: string,
      cursorPos: number,
      onChange: (value: string) => void
    ): number => {
      const textBefore = currentValue.substring(0, triggerPos);
      const textAfter = currentValue.substring(cursorPos);

      let insertText: string;
      if (type === "agent") {
        const agent = item as AutocompleteAgent;
        insertText = `@${agent.name.split(" - ")[0]} `;
      } else {
        const cmd = item as AutocompleteCommand;
        insertText = `${cmd.slashCommand} `;
      }

      onChange(textBefore + insertText + textAfter);
      const newCursorPos = textBefore.length + insertText.length;
      close();
      return newCursorPos;
    },
    [type, triggerPos, close]
  );

  /** Detect @/slash triggers and update filter on input changes. */
  const handleInputChange = useCallback(
    (newValue: string, cursorPos: number) => {
      if (cursorPos > 0) {
        const charBeforeCursor = newValue[cursorPos - 1];

        // Check for @ or / trigger at start or after whitespace
        if (charBeforeCursor === "@" || charBeforeCursor === "/") {
          if (cursorPos === 1 || /\s/.test(newValue[cursorPos - 2])) {
            open(charBeforeCursor === "@" ? "agent" : "command", cursorPos - 1);
            return;
          }
        }

        // Update filter while autocomplete is open
        if (isOpen) {
          const filterText = newValue.substring(triggerPos + 1, cursorPos);
          if (cursorPos <= triggerPos || /\s/.test(filterText)) {
            close();
          } else {
            setFilter(filterText);
            setSelectedIndex(0);
          }
        }
      } else if (isOpen) {
        close();
      }
    },
    [isOpen, triggerPos, open, close]
  );

  /**
   * Handle keyboard events for autocomplete navigation.
   * Returns true if the event was consumed.
   */
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent): boolean => {
      if (!isOpen || filteredItems.length === 0) return false;

      if (e.key === "ArrowDown") {
        e.preventDefault();
        setSelectedIndex((prev) =>
          prev < filteredItems.length - 1 ? prev + 1 : prev
        );
        return true;
      }
      if (e.key === "ArrowUp") {
        e.preventDefault();
        setSelectedIndex((prev) => (prev > 0 ? prev - 1 : 0));
        return true;
      }
      if (e.key === "Enter" || e.key === "Tab") {
        e.preventDefault();
        // Caller must handle the actual selection since it needs textarea ref
        return true;
      }
      if (e.key === "Escape") {
        e.preventDefault();
        close();
        return true;
      }
      return false;
    },
    [isOpen, filteredItems.length, close]
  );

  return {
    isOpen,
    type,
    filter,
    triggerPos,
    selectedIndex,
    setSelectedIndex,
    filteredItems,
    open,
    close,
    select,
    handleInputChange,
    handleKeyDown,
  };
}
