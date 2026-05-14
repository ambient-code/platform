"use client";

import { useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Info, Loader2, Save } from "lucide-react";
import { useClusterInfo } from "@/hooks/use-cluster-info"; import type { CreateProjectRequest } from "@/types/project";

/** Form does NOT own the mutation -- consumers provide callbacks. */
export type WorkspaceFormProps = {
  onSubmit: (data: CreateProjectRequest) => void;
  onCancel?: () => void;
  isSubmitting: boolean;
  error?: string | null;
  submitLabel?: string;
  showCancel?: boolean;
};

function generateWorkspaceName(displayName: string): string {
  return displayName
    .toLowerCase()
    .replace(/\s+/g, "-")
    .replace(/[^a-z0-9-]/g, "")
    .replace(/-+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 63);
}

function validateProjectName(name: string): string | null {
  const namePattern = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/;

  if (!name) return "Workspace name is required";
  if (name.length > 63) return "Workspace name must be 63 characters or less";
  if (!namePattern.test(name))
    return "Workspace name must be lowercase alphanumeric with hyphens (cannot start or end with hyphen)";
  return null;
}

export function WorkspaceForm({
  onSubmit,
  onCancel,
  isSubmitting,
  error: externalError,
  submitLabel = "Create Workspace",
  showCancel = true,
}: WorkspaceFormProps) {
  const { isOpenShift, isLoading: clusterLoading } = useClusterInfo();
  const [formData, setFormData] = useState<CreateProjectRequest>({ name: "", displayName: "", description: "" });
  const [nameError, setNameError] = useState<string | null>(null);
  const [localError, setLocalError] = useState<string | null>(null);
  const [manuallyEditedName, setManuallyEditedName] = useState(false);
  const displayError = externalError || localError;

  const handleDisplayNameChange = useCallback(
    (displayName: string) => {
      setFormData((prev) => ({
        ...prev,
        displayName,
        name: manuallyEditedName
          ? prev.name
          : generateWorkspaceName(displayName),
      }));
      if (!manuallyEditedName) {
        setNameError(validateProjectName(generateWorkspaceName(displayName)));
      }
    },
    [manuallyEditedName]
  );

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();

      if (isOpenShift && !formData.displayName?.trim()) {
        setLocalError("Display Name is required");
        return;
      }

      const nameValidationError = validateProjectName(formData.name);
      if (nameValidationError) {
        setNameError(nameValidationError);
        return;
      }

      setLocalError(null);

      const payload: CreateProjectRequest = {
        name: formData.name,
        ...(isOpenShift &&
          formData.displayName?.trim() && {
            displayName: formData.displayName.trim(),
          }),
        ...(isOpenShift &&
          formData.description?.trim() && {
            description: formData.description.trim(),
          }),
      };

      onSubmit(payload);
    },
    [formData, isOpenShift, onSubmit]
  );

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {!clusterLoading && !isOpenShift && (
        <Alert>
          <Info className="h-4 w-4" />
          <AlertDescription>
            Running on vanilla Kubernetes. Display name and description fields
            are not available.
          </AlertDescription>
        </Alert>
      )}

      <div className="space-y-4">
        {isOpenShift && (
          <div className="space-y-2">
            <Label htmlFor="displayName">Workspace Name *</Label>
            <Input
              id="displayName"
              data-testid="workspace-display-name-input"
              value={formData.displayName}
              onChange={(e) => handleDisplayNameChange(e.target.value)}
              placeholder="e.g. My Research Workspace"
              maxLength={100}
            />
          </div>
        )}

        {!isOpenShift && (
          <div className="space-y-2">
            <Label htmlFor="name">Workspace Name *</Label>
            <Input
              id="name"
              data-testid="workspace-slug-input"
              value={formData.name}
              onChange={(e) => {
                const name = e.target.value;
                setManuallyEditedName(true);
                setFormData((prev) => ({ ...prev, name }));
                setNameError(validateProjectName(name));
              }}
              placeholder="my-research-workspace"
              className={nameError ? "border-red-500" : ""}
            />
            {nameError && (
              <p className="text-sm text-red-600 dark:text-red-400">
                {nameError}
              </p>
            )}
            <p className="text-sm text-muted-foreground">
              Lowercase alphanumeric with hyphens.
            </p>
          </div>
        )}

        {isOpenShift && (
          <div className="space-y-2">
            <Label htmlFor="description">Description</Label>
            <Textarea
              id="description"
              value={formData.description}
              onChange={(e) =>
                setFormData((prev) => ({
                  ...prev,
                  description: e.target.value,
                }))
              }
              placeholder="Description of the workspace purpose and goals..."
              maxLength={500}
              rows={3}
            />
          </div>
        )}
      </div>

      {displayError && (
        <p className="p-4 bg-red-50 border border-red-200 rounded-md text-red-700 dark:bg-red-950/50 dark:border-red-800 dark:text-red-300">
          {displayError}
        </p>
      )}
      <div className="flex justify-end gap-2 pt-2">
        {showCancel && onCancel && (
          <Button type="button" variant="outline" onClick={onCancel} disabled={isSubmitting}>
            Cancel
          </Button>
        )}
        <Button data-testid="create-workspace-submit" type="submit" disabled={isSubmitting || !!nameError}>
          {isSubmitting
            ? <><Loader2 className="w-4 h-4 mr-2 animate-spin" />Creating...</>
            : <><Save className="w-4 h-4 mr-2" />{submitLabel}</>}
        </Button>
      </div>
    </form>
  );
}
