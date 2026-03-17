"use client";

import { useCallback, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import { NewSessionView } from "../sessions/[sessionName]/components/new-session-view";
import { CustomWorkflowDialog } from "../sessions/[sessionName]/components/modals/custom-workflow-dialog";
import { useCreateSession } from "@/services/queries";
import { useOOTBWorkflows } from "@/services/queries/use-workflows";

export default function NewSessionPage() {
  const params = useParams();
  const router = useRouter();
  const projectName = params?.name as string;

  const { data: ootbWorkflows = [] } = useOOTBWorkflows(projectName);
  const createSessionMutation = useCreateSession();
  const [customWorkflowDialogOpen, setCustomWorkflowDialogOpen] = useState(false);
  const [customWorkflow, setCustomWorkflow] = useState<{ gitUrl: string; branch: string; path: string } | null>(null);

  const handleCreateNewSession = useCallback(
    (config: {
      prompt: string;
      runner: string;
      model: string;
      workflow?: string;
      repos?: Array<{ url: string }>;
    }) => {
      const workflowConfig = config.workflow === "custom" && customWorkflow
        ? { gitUrl: customWorkflow.gitUrl, branch: customWorkflow.branch, path: customWorkflow.path }
        : config.workflow
          ? ootbWorkflows.find((w) => w.id === config.workflow)
          : undefined;

      createSessionMutation.mutate(
        {
          projectName,
          data: {
            initialPrompt: config.prompt,
            runnerType: config.runner,
            llmSettings: { model: config.model },
            ...(workflowConfig
              ? {
                  activeWorkflow: {
                    gitUrl: workflowConfig.gitUrl,
                    branch: workflowConfig.branch || "main",
                    path: workflowConfig.path,
                  },
                }
              : {}),
            ...(config.repos && config.repos.length > 0
              ? {
                  repos: config.repos.map((r) => ({ url: r.url })),
                }
              : {}),
          },
        },
        {
          onSuccess: (session) => {
            router.push(
              `/projects/${encodeURIComponent(projectName)}/sessions/${session.metadata.name}`
            );
          },
          onError: (err) => {
            toast.error(
              err instanceof Error
                ? err.message
                : "Failed to create session"
            );
          },
        }
      );
    },
    [projectName, ootbWorkflows, customWorkflow, createSessionMutation, router]
  );

  if (!projectName) return null;

  return (
    <div className="h-full overflow-auto">
      <NewSessionView
        projectName={projectName}
        onCreateSession={handleCreateNewSession}
        ootbWorkflows={ootbWorkflows}
        onLoadCustomWorkflow={() => setCustomWorkflowDialogOpen(true)}
        isSubmitting={createSessionMutation.isPending}
      />
      <CustomWorkflowDialog
        open={customWorkflowDialogOpen}
        onOpenChange={setCustomWorkflowDialogOpen}
        onSubmit={(url, branch, path) => {
          setCustomWorkflow({ gitUrl: url, branch: branch || "main", path: path || "" });
          setCustomWorkflowDialogOpen(false);
        }}
      />
    </div>
  );
}
