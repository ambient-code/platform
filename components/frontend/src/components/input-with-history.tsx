"use client";

import * as React from "react";
import { cn } from "@/lib/utils";
import { Input } from "@/components/ui/input";
import { Popover, PopoverAnchor, PopoverContent } from "@/components/ui/popover";
import { useInputHistory } from "@/hooks/use-input-history";

interface InputWithHistoryProps extends React.ComponentProps<"input"> {
  historyKey: string;
  maxHistoryItems?: number;
}

export function InputWithHistory({
  historyKey,
  maxHistoryItems = 10,
  value,
  onChange,
  onKeyDown,
  onFocus,
  onBlur,
  className,
  autoComplete,
  ...props
}: InputWithHistoryProps) {
  const { history } = useInputHistory(historyKey, maxHistoryItems);
  const [showDropdown, setShowDropdown] = React.useState(false);
  const [selectedIndex, setSelectedIndex] = React.useState(-1);

  const currentValue = typeof value === "string" ? value : "";

  const filteredHistory = React.useMemo(
    () =>
      history.filter(
        (item) =>
          item.toLowerCase().includes(currentValue.toLowerCase()) &&
          item !== currentValue
      ),
    [history, currentValue]
  );
  const isDropdownOpen = showDropdown && filteredHistory.length > 0;

  const handleSelect = (item: string) => {
    const syntheticEvent = {
      target: { value: item },
    } as React.ChangeEvent<HTMLInputElement>;
    onChange?.(syntheticEvent);
    setShowDropdown(false);
    setSelectedIndex(-1);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (filteredHistory.length > 0) {
      if (!isDropdownOpen && e.key === "ArrowDown") {
        e.preventDefault();
        setShowDropdown(true);
        setSelectedIndex(0);
        return;
      }

      if (!isDropdownOpen && e.key === "ArrowUp") {
        e.preventDefault();
        setShowDropdown(true);
        setSelectedIndex(filteredHistory.length - 1);
        return;
      }
    }

    if (isDropdownOpen && filteredHistory.length > 0) {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setSelectedIndex((prev) =>
          prev < 0 ? 0 : Math.min(prev + 1, filteredHistory.length - 1)
        );
        return;
      }
      if (e.key === "ArrowUp") {
        e.preventDefault();
        setSelectedIndex((prev) =>
          prev < 0 ? filteredHistory.length - 1 : Math.max(prev - 1, -1)
        );
        return;
      }
      if (e.key === "Enter" && selectedIndex >= 0) {
        e.preventDefault();
        handleSelect(filteredHistory[selectedIndex]);
        return;
      }
      if (e.key === "Escape") {
        setShowDropdown(false);
        setSelectedIndex(-1);
        return;
      }
    }
    onKeyDown?.(e);
  };

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setSelectedIndex(-1);
    onChange?.(e);
  };

  const handleFocus = (e: React.FocusEvent<HTMLInputElement>) => {
    setShowDropdown(true);
    onFocus?.(e);
  };

  const handleBlur = (e: React.FocusEvent<HTMLInputElement>) => {
    setTimeout(() => {
      setShowDropdown(false);
      setSelectedIndex(-1);
    }, 150);
    onBlur?.(e);
  };

  return (
    <Popover open={isDropdownOpen}>
      <PopoverAnchor asChild>
        <div className="relative w-full">
          <Input
            value={value}
            onChange={handleChange}
            onFocus={handleFocus}
            onBlur={handleBlur}
            onKeyDown={handleKeyDown}
            className={className}
            autoComplete={autoComplete ?? "off"}
            aria-autocomplete="list"
            aria-expanded={isDropdownOpen}
            aria-controls={`${historyKey}-history-list`}
            {...props}
          />
        </div>
      </PopoverAnchor>
      <PopoverContent
        id={`${historyKey}-history-list`}
        align="start"
        side="bottom"
        sideOffset={4}
        onOpenAutoFocus={(e) => e.preventDefault()}
        className="z-[100] w-[var(--radix-popover-trigger-width)] overflow-hidden p-0"
      >
        <div>
          {filteredHistory.map((item, index) => (
            <button
              key={item}
              type="button"
              onMouseDown={(e) => {
                e.preventDefault();
                handleSelect(item);
              }}
              className={cn(
                "w-full px-3 py-2 text-sm text-left truncate hover:bg-accent hover:text-accent-foreground",
                index === selectedIndex && "bg-accent text-accent-foreground"
              )}
            >
              {item}
            </button>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
}
