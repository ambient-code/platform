"use client";

import { useState, useRef, useCallback, useEffect } from "react";
import { MessageSquarePlus, ArrowUp, Loader2, Plus, GitBranch, Upload, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Textarea } from "@/components/ui/textarea";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { RunnerModelSelector, getDefaultModel } from "./runner-model-selector";
import { WorkflowSelector } from "./workflow-selector";
import { AddContextModal } from "./modals/add-context-modal";
import { useRunnerTypes } from "@/services/queries/use-runner-types";
import { DEFAULT_RUNNER_TYPE_ID } from "@/services/api/runner-types";
import type { WorkflowConfig } from "../lib/types";

type PendingRepo = {
  url: string;
  name: string;
};

type NewSessionViewProps = {
  projectName: string;
  onCreateSession: (config: {
    prompt: string;
    runner: string;
    model: string;
    workflow?: string;
    repos?: Array<{ url: string }>;
  }) => void;
  ootbWorkflows: WorkflowConfig[];
  onLoadCustomWorkflow?: () => void;
  isSubmitting?: boolean;
};

export function NewSessionView({
  projectName,
  onCreateSession,
  ootbWorkflows,
  onLoadCustomWorkflow,
  isSubmitting = false,
}: NewSessionViewProps) {
  const { data: runnerTypes } = useRunnerTypes(projectName);

  const [prompt, setPrompt] = useState("");
  const [selectedRunner, setSelectedRunner] = useState<string>(DEFAULT_RUNNER_TYPE_ID);
  const [selectedModel, setSelectedModel] = useState(() =>
    getDefaultModel(DEFAULT_RUNNER_TYPE_ID)
  );

  // Once runner types load, default to the first available if current selection isn't available
  useEffect(() => {
    if (runnerTypes && runnerTypes.length > 0) {
      const isCurrentAvailable = runnerTypes.some((r) => r.id === selectedRunner);
      if (!isCurrentAvailable) {
        const firstRunner = runnerTypes[0].id;
        setSelectedRunner(firstRunner);
        setSelectedModel(getDefaultModel(firstRunner));
      }
    }
  }, [runnerTypes, selectedRunner]);
  const [selectedWorkflow, setSelectedWorkflow] = useState("none");
  const [pendingRepos, setPendingRepos] = useState<PendingRepo[]>([]);
  const [contextModalOpen, setContextModalOpen] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const addPendingRepo = (url: string) => {
    if (pendingRepos.some((r) => r.url === url)) return;
    const name = url.replace(/\/+$/, "").split("/").pop()?.replace(/\.git$/, "") || url;
    setPendingRepos((prev) => [...prev, { url, name }]);
  };

  const removePendingRepo = (url: string) => {
    setPendingRepos((prev) => prev.filter((r) => r.url !== url));
  };

  const handleSubmit = useCallback(() => {
    const trimmed = prompt.trim();
    const hasWorkflow = selectedWorkflow !== "none";

    // Require either a prompt OR a workflow with startupPrompt
    if (!trimmed && !hasWorkflow) return;

    onCreateSession({
      prompt: trimmed,
      runner: selectedRunner,
      model: selectedModel,
      workflow: hasWorkflow ? selectedWorkflow : undefined,
      repos: pendingRepos.length > 0 ? pendingRepos.map((r) => ({ url: r.url })) : undefined,
    });
  }, [prompt, selectedRunner, selectedModel, selectedWorkflow, pendingRepos, onCreateSession]);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  };

  const handleRunnerModelSelect = (runner: string, model: string) => {
    setSelectedRunner(runner);
    setSelectedModel(model);
  };

  return (
    <div className="h-full flex items-center justify-center p-8">
      <div className="w-full max-w-2xl space-y-4">
        {/* Header */}
        <div className="text-center space-y-2">
          <div className="flex justify-center mb-4">
            <MessageSquarePlus className="h-10 w-10 text-primary" />
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">
            What are you working on?
          </h1>
          <p className="text-muted-foreground">
            Start a new session by typing a message or selecting a workflow.
          </p>
        </div>

        {/* Input area */}
        <div className="relative border rounded-lg bg-background shadow-sm focus-within:ring-1 focus-within:ring-ring">
          <Textarea
            ref={textareaRef}
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Describe what you'd like to work on..."
            className="min-h-[100px] resize-none border-0 focus-visible:ring-0 focus-visible:ring-offset-0 pb-12"
          />
          <div className="absolute bottom-0 left-0 right-0 flex items-center justify-between px-2 py-2">
            <div className="flex items-center gap-1">
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="sm" className="h-7 w-7 p-0">
                    <Plus className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start" side="top">
                  <DropdownMenuItem onClick={() => setContextModalOpen(true)}>
                    <GitBranch className="w-4 h-4 mr-2" />
                    Add Repository
                  </DropdownMenuItem>
                  <DropdownMenuItem disabled className="text-muted-foreground">
                    <Upload className="w-4 h-4 mr-2" />
                    Upload File
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
              <RunnerModelSelector
                projectName={projectName}
                selectedRunner={selectedRunner}
                selectedModel={selectedModel}
                onSelect={handleRunnerModelSelect}
              />
            </div>
            <div className="flex items-center gap-1">
              <WorkflowSelector
                activeWorkflow={null}
                selectedWorkflow={selectedWorkflow}
                workflowActivating={false}
                ootbWorkflows={ootbWorkflows}
                onWorkflowChange={setSelectedWorkflow}
                onLoadCustom={onLoadCustomWorkflow}
              />
              <Button
                size="icon"
                className="h-8 w-8 rounded-full bg-primary hover:bg-primary/90 text-primary-foreground"
                disabled={(!prompt.trim() && selectedWorkflow === "none") || isSubmitting}
                onClick={handleSubmit}
              >
                {isSubmitting ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <ArrowUp className="h-4 w-4" />
                )}
              </Button>
            </div>
          </div>
        </div>

        {/* Pending repo badges */}
        {pendingRepos.length > 0 && (
          <div className="flex gap-2 flex-wrap">
            {pendingRepos.map((repo) => (
              <TooltipProvider key={repo.url}>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Badge variant="secondary" className="gap-1">
                      <GitBranch className="h-3 w-3" />
                      {repo.name}
                      <X
                        className="h-3 w-3 cursor-pointer"
                        onClick={() => removePendingRepo(repo.url)}
                      />
                    </Badge>
                  </TooltipTrigger>
                  <TooltipContent>{repo.url}</TooltipContent>
                </Tooltip>
              </TooltipProvider>
            ))}
          </div>
        )}

      </div>

      <AddContextModal
        open={contextModalOpen}
        onOpenChange={setContextModalOpen}
        onAddRepository={async (url) => {
          addPendingRepo(url);
          setContextModalOpen(false);
        }}
      />
    </div>
  );
}
