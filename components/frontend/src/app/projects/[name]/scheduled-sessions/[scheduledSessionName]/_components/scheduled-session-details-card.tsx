"use client";

import { formatDistanceToNow } from "date-fns";
import { Info } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { getCronDescription } from "@/lib/cron";
import { INACTIVITY_TIMEOUT_TOOLTIP } from "@/lib/constants";
import { useRunnerTypes } from "@/services/queries/use-runner-types";
import { useModels } from "@/services/queries/use-models";
import type { ScheduledSession } from "@/types/api";

type ScheduledSessionDetailsCardProps = {
  scheduledSession: ScheduledSession;
  projectName: string;
};

export function ScheduledSessionDetailsCard({
  scheduledSession,
  projectName,
}: ScheduledSessionDetailsCardProps) {
  const { sessionTemplate } = scheduledSession;
  const { data: runnerTypes } = useRunnerTypes(projectName);
  const runnerTypeId = sessionTemplate.runnerType;
  const selectedRunner = runnerTypes?.find((rt) => rt.id === runnerTypeId);

  const { data: modelsData } = useModels(projectName, !!selectedRunner, selectedRunner?.provider);
  const modelId = sessionTemplate.llmSettings?.model;
  const modelLabel = modelsData?.models.find((m) => m.id === modelId)?.label;

  const runnerLabel = selectedRunner?.displayName ?? runnerTypeId;
  const modelDisplay = modelLabel ?? modelId;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Details</CardTitle>
      </CardHeader>
      <CardContent>
        <dl className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
          <div>
            <dt className="text-muted-foreground">Name</dt>
            <dd className="font-mono">{scheduledSession.name}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Schedule</dt>
            <dd>
              <span className="font-mono">{scheduledSession.schedule}</span>
              <span className="text-muted-foreground ml-2">
                ({getCronDescription(scheduledSession.schedule)})
              </span>
            </dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Created</dt>
            <dd>
              {formatDistanceToNow(new Date(scheduledSession.creationTimestamp), {
                addSuffix: true,
              })}
            </dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Last Run</dt>
            <dd>
              {scheduledSession.lastScheduleTime
                ? formatDistanceToNow(new Date(scheduledSession.lastScheduleTime), {
                    addSuffix: true,
                  })
                : "Never"}
            </dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Active Sessions</dt>
            <dd>{scheduledSession.activeCount}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Session Reuse</dt>
            <dd>{scheduledSession.reuseLastSession ? "Reuse last session" : "New session each run"}</dd>
          </div>
          {(runnerLabel || modelDisplay) && (
            <div>
              <dt className="text-muted-foreground">Runner / Model</dt>
              <dd>{[runnerLabel, modelDisplay].filter(Boolean).join(" / ")}</dd>
            </div>
          )}
          <div>
            <dt className="text-muted-foreground flex items-center gap-1.5">
              Inactivity Timeout
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <button
                      type="button"
                      aria-label="Inactivity timeout information"
                      className="inline-flex items-center"
                    >
                      <Info className="h-3.5 w-3.5 text-muted-foreground cursor-help" />
                    </button>
                  </TooltipTrigger>
                  <TooltipContent side="right" className="max-w-xs">
                    <p>{INACTIVITY_TIMEOUT_TOOLTIP}</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </dt>
            <dd>
              {sessionTemplate.inactivityTimeout === 0
                ? "Disabled"
                : sessionTemplate.inactivityTimeout
                  ? `${sessionTemplate.inactivityTimeout}s`
                  : "Default (24 hours)"}
            </dd>
          </div>
          {sessionTemplate.activeWorkflow && (
            <div className="sm:col-span-2">
              <dt className="text-muted-foreground">Workflow</dt>
              <dd className="mt-1 space-y-0.5 text-sm">
                <p><span className="text-muted-foreground">Repository:</span> {sessionTemplate.activeWorkflow.gitUrl}</p>
                <p><span className="text-muted-foreground">Branch:</span> {sessionTemplate.activeWorkflow.branch}</p>
                {sessionTemplate.activeWorkflow.path && (
                  <p><span className="text-muted-foreground">Path:</span> {sessionTemplate.activeWorkflow.path}</p>
                )}
              </dd>
            </div>
          )}
          {sessionTemplate.initialPrompt && (
            <div className="sm:col-span-2">
              <dt className="text-muted-foreground">Initial Prompt</dt>
              <dd className="whitespace-pre-wrap mt-1">
                {sessionTemplate.initialPrompt}
              </dd>
            </div>
          )}
          {sessionTemplate.repos && sessionTemplate.repos.length > 0 && (
            <div className="sm:col-span-2">
              <dt className="text-muted-foreground">Context Repositories</dt>
              <dd className="mt-1">
                <ul className="list-disc list-inside space-y-1">
                  {sessionTemplate.repos.map((repo, index) => {
                    let isHttpUrl = false;
                    try {
                      const parsed = new URL(repo.url);
                      isHttpUrl = parsed.protocol === "http:" || parsed.protocol === "https:";
                    } catch { /* not a valid URL */ }
                    return (
                    <li key={index}>
                      {isHttpUrl ? (
                      <a
                        href={repo.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-primary hover:underline"
                      >
                        {repo.url}
                      </a>
                      ) : (
                        <span className="font-mono">{repo.url}</span>
                      )}
                      {repo.branch && (
                        <span className="text-muted-foreground"> ({repo.branch})</span>
                      )}
                      {repo.autoPush && (
                        <span className="text-muted-foreground"> — auto-push</span>
                      )}
                    </li>
                    );
                  })}
                </ul>
              </dd>
            </div>
          )}
        </dl>
      </CardContent>
    </Card>
  );
}
