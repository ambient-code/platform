"use client";

import { formatDistanceToNow } from "date-fns";
import { Plus, RefreshCw, MoreVertical, Play, Pause, PlayCircle, Trash2, Calendar, Loader2, AlertCircle } from "lucide-react";
import { getCronDescription } from "@/lib/cron";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { EmptyState } from "@/components/empty-state";
import { CreateScheduledSessionDialog } from "./create-scheduled-session-dialog";

import {
  useScheduledSessions,
  useDeleteScheduledSession,
  useSuspendScheduledSession,
  useResumeScheduledSession,
  useTriggerScheduledSession,
} from "@/services/queries/use-scheduled-sessions";
import { toast } from "sonner";
import Link from "next/link";

type SchedulesSectionProps = {
  projectName: string;
};

export function SchedulesSection({ projectName }: SchedulesSectionProps) {
  const { data: scheduledSessions, isLoading, error, refetch } = useScheduledSessions(projectName);

  const deleteMutation = useDeleteScheduledSession();
  const suspendMutation = useSuspendScheduledSession();
  const resumeMutation = useResumeScheduledSession();
  const triggerMutation = useTriggerScheduledSession();

  const items = scheduledSessions ?? [];

  const handleTrigger = (name: string) => {
    triggerMutation.mutate(
      { projectName, name },
      {
        onSuccess: () => toast.success(`Triggered "${name}"`),
        onError: (error) => toast.error(error.message || "Failed to trigger"),
      }
    );
  };

  const handleSuspend = (name: string) => {
    suspendMutation.mutate(
      { projectName, name },
      {
        onSuccess: () => toast.success(`Suspended "${name}"`),
        onError: (error) => toast.error(error.message || "Failed to suspend"),
      }
    );
  };

  const handleResume = (name: string) => {
    resumeMutation.mutate(
      { projectName, name },
      {
        onSuccess: () => toast.success(`Resumed "${name}"`),
        onError: (error) => toast.error(error.message || "Failed to resume"),
      }
    );
  };

  const handleDelete = (name: string) => {
    if (!confirm(`Delete scheduled session "${name}"? This action cannot be undone.`)) return;
    deleteMutation.mutate(
      { projectName, name },
      {
        onSuccess: () => toast.success(`Deleted "${name}"`),
        onError: (error) => toast.error(error.message || "Failed to delete"),
      }
    );
  };

  return (
    <Card className="flex-1">
      <CardHeader>
        <div className="flex items-start justify-between">
          <div>
            <CardTitle>Scheduled Sessions</CardTitle>
            <CardDescription>
              Recurring sessions that run on a schedule
            </CardDescription>
          </div>
          <div className="flex gap-2">
            <CreateScheduledSessionDialog
              projectName={projectName}
              onSuccess={() => refetch()}
              trigger={
                <Button>
                  <Plus className="w-4 h-4 mr-2" />
                  New Scheduled Session
                </Button>
              }
            />
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="flex justify-center p-8">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        ) : error ? (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>Failed to load scheduled sessions</AlertDescription>
          </Alert>
        ) : items.length === 0 ? (
          <EmptyState
            icon={Calendar}
            title="No scheduled sessions"
            description="Create a scheduled session to run sessions on a recurring schedule"
          />
        ) : (
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="min-w-[180px]">Name</TableHead>
                  <TableHead>Schedule</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="hidden md:table-cell">Last Run</TableHead>
                  <TableHead className="w-[50px]">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {items.map((ss) => {
                  const isActionPending =
                    (deleteMutation.isPending && deleteMutation.variables?.name === ss.name) ||
                    (suspendMutation.isPending && suspendMutation.variables?.name === ss.name) ||
                    (resumeMutation.isPending && resumeMutation.variables?.name === ss.name) ||
                    (triggerMutation.isPending && triggerMutation.variables?.name === ss.name);

                  return (
                    <TableRow key={ss.name}>
                      <TableCell className="font-medium">
                        <Link
                          href={`/projects/${projectName}/scheduled-sessions/${ss.name}`}
                          className="text-link hover:underline hover:text-link-hover transition-colors"
                        >
                          <div>
                            <div className="font-medium">{ss.displayName || ss.name}</div>
                            {ss.displayName && (
                              <div className="text-xs text-muted-foreground font-normal">{ss.name}</div>
                            )}
                          </div>
                        </Link>
                      </TableCell>
                      <TableCell>
                        <div className="text-sm">{getCronDescription(ss.schedule)}</div>
                        <div className="text-xs text-muted-foreground font-mono">{ss.schedule}</div>
                      </TableCell>
                      <TableCell>
                        {ss.suspend ? (
                          <Badge variant="secondary">Suspended</Badge>
                        ) : (
                          <Badge variant="default">Active</Badge>
                        )}
                      </TableCell>
                      <TableCell className="hidden md:table-cell">
                        {ss.lastScheduleTime
                          ? formatDistanceToNow(new Date(ss.lastScheduleTime), { addSuffix: true })
                          : <span className="text-muted-foreground">Never</span>}
                      </TableCell>
                      <TableCell>
                        {isActionPending ? (
                          <Button variant="ghost" size="sm" className="h-8 w-8 p-0" disabled>
                            <RefreshCw className="h-4 w-4 animate-spin" />
                          </Button>
                        ) : (
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                                <MoreVertical className="h-4 w-4" />
                                <span className="sr-only">Open menu</span>
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                              <DropdownMenuItem onClick={() => handleTrigger(ss.name)}>
                                <PlayCircle className="h-4 w-4 mr-2" />
                                Trigger Now
                              </DropdownMenuItem>
                              {ss.suspend ? (
                                <DropdownMenuItem onClick={() => handleResume(ss.name)}>
                                  <Play className="h-4 w-4 mr-2" />
                                  Resume
                                </DropdownMenuItem>
                              ) : (
                                <DropdownMenuItem onClick={() => handleSuspend(ss.name)}>
                                  <Pause className="h-4 w-4 mr-2" />
                                  Suspend
                                </DropdownMenuItem>
                              )}
                              <DropdownMenuItem onClick={() => handleDelete(ss.name)} className="text-red-600">
                                <Trash2 className="h-4 w-4 mr-2" />
                                Delete
                              </DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        )}
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
