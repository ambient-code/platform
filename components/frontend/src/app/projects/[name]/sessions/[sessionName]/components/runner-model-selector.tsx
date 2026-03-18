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

type ModelOption = {
  id: string;
  name: string;
};

const MODELS_BY_RUNNER: Record<string, ModelOption[]> = {
  "claude-code": [
    { id: "claude-haiku-4-5", name: "Claude Haiku 4.5" },
    { id: "claude-sonnet-4-5", name: "Claude Sonnet 4.5" },
    { id: "claude-opus-4-5", name: "Claude Opus 4.5" },
  ],
  "claude-agent-sdk": [
    { id: "claude-haiku-4-5", name: "Claude Haiku 4.5" },
    { id: "claude-sonnet-4-5", name: "Claude Sonnet 4.5" },
    { id: "claude-opus-4-5", name: "Claude Opus 4.5" },
  ],
  "gemini-cli": [
    { id: "gemini-2.0-flash", name: "Gemini 2.0 Flash" },
    { id: "gemini-2.5-pro", name: "Gemini 2.5 Pro" },
  ],
  "openai-codex": [
    { id: "gpt-4o", name: "GPT-4o" },
    { id: "gpt-4.1", name: "GPT-4.1" },
  ],
  amp: [
    { id: "claude-sonnet-4-5", name: "Claude Sonnet 4.5" },
    { id: "gpt-4o", name: "GPT-4o" },
  ],
};

function getModelsForRunner(runnerId: string): ModelOption[] {
  return MODELS_BY_RUNNER[runnerId] ?? [{ id: "default", name: "Default" }];
}

function getDefaultModel(runnerId: string): string {
  const models = getModelsForRunner(runnerId);
  return models[1]?.id ?? models[0]?.id ?? "default";
}

type RunnerModelSelectorProps = {
  projectName: string;
  selectedRunner: string;
  selectedModel: string;
  onSelect: (runner: string, model: string) => void;
};

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

  const models = useMemo(() => getModelsForRunner(selectedRunner), [selectedRunner]);
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
        {runners.map((runner) => {
          const runnerModels = getModelsForRunner(runner.id);
          return (
            <DropdownMenuSub key={runner.id}>
              <DropdownMenuSubTrigger>{runner.displayName}</DropdownMenuSubTrigger>
              <DropdownMenuSubContent>
                <DropdownMenuRadioGroup
                  value={selectedRunner === runner.id ? selectedModel : ""}
                  onValueChange={(modelId) => onSelect(runner.id, modelId)}
                >
                  {runnerModels.map((model) => (
                    <DropdownMenuRadioItem key={model.id} value={model.id}>
                      {model.name}
                    </DropdownMenuRadioItem>
                  ))}
                </DropdownMenuRadioGroup>
              </DropdownMenuSubContent>
            </DropdownMenuSub>
          );
        })}
        {runners.length === 0 && (
          <div className="px-2 py-4 text-center text-sm text-muted-foreground">
            No runner types available
          </div>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export { getDefaultModel, getModelsForRunner };
