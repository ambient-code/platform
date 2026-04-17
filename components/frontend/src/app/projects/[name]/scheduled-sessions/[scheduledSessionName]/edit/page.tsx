"use client";

import { useParams } from "next/navigation";
import { Loader2 } from "lucide-react";
import { useScheduledSession } from "@/services/queries/use-scheduled-sessions";
import { ScheduledSessionForm } from "../../_components/scheduled-session-form";

export default function EditScheduledSessionPage() {
  const params = useParams<{ name: string; scheduledSessionName: string }>();
  const projectName = params.name;
  const scheduledSessionName = params.scheduledSessionName;

  const { data: scheduledSession, isLoading } = useScheduledSession(projectName, scheduledSessionName);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!scheduledSession) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <p className="text-muted-foreground">Scheduled session not found</p>
      </div>
    );
  }

  return (
    <ScheduledSessionForm
      projectName={projectName}
      mode="edit"
      initialData={scheduledSession}
    />
  );
}
