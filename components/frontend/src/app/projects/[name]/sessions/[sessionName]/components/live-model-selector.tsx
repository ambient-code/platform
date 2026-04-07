"use client";

import { useMemo } from "react";
import { ChevronDown, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useModels } from "@/services/queries/use-models";

type LiveModelSelectorProps = {
  projectName: string;
  currentModel: string;
  provider?: string;
  disabled?: boolean;
  switching?: boolean;
  onSelect: (model: string) => void;
};

export function LiveModelSelector({
  projectName,
  currentModel,
  provider,
  disabled,
  switching,
  onSelect,
}: LiveModelSelectorProps) {
  const { data: modelsData, isLoading, isError } = useModels(projectName, true, provider);

  const models = useMemo(() => {
    return modelsData?.models.map((m) => ({ id: m.id, name: m.label })) ?? [];
  }, [modelsData]);

  const currentModelName =
    models.find((m) => m.id === currentModel)?.name ?? currentModel;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="sm"
          className="gap-1 text-xs text-muted-foreground hover:text-foreground h-7 px-2"
          disabled={disabled || switching}
        >
          {switching ? (
            <Loader2 className="h-3 w-3 animate-spin" />
          ) : null}
          <span className="truncate max-w-[160px]">
            {currentModelName}
          </span>
          <ChevronDown className="h-3 w-3 opacity-50 flex-shrink-0" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" side="top" sideOffset={4}>
        {isLoading ? (
          <div className="flex items-center justify-center px-2 py-4">
            <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
          </div>
        ) : isError ? (
          <div className="px-2 py-4 text-center text-sm text-destructive">
            Failed to load models
          </div>
        ) : models.length > 0 ? (
          <DropdownMenuRadioGroup
            value={currentModel}
            onValueChange={(modelId) => {
              if (modelId !== currentModel) {
                onSelect(modelId);
              }
            }}
          >
            {models.map((model) => (
              <DropdownMenuRadioItem key={model.id} value={model.id}>
                {model.name}
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        ) : (
          <div className="px-2 py-4 text-center text-sm text-muted-foreground">
            No models available
          </div>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
