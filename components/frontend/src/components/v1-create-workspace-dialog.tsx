'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Save, Loader2 } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { successToast, errorToast } from '@/hooks/use-toast';
import { useV1CreateProject } from '@/services/queries/v1';

type V1CreateWorkspaceDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function V1CreateWorkspaceDialog({
  open,
  onOpenChange,
}: V1CreateWorkspaceDialogProps) {
  const router = useRouter();
  const createMutation = useV1CreateProject('default');
  const [error, setError] = useState<string | null>(null);
  const [nameError, setNameError] = useState<string | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    displayName: '',
    description: '',
  });

  const generateWorkspaceName = (displayName: string): string => {
    return displayName
      .toLowerCase()
      .replace(/\s+/g, '-')
      .replace(/[^a-z0-9-]/g, '')
      .replace(/-+/g, '-')
      .replace(/^-+|-+$/g, '')
      .slice(0, 63);
  };

  const validateName = (name: string) => {
    const pattern = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/;
    if (!name) return 'Workspace name is required';
    if (name.length > 63) return 'Workspace name must be 63 characters or less';
    if (!pattern.test(name)) return 'Workspace name must be lowercase alphanumeric with hyphens';
    return null;
  };

  const handleDisplayNameChange = (displayName: string) => {
    const name = generateWorkspaceName(displayName);
    setFormData((prev) => ({ ...prev, displayName, name }));
    setNameError(validateName(name));
  };

  const resetForm = () => {
    setFormData({ name: '', displayName: '', description: '' });
    setNameError(null);
    setError(null);
  };

  const handleClose = () => {
    if (!createMutation.isPending) {
      resetForm();
      onOpenChange(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const nameValidation = validateName(formData.name);
    if (nameValidation) {
      setNameError(nameValidation);
      return;
    }

    setError(null);

    createMutation.mutate(
      {
        name: formData.name,
        ...(formData.displayName?.trim() && { display_name: formData.displayName.trim() }),
        ...(formData.description?.trim() && { description: formData.description.trim() }),
      },
      {
        onSuccess: (project) => {
          successToast(`Workspace "${formData.displayName || formData.name}" created`);
          resetForm();
          onOpenChange(false);
          router.push(`/projects/${encodeURIComponent(project.name)}`);
        },
        onError: (err) => {
          const message = err instanceof Error ? err.message : 'Failed to create workspace';
          setError(message);
          errorToast(message);
        },
      }
    );
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="w-[672px] max-w-[90vw] max-h-[90vh] overflow-y-auto">
        <DialogHeader className="space-y-3">
          <DialogTitle>Create New Workspace</DialogTitle>
          <DialogDescription>
            Create a workspace on the API server. Each workspace is an isolated environment for managing AI-powered sessions.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-8 pt-2">
          <div className="space-y-6">
            <div className="space-y-2">
              <Label htmlFor="v1-displayName">Workspace Name *</Label>
              <Input
                id="v1-displayName"
                value={formData.displayName}
                onChange={(e) => handleDisplayNameChange(e.target.value)}
                placeholder="e.g. My Research Workspace"
                maxLength={100}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="v1-name">Identifier</Label>
              <Input
                id="v1-name"
                value={formData.name}
                disabled
                className="text-muted-foreground bg-muted/50"
              />
              {nameError && <p className="text-sm text-red-600 dark:text-red-400">{nameError}</p>}
              <p className="text-sm text-muted-foreground">
                Auto-generated from workspace name.
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="v1-description">Description</Label>
              <Textarea
                id="v1-description"
                value={formData.description}
                onChange={(e) => setFormData((prev) => ({ ...prev, description: e.target.value }))}
                placeholder="Description of the workspace purpose and goals..."
                maxLength={500}
                rows={3}
              />
            </div>
          </div>

          {error && (
            <div className="p-4 bg-red-50 border border-red-200 rounded-md dark:bg-red-950/50 dark:border-red-800">
              <p className="text-red-700 dark:text-red-300">{error}</p>
            </div>
          )}

          <DialogFooter className="pt-2">
            <Button
              type="button"
              variant="outline"
              onClick={handleClose}
              disabled={createMutation.isPending}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={createMutation.isPending || !!nameError || !formData.name}
            >
              {createMutation.isPending ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Creating...
                </>
              ) : (
                <>
                  <Save className="w-4 h-4 mr-2" />
                  Create Workspace
                </>
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
