"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { toast } from "sonner";
import { useCreateProject } from "@/services/queries";
import { WorkspaceForm } from "@/components/workspace-form";
import type { CreateProjectRequest } from "@/types/project";

type CreateWorkspaceDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function CreateWorkspaceDialog({
  open,
  onOpenChange,
}: CreateWorkspaceDialogProps) {
  const router = useRouter();
  const createProjectMutation = useCreateProject();
  const [error, setError] = useState<string | null>(null);

  const handleClose = () => {
    if (!createProjectMutation.isPending) {
      setError(null);
      onOpenChange(false);
    }
  };

  const handleSubmit = (payload: CreateProjectRequest) => {
    setError(null);
    createProjectMutation.mutate(payload, {
      onSuccess: (project) => {
        toast.success(
          `Workspace "${payload.displayName || payload.name}" created successfully`
        );
        setError(null);
        onOpenChange(false);
        router.push(`/projects/${encodeURIComponent(project.name)}`);
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
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="w-[672px] max-w-[90vw] max-h-[90vh] overflow-y-auto">
        <DialogHeader className="space-y-3">
          <DialogTitle>Create New Workspace</DialogTitle>
          <DialogDescription>
            A workspace is an isolated environment where your team can create and
            manage AI-powered agentic sessions. Each workspace has its own
            settings, permissions, and resources.
          </DialogDescription>
        </DialogHeader>

        <WorkspaceForm
          onSubmit={handleSubmit}
          onCancel={handleClose}
          isSubmitting={createProjectMutation.isPending}
          error={error}
        />
      </DialogContent>
    </Dialog>
  );
}
