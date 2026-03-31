"use client";

import { useState } from "react";
import { Pencil, Check, X } from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import { Button } from "@/components/ui/button";
import { SessionPhaseBadge } from "@/components/status-badge";
import { RunnerModelSelector } from "../runner-model-selector";
import type { AgenticSession } from "@/types/agentic-session";

type SessionDetailsProps = {
  session: AgenticSession;
  projectName: string;
  onEditName?: () => void;
  onModelUpdate?: (model: string) => Promise<void>;
};

export function SessionDetails({ session, projectName, onEditName, onModelUpdate }: SessionDetailsProps) {
  const phase = session.status?.phase || "Pending";
  const stoppedReason = session.status?.stoppedReason;
  const displayName = session.spec.displayName || session.metadata.name;
  const model = session.spec.llmSettings?.model || "—";
  const runnerType = session.spec.environmentVariables?.RUNNER_TYPE || "ambient-runner";
  const createdAt = session.metadata.creationTimestamp;

  const [editingModel, setEditingModel] = useState(false);
  const [selectedModel, setSelectedModel] = useState(model);
  const [isUpdating, setIsUpdating] = useState(false);

  const canEditModel = phase === "Running" && onModelUpdate;

  const handleModelSave = async () => {
    if (!onModelUpdate || selectedModel === model) {
      setEditingModel(false);
      return;
    }

    setIsUpdating(true);
    try {
      await onModelUpdate(selectedModel);
      setEditingModel(false);
    } catch (error) {
      console.error("Failed to update model:", error);
      setSelectedModel(model); // Revert on error
    } finally {
      setIsUpdating(false);
    }
  };

  const handleModelCancel = () => {
    setSelectedModel(model);
    setEditingModel(false);
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-1">
        <h3 className="text-lg font-semibold">Session Details</h3>
      </div>
      <p className="text-sm text-muted-foreground mb-4">
        View and manage this session.
      </p>

      <div className="space-y-0 border rounded-lg divide-y">
        <Row label="Name">
          <div className="flex items-center gap-2">
            <span className="font-medium">{displayName}</span>
            {onEditName && (
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                onClick={onEditName}
              >
                <Pencil className="h-3.5 w-3.5" />
              </Button>
            )}
          </div>
        </Row>
        <Row label="Status">
          <SessionPhaseBadge phase={phase} stoppedReason={stoppedReason} />
        </Row>
        <Row label="Session ID">
          <span className="font-mono text-sm text-muted-foreground">
            {session.metadata.name}
          </span>
        </Row>
        <Row label="Model">
          {editingModel ? (
            <div className="flex items-center gap-2">
              <RunnerModelSelector
                projectName={projectName}
                selectedRunner={runnerType}
                selectedModel={selectedModel}
                onSelect={(_, newModel) => setSelectedModel(newModel)}
              />
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                onClick={handleModelSave}
                disabled={isUpdating}
              >
                <Check className="h-3.5 w-3.5" />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                onClick={handleModelCancel}
                disabled={isUpdating}
              >
                <X className="h-3.5 w-3.5" />
              </Button>
            </div>
          ) : (
            <div className="flex items-center gap-2">
              <span className="text-sm text-muted-foreground truncate max-w-[200px]">
                {model}
              </span>
              {canEditModel && (
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6"
                  onClick={() => setEditingModel(true)}
                >
                  <Pencil className="h-3.5 w-3.5" />
                </Button>
              )}
            </div>
          )}
        </Row>
        <Row label="Created">
          <span className="text-sm text-muted-foreground">
            {formatDistanceToNow(new Date(createdAt), { addSuffix: true })}
          </span>
        </Row>
      </div>
    </div>
  );
}

function Row({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between px-4 py-3">
      <span className="text-sm text-muted-foreground">{label}</span>
      <div className="flex items-center">{children}</div>
    </div>
  );
}
