"use client";

import { useMemo } from "react";
import { ChevronDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useRunnerTypes } from "@/services/queries/use-runner-types";
import { useModels } from "@/services/queries/use-models";

type ModelOption = {
  id: string;
  name: string;
};

function getDefaultModel(models: ModelOption[], defaultModelId?: string): string {
  if (defaultModelId && models.some((m) => m.id === defaultModelId)) {
    return defaultModelId;
  }
  return models[1]?.id ?? models[0]?.id ?? "default";
}

type RunnerModelSelectorProps = {
  projectName: string;
  selectedRunner: string;
  selectedModel: string;
  onSelect: (runner: string, model: string) => void;
};

type RunnerModelsRadioGroupProps = {
  projectName: string;
  runner: { id: string; provider: string };
  selectedRunner: string;
  selectedModel: string;
  onSelect: (runner: string, model: string) => void;
};

function RunnerModelsRadioGroup({
  projectName,
  runner,
  selectedRunner,
  selectedModel,
  onSelect,
}: RunnerModelsRadioGroupProps) {
  // Fetch models for this specific runner's provider
  const { data: modelsData } = useModels(projectName, true, runner.provider);

  const models = modelsData?.models.map((m) => ({ id: m.id, name: m.label })) ?? [];

  if (models.length === 0) {
    return (
      <div className="px-2 py-4 text-center text-sm text-muted-foreground">
        No models available
      </div>
    );
  }

  return (
    <DropdownMenuRadioGroup
      value={selectedRunner === runner.id ? selectedModel : ""}
      onValueChange={(modelId) => onSelect(runner.id, modelId)}
    >
      {models.map((model) => (
        <DropdownMenuRadioItem key={model.id} value={model.id}>
          {model.name}
        </DropdownMenuRadioItem>
      ))}
    </DropdownMenuRadioGroup>
  );
}

export function RunnerModelSelector({
  projectName,
  selectedRunner,
  selectedModel,
  onSelect,
}: RunnerModelSelectorProps) {
  const { data: runnerTypes } = useRunnerTypes(projectName);

  const runners = runnerTypes ?? [];

  const currentRunner = runners.find((r) => r.id === selectedRunner);
  const currentRunnerName = currentRunner?.displayName ?? selectedRunner;

  // Fetch models from API filtered by the current runner's provider.
  // models.json is the single source of truth — no hardcoded fallback lists.
  const { data: modelsData } = useModels(projectName, true, currentRunner?.provider);

  const models = useMemo(() => {
    return modelsData?.models.map((m) => ({ id: m.id, name: m.label })) ?? [];
  }, [modelsData]);

  const currentModel = models.find((m) => m.id === selectedModel);
  const currentModelName = currentModel?.name ?? selectedModel;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="sm"
          className="gap-1 text-xs text-muted-foreground hover:text-foreground h-7 px-2"
        >
          <span className="truncate max-w-[200px]">
            {currentRunnerName} &middot; {currentModelName}
          </span>
          <ChevronDown className="h-3 w-3 opacity-50 flex-shrink-0" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" side="top" sideOffset={4}>
        {runners.map((runner) => (
          <DropdownMenuSub key={runner.id}>
            <DropdownMenuSubTrigger>{runner.displayName}</DropdownMenuSubTrigger>
            <DropdownMenuSubContent>
              <RunnerModelsRadioGroup
                projectName={projectName}
                runner={runner}
                selectedRunner={selectedRunner}
                selectedModel={selectedModel}
                onSelect={onSelect}
              />
            </DropdownMenuSubContent>
          </DropdownMenuSub>
        ))}
        {runners.length === 0 && (
          <div className="px-2 py-4 text-center text-sm text-muted-foreground">
            No runner types available
          </div>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export { getDefaultModel };
