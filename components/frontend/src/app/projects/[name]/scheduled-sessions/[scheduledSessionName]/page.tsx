"use client";

import { useParams, useRouter } from "next/navigation";
import {
  ArrowLeft,
  Calendar,
  Loader2,
  MoreVertical,
  Pause,
  Play,
  PlayCircle,
  Trash2,
} from "lucide-react";
import { getCronDescription } from "@/lib/cron";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

import {
  useScheduledSession,
  useScheduledSessionRuns,
  useDeleteScheduledSession,
  useSuspendScheduledSession,
  useResumeScheduledSession,
  useTriggerScheduledSession,
} from "@/services/queries/use-scheduled-sessions";
import { toast } from "sonner";

import { ScheduledSessionDetailsCard } from "./_components/scheduled-session-details-card";
import { ScheduledSessionRunsTable } from "./_components/scheduled-session-runs-table";

export default function ScheduledSessionDetailPage() {
  const params = useParams<{ name: string; scheduledSessionName: string }>();
  const router = useRouter();
  const projectName = params.name;
  const scheduledSessionName = params.scheduledSessionName;

  const { data: scheduledSession, isLoading } = useScheduledSession(
    projectName,
    scheduledSessionName
  );
  const { data: runs } = useScheduledSessionRuns(projectName, scheduledSessionName);

  const deleteMutation = useDeleteScheduledSession();
  const suspendMutation = useSuspendScheduledSession();
  const resumeMutation = useResumeScheduledSession();
  const triggerMutation = useTriggerScheduledSession();

  const handleTrigger = () => {
    triggerMutation.mutate(
      { projectName, name: scheduledSessionName },
      {
        onSuccess: () => toast.success("Session triggered"),
        onError: (error) => toast.error(error.message || "Failed to trigger"),
      }
    );
  };

  const handleSuspendResume = () => {
    if (!scheduledSession) return;
    if (scheduledSession.suspend) {
      resumeMutation.mutate(
        { projectName, name: scheduledSessionName },
        {
          onSuccess: () => toast.success("Schedule resumed"),
          onError: (error) => toast.error(error.message || "Failed to resume"),
        }
      );
    } else {
      suspendMutation.mutate(
        { projectName, name: scheduledSessionName },
        {
          onSuccess: () => toast.success("Schedule suspended"),
          onError: (error) => toast.error(error.message || "Failed to suspend"),
        }
      );
    }
  };

  const handleDelete = () => {
    if (!confirm(`Delete scheduled session "${scheduledSessionName}"? This action cannot be undone.`)) return;
    deleteMutation.mutate(
      { projectName, name: scheduledSessionName },
      {
        onSuccess: () => {
          toast.success("Scheduled session deleted");
          router.push(`/projects/${encodeURIComponent(projectName)}?section=schedules`);
        },
        onError: (error) => toast.error(error.message || "Failed to delete"),
      }
    );
  };

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

  const isActionPending =
    deleteMutation.isPending ||
    suspendMutation.isPending ||
    resumeMutation.isPending ||
    triggerMutation.isPending;

  return (
    <div className="space-y-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => router.push(`/projects/${encodeURIComponent(projectName)}?section=schedules`)}
          >
            <ArrowLeft className="h-4 w-4 mr-1" />
            Back
          </Button>
          <div>
            <h1 className="text-2xl font-bold">
              {scheduledSession.displayName || scheduledSession.name}
            </h1>
            <div className="flex items-center gap-2 mt-1">
              <Calendar className="h-4 w-4 text-muted-foreground" />
              <span className="text-sm text-muted-foreground">
                {getCronDescription(scheduledSession.schedule)}
              </span>
              {scheduledSession.suspend ? (
                <Badge variant="secondary">Suspended</Badge>
              ) : (
                <Badge variant="default">Active</Badge>
              )}
            </div>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={handleTrigger} disabled={isActionPending}>
            <PlayCircle className="h-4 w-4 mr-2" />
            Trigger Now
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="icon" disabled={isActionPending}>
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={handleSuspendResume}>
                {scheduledSession.suspend ? (
                  <>
                    <Play className="h-4 w-4 mr-2" />
                    Resume
                  </>
                ) : (
                  <>
                    <Pause className="h-4 w-4 mr-2" />
                    Suspend
                  </>
                )}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={handleDelete} className="text-red-600">
                <Trash2 className="h-4 w-4 mr-2" />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <ScheduledSessionDetailsCard scheduledSession={scheduledSession} />
      <ScheduledSessionRunsTable runs={runs} projectName={projectName} />
    </div>
  );
}
