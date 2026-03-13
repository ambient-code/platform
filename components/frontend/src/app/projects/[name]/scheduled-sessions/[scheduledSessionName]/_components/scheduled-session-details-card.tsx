import { formatDistanceToNow } from "date-fns";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { getCronDescription } from "@/lib/cron";
import type { ScheduledSession } from "@/types/api";

type ScheduledSessionDetailsCardProps = {
  scheduledSession: ScheduledSession;
};

export function ScheduledSessionDetailsCard({
  scheduledSession,
}: ScheduledSessionDetailsCardProps) {
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
          {scheduledSession.sessionTemplate.llmSettings?.model && (
            <div>
              <dt className="text-muted-foreground">Model</dt>
              <dd>{scheduledSession.sessionTemplate.llmSettings.model}</dd>
            </div>
          )}
          {scheduledSession.sessionTemplate.initialPrompt && (
            <div className="sm:col-span-2">
              <dt className="text-muted-foreground">Initial Prompt</dt>
              <dd className="whitespace-pre-wrap mt-1">
                {scheduledSession.sessionTemplate.initialPrompt}
              </dd>
            </div>
          )}
        </dl>
      </CardContent>
    </Card>
  );
}
