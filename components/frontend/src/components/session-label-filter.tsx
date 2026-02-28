"use client";

import { useState } from "react";
import { Filter, ChevronLeft } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";

type SessionLabelFilterProps = {
  labelsData: Record<string, string[]> | undefined;
  activeFilters: Record<string, string>;
  onAddFilter: (key: string, value: string) => void;
};

export function SessionLabelFilter({
  labelsData,
  activeFilters,
  onAddFilter,
}: SessionLabelFilterProps) {
  const [open, setOpen] = useState(false);
  const [selectedKey, setSelectedKey] = useState<string | null>(null);

  const activeCount = Object.keys(activeFilters).length;

  const handleSelectKey = (key: string) => {
    setSelectedKey(key);
  };

  const handleSelectValue = (value: string) => {
    if (selectedKey) {
      onAddFilter(selectedKey, value);
      setSelectedKey(null);
      setOpen(false);
    }
  };

  const handleBack = () => {
    setSelectedKey(null);
  };

  const handleOpenChange = (nextOpen: boolean) => {
    setOpen(nextOpen);
    if (!nextOpen) {
      setSelectedKey(null);
    }
  };

  const keys = labelsData ? Object.keys(labelsData).sort() : [];

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className="h-9 gap-1.5">
          <Filter className="h-3.5 w-3.5" />
          Filter
          {activeCount > 0 && (
            <Badge variant="secondary" className="ml-1 h-5 px-1.5 text-[10px]">
              {activeCount}
            </Badge>
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[220px] p-0" align="start">
        <Command>
          <CommandInput placeholder={selectedKey ? `${selectedKey}: search values...` : "Search labels..."} />
          <CommandList>
            <CommandEmpty>No results found.</CommandEmpty>
            {selectedKey ? (
              <CommandGroup>
                <CommandItem onSelect={handleBack} className="text-muted-foreground">
                  <ChevronLeft className="h-3.5 w-3.5 mr-1" />
                  Back
                </CommandItem>
                {(labelsData?.[selectedKey] ?? []).map((value) => (
                  <CommandItem
                    key={value}
                    onSelect={() => handleSelectValue(value)}
                    disabled={activeFilters[selectedKey] === value}
                  >
                    {value}
                  </CommandItem>
                ))}
              </CommandGroup>
            ) : (
              <CommandGroup heading="Label keys">
                {keys.map((key) => (
                  <CommandItem key={key} onSelect={() => handleSelectKey(key)}>
                    <span className="font-medium">{key}</span>
                    <span className="ml-auto text-xs text-muted-foreground">
                      {labelsData?.[key]?.length ?? 0}
                    </span>
                  </CommandItem>
                ))}
              </CommandGroup>
            )}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
