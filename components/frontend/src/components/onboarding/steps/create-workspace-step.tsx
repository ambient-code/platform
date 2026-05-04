"use client";

import { useState } from "react";
import { toast } from "sonner";
import { useCreateProject } from "@/services/queries";
import { WorkspaceForm } from "@/components/workspace-form";
import type { CreateProjectRequest } from "@/types/project";
import type { WizardStepProps } from "../welcome-wizard";

export function CreateWorkspaceStep({ onNext }: WizardStepProps) {
  const createProjectMutation = useCreateProject();
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = (payload: CreateProjectRequest) => {
    setError(null);
    createProjectMutation.mutate(payload, {
      onSuccess: (project) => {
        toast.success(
          `Workspace "${payload.displayName || payload.name}" created`
        );
        onNext({ createdWorkspaceName: project.name });
      },
      onError: (err) => {
        const message =
          err instanceof Error ? err.message : "Failed to create workspace";
        setError(message);
        toast.error(message);
      },
    });
  };

  return (
    <div className="space-y-4">
      <div className="space-y-1">
        <h2 className="text-lg font-semibold">Create your workspace</h2>
        <p className="text-sm text-muted-foreground">
          A workspace is an isolated environment for organizing AI-powered
          sessions, managing API keys, and controlling access.
        </p>
      </div>

      <WorkspaceForm
        onSubmit={handleSubmit}
        isSubmitting={createProjectMutation.isPending}
        error={error}
        submitLabel="Create & Continue"
        showCancel={false}
      />
    </div>
  );
}
