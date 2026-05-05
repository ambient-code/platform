"use client";

import { useMemo, useState, useCallback } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { AgentStatusIndicator, agentStatusLabel } from "@/components/agent-status-indicator";
import { deriveAgentStatusFromPhase } from "@/hooks/use-agent-status";
import { SessionStatusDot, sessionPhaseLabel } from "@/components/session-status-dot";
import { EditSessionNameDialog } from "@/components/edit-session-name-dialog";
import { DestructiveConfirmationDialog } from "@/components/confirmation-dialog";
import {
  Plus,
  PanelLeftClose,
  ChevronLeft,
  LayoutList,
  Calendar,
  Share2,
  Key,
  Settings,
  MoreHorizontal,
  MoreVertical,
  Cpu,
  Clock,
  MessageSquare,
  RefreshCw,
  Pencil,
  Square,
  ArrowRight,
  Trash2,
  User,
} from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import {
  useSessionsPaginated,
  useStopSession,
  useDeleteSession,
  useContinueSession,
  useUpdateSessionDisplayName,
} from "@/services/queries/use-sessions";
import { useProjectAccess } from "@/services/queries/use-project-access";
import { useCurrentUser } from "@/services/queries/use-auth";
import { useVersion } from "@/services/queries/use-version";
import { useLocalStorage } from "@/hooks/use-local-storage";
import { cn } from "@/lib/utils";
import type { AgenticSession } from "@/types/api";

type StatusFilter = "all" | "running" | "completed" | "failed";

type SessionsSidebarProps = {
  projectName: string;
  currentSessionName: string;
  collapsed: boolean;
  onCollapse?: () => void;
  onNewSession?: () => void;
  onSessionSelect?: () => void;
};

const INITIAL_RECENTS_COUNT = 10;

type NavItem = {
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  href: string;
};

/** Get the most relevant activity timestamp for sorting (prefer lastActivityTime, fall back to creationTimestamp). */
function getActivityTime(session: AgenticSession): number {
  const lastActivity = session.status?.lastActivityTime;
  if (lastActivity) return new Date(lastActivity).getTime();
  return new Date(session.metadata.creationTimestamp).getTime();
}

export function SessionsSidebar({
  projectName,
  currentSessionName,
  collapsed,
  onCollapse,
  onNewSession,
  onSessionSelect,
}: SessionsSidebarProps) {
  const router = useRouter();
  const pathname = usePathname();
  const { data: version } = useVersion();
  const [showAll, setShowAll] = useState(false);
  const { data, isLoading, isFetching, dataUpdatedAt, refetch } = useSessionsPaginated(
    collapsed ? "" : projectName,
    { limit: 20 },
  );

  const { data: access } = useProjectAccess(projectName);
  const canDelete = access?.userRole === "admin";
  const canModify = !!access?.userRole && access.userRole !== "view";

  const stopMutation = useStopSession();
  const deleteMutation = useDeleteSession();
  const continueMutation = useContinueSession();
  const updateDisplayNameMutation = useUpdateSessionDisplayName();

  const { data: currentUser } = useCurrentUser();

  const [editingSession, setEditingSession] = useState<{ name: string; displayName: string } | null>(null);
  const [deletingSessionName, setDeletingSessionName] = useState<string | null>(null);
  const [mineOnly, setMineOnly] = useLocalStorage(`acp:sidebar:mine:${projectName}`, false);
  const [statusFilter, setStatusFilter] = useLocalStorage<StatusFilter>(`acp:sidebar:status:${projectName}`, "all");

  const toggleMineOnly = useCallback(() => setMineOnly((prev: boolean) => !prev), [setMineOnly]);
  const toggleStatusFilter = useCallback(
    (filter: StatusFilter) => setStatusFilter((prev: StatusFilter) => prev === filter ? "all" : filter),
    [setStatusFilter],
  );

  const sessions = useMemo(() => {
    const items = data?.items ?? [];
    return [...items].sort((a, b) => getActivityTime(b) - getActivityTime(a));
  }, [data?.items]);

  const filteredSessions = useMemo(() => {
    let result = sessions;

    if (mineOnly && currentUser?.userId) {
      result = result.filter(
        (s) => s.spec.userContext?.userId === currentUser.userId,
      );
    }

    if (statusFilter !== "all") {
      result = result.filter((s) => {
        const phase = s.status?.phase;
        switch (statusFilter) {
          case "running":
            return phase === "Running" || phase === "Pending" || phase === "Creating";
          case "completed":
            return phase === "Completed" || phase === "Stopped";
          case "failed":
            return phase === "Failed";
          default:
            return true;
        }
      });
    }

    return result;
  }, [sessions, mineOnly, currentUser?.userId, statusFilter]);

  const visibleSessions = useMemo(() => {
    if (showAll) return filteredSessions;
    return filteredSessions.slice(0, INITIAL_RECENTS_COUNT);
  }, [filteredSessions, showAll]);

  const hasMore = filteredSessions.length > INITIAL_RECENTS_COUNT && !showAll;
  const isFiltered = mineOnly || statusFilter !== "all";

  const navItems: NavItem[] = useMemo(
    () => [
      {
        label: "Sessions",
        icon: LayoutList,
        href: `/projects/${projectName}/sessions`,
      },
      {
        label: "Schedules",
        icon: Calendar,
        href: `/projects/${projectName}/scheduled-sessions`,
      },
      {
        label: "Pair Prompting",
        icon: Share2,
        href: `/projects/${projectName}/permissions`,
      },
      {
        label: "Access Keys",
        icon: Key,
        href: `/projects/${projectName}/keys`,
      },
      {
        label: "Workspace Settings",
        icon: Settings,
        href: `/projects/${projectName}/settings`,
      },
    ],
    [projectName]
  );

  if (collapsed) return null;

  const handleNewSession = () => {
    if (onNewSession) {
      onNewSession();
    } else {
      router.push(`/projects/${projectName}/new`);
    }
  };

  const handleStop = (sessionName: string) => {
    stopMutation.mutate(
      { projectName, sessionName },
      {
        onSuccess: () => toast.success(`Session stopped`),
        onError: () => toast.error(`Failed to stop session`),
      },
    );
  };

  const handleContinue = (sessionName: string) => {
    continueMutation.mutate(
      { projectName, parentSessionName: sessionName },
      {
        onSuccess: () => toast.success(`Session restarted`),
        onError: () => toast.error(`Failed to continue session`),
      },
    );
  };

  const handleDelete = (sessionName: string) => {
    setDeletingSessionName(sessionName);
  };

  const confirmDelete = () => {
    if (!deletingSessionName) return;
    const isCurrentSession = deletingSessionName === currentSessionName;
    deleteMutation.mutate(
      { projectName, sessionName: deletingSessionName },
      {
        onSuccess: () => {
          toast.success(`Session deleted`);
          setDeletingSessionName(null);
          if (isCurrentSession) {
            router.push(`/projects/${projectName}/sessions`);
          }
        },
        onError: () => toast.error(`Failed to delete session`),
      },
    );
  };

  const handleEditName = (sessionName: string, currentDisplayName: string) => {
    setEditingSession({ name: sessionName, displayName: currentDisplayName });
  };

  const handleSaveEditName = (newName: string) => {
    if (!editingSession) return;
    updateDisplayNameMutation.mutate(
      { projectName, sessionName: editingSession.name, displayName: newName },
      {
        onSuccess: () => {
          toast.success(`Session renamed`);
          setEditingSession(null);
        },
        onError: () => toast.error(`Failed to rename session`),
      },
    );
  };

  const sessionHref = (sessionName: string) =>
    `/projects/${projectName}/sessions/${sessionName}`;

  return (
    <div className="flex flex-col h-full">
      {/* Branding row */}
      <div className="flex items-center justify-between h-14 px-3 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <Link href="/" className="flex items-end gap-2 min-w-0">
          <span className="text-base font-bold truncate">Ambient Code Platform</span>
          {version && (
            <span className="text-[0.65rem] text-muted-foreground/60 pb-0.5 flex-shrink-0">
              {version}
            </span>
          )}
        </Link>
        {onCollapse && (
          <Button
            variant="ghost"
            size="sm"
            className="h-8 w-8 p-0 flex-shrink-0"
            onClick={onCollapse}
            title="Hide sidebar"
          >
            <PanelLeftClose className="h-4 w-4" />
          </Button>
        )}
      </div>

      {/* New Session button */}
      <div className="flex items-center gap-2 p-3 border-b">
        <Button
          variant="outline"
          size="sm"
          className="flex-1"
          onClick={handleNewSession}
        >
          <Plus className="w-4 h-4 mr-1" />
          New Session
        </Button>
      </div>

      {/* Workspace Navigation */}
      <div className="p-2 space-y-0.5">
        <Link href="/projects" className="block">
          <Button
            variant="ghost"
            size="sm"
            className="w-full justify-between text-muted-foreground hover:text-foreground"
          >
            <span className="flex items-center">
              <ChevronLeft className="w-4 h-4 mr-2" />
              Workspaces
            </span>
            <span className="text-xs font-semibold text-foreground truncate max-w-[60%]" title={projectName}>
              {projectName}
            </span>
          </Button>
        </Link>

        {navItems.map((item) => {
          const isActive = pathname?.startsWith(item.href);
          const Icon = item.icon;

          return (
            <Link key={item.label} href={item.href} className="block">
              <Button
                variant="ghost"
                size="sm"
                className={cn(
                  "w-full justify-start",
                  isActive && "bg-accent text-accent-foreground font-medium"
                )}
              >
                <Icon className="w-4 h-4 mr-2" />
                {item.label}
              </Button>
            </Link>
          );
        })}
      </div>

      <Separator className="mx-2" />

      {/* Recents Section */}
      <div className="flex flex-col flex-1 min-h-0">
        <div className="flex items-center justify-between px-3 pt-3 pb-1">
          <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
            Recents
          </span>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => refetch()}
            disabled={isFetching}
            className="h-5 w-5 p-0 text-muted-foreground/60 hover:text-muted-foreground"
            title={dataUpdatedAt ? `Last updated ${formatDistanceToNow(new Date(dataUpdatedAt), { addSuffix: true })}` : "Refresh"}
          >
            <RefreshCw className="h-3 w-3" />
          </Button>
        </div>

        {/* Filter chips */}
        <div className="flex items-center gap-1 px-3 pb-1 flex-wrap" data-testid="sidebar-filters">
          <Button
            variant={mineOnly ? "default" : "secondary"}
            size="sm"
            onClick={toggleMineOnly}
            className={cn(
              "h-auto rounded-full px-2 py-0.5 text-[0.6875rem] font-medium gap-1",
              !mineOnly && "text-muted-foreground",
            )}
            title="Show only my sessions"
            data-testid="filter-mine"
          >
            <User className="h-3 w-3" />
            Mine
          </Button>
          {(["running", "completed", "failed"] as const).map((filter) => (
            <Button
              key={filter}
              variant={statusFilter === filter ? "default" : "secondary"}
              size="sm"
              onClick={() => toggleStatusFilter(filter)}
              className={cn(
                "h-auto rounded-full px-2 py-0.5 text-[0.6875rem] font-medium capitalize",
                statusFilter !== filter && "text-muted-foreground",
              )}
              data-testid={`filter-${filter}`}
            >
              {filter}
            </Button>
          ))}
        </div>

        <div className="flex-1 overflow-y-auto">
          {isLoading ? (
            <div className="space-y-2 p-2">
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-10 w-full rounded-md" />
              ))}
            </div>
          ) : filteredSessions.length === 0 ? (
            <div className="p-4 text-center text-sm text-muted-foreground">
              {isFiltered ? "No matching sessions" : "No sessions yet"}
            </div>
          ) : (
              <div className="space-y-0.5 p-1">
                {visibleSessions.map((session: AgenticSession) => {
                  const name =
                    session.spec.displayName || session.metadata.name;
                  const phase = session.status?.phase || "Pending";
                  const isActive =
                    session.metadata.name === currentSessionName;
                  const activityTime = session.status?.lastActivityTime || session.metadata.creationTimestamp;

                  const borderColor =
                    phase === "Running"
                      ? "border-l-blue-500"
                      : phase === "Failed"
                        ? "border-l-red-500"
                        : phase === "Pending" || phase === "Creating" || phase === "Stopping"
                          ? "border-l-orange-400"
                          : "border-l-transparent";

                  const agentStatus = session.status?.agentStatus ?? deriveAgentStatusFromPhase(phase);

                  return (
                    <div
                      key={session.metadata.uid}
                      className={cn(
                        "group relative w-full flex items-center gap-2 rounded-md text-left text-sm transition-colors",
                        "border-l-2",
                        borderColor,
                        "hover:bg-accent hover:text-accent-foreground",
                        isActive &&
                          "bg-accent text-accent-foreground font-medium"
                      )}
                    >
                      <HoverCard openDelay={300} closeDelay={100}>
                        <HoverCardTrigger asChild>
                          <Link
                            href={sessionHref(session.metadata.name)}
                            onClick={() => onSessionSelect?.()}
                            className="flex items-center gap-2 flex-1 min-w-0 px-2 py-2"
                          >
                            <AgentStatusIndicator
                              status={agentStatus}
                              compact
                              className="flex-shrink-0"
                            />
                            <span className="flex-1 truncate">{name}</span>
                            <span className="text-xs text-muted-foreground flex-shrink-0 group-hover:hidden">
                              {activityTime
                                ? formatDistanceToNow(
                                    new Date(activityTime),
                                    { addSuffix: false }
                                  )
                                : ""}
                            </span>
                          </Link>
                        </HoverCardTrigger>
                        <HoverCardContent side="right" align="start" className="w-80">
                          <div className="space-y-2">
                            <p className="text-sm font-semibold truncate">
                              {name}
                            </p>
                            {session.spec.displayName && (
                              <p className="text-xs text-muted-foreground">{session.metadata.name}</p>
                            )}
                            <div className="flex flex-col gap-1.5 pt-1">
                              <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                                <SessionStatusDot phase={phase} />
                                <span>Session: {sessionPhaseLabel(phase)}</span>
                              </div>
                              <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                                <AgentStatusIndicator
                                  status={agentStatus}
                                  compact
                                />
                                <span>Agent: {agentStatusLabel(agentStatus)}</span>
                              </div>
                              <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                                <Cpu className="h-3 w-3" />
                                <span>{session.spec.llmSettings.model}</span>
                              </div>
                              {activityTime && (
                                <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                                  <Clock className="h-3 w-3" />
                                  <span>{formatDistanceToNow(new Date(activityTime), { addSuffix: true })}</span>
                                </div>
                              )}
                              {session.spec.initialPrompt && (
                                <div className="flex items-start gap-1.5 text-xs text-muted-foreground pt-1">
                                  <MessageSquare className="h-3 w-3 mt-0.5 shrink-0" />
                                  <span className="line-clamp-3">{session.spec.initialPrompt}</span>
                                </div>
                              )}
                            </div>
                          </div>
                        </HoverCardContent>
                      </HoverCard>
                      <SidebarSessionActions
                        sessionName={session.metadata.name}
                        displayName={name}
                        phase={phase}
                        onStop={handleStop}
                        onContinue={handleContinue}
                        onDelete={handleDelete}
                        onEditName={handleEditName}
                        canDelete={canDelete}
                        canModify={canModify}
                      />
                    </div>
                  );
                })}

                {hasMore && (
                  <button
                    type="button"
                    onClick={() => setShowAll(true)}
                    className="w-full flex items-center gap-2 px-2 py-2 rounded-md text-sm text-muted-foreground hover:bg-accent hover:text-accent-foreground transition-colors"
                  >
                    <MoreHorizontal className="w-4 h-4 flex-shrink-0" />
                    <span>Show more</span>
                  </button>
                )}
              </div>
          )}
        </div>
      </div>
      <EditSessionNameDialog
        open={!!editingSession}
        onOpenChange={(open) => !open && setEditingSession(null)}
        currentName={editingSession?.displayName || ""}
        onSave={handleSaveEditName}
        isLoading={updateDisplayNameMutation.isPending}
      />
      <DestructiveConfirmationDialog
        open={!!deletingSessionName}
        onOpenChange={(open) => !open && setDeletingSessionName(null)}
        onConfirm={confirmDelete}
        title="Delete session"
        description={`Delete session "${deletingSessionName}"? This action cannot be undone.`}
        confirmText="Delete"
        loading={deleteMutation.isPending}
      />
    </div>
  );
}

type SidebarSessionActionsProps = {
  sessionName: string;
  displayName: string;
  phase: string;
  onStop: (sessionName: string) => void;
  onContinue: (sessionName: string) => void;
  onDelete: (sessionName: string) => void;
  onEditName: (sessionName: string, currentDisplayName: string) => void;
  canDelete: boolean;
  canModify: boolean;
};

function SidebarSessionActions({
  sessionName,
  displayName,
  phase,
  onStop,
  onContinue,
  onDelete,
  onEditName,
  canDelete,
  canModify,
}: SidebarSessionActionsProps) {
  type RowAction = {
    key: string;
    label: string;
    onClick: () => void;
    icon: React.ReactNode;
    className?: string;
  };

  const actions: RowAction[] = [];

  if (canModify) {
    actions.push({
      key: "edit",
      label: "Rename",
      onClick: () => onEditName(sessionName, displayName),
      icon: <Pencil className="h-4 w-4" />,
    });
  }

  if (canModify && (phase === "Pending" || phase === "Creating" || phase === "Running")) {
    actions.push({
      key: "stop",
      label: "Stop",
      onClick: () => onStop(sessionName),
      icon: <Square className="h-4 w-4" />,
      className: "text-orange-600",
    });
  }

  if (canModify && (phase === "Completed" || phase === "Failed" || phase === "Stopped" || phase === "Error")) {
    actions.push({
      key: "continue",
      label: "Continue",
      onClick: () => onContinue(sessionName),
      icon: <ArrowRight className="h-4 w-4" />,
      className: "text-green-600",
    });
  }

  if (canDelete && phase !== "Creating") {
    actions.push({
      key: "delete",
      label: "Delete",
      onClick: () => onDelete(sessionName),
      icon: <Trash2 className="h-4 w-4" />,
      className: "text-red-600",
    });
  }

  if (actions.length === 0) return null;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          className="opacity-0 group-hover:opacity-100 focus:opacity-100 focus-visible:opacity-100 flex items-center justify-center h-6 w-6 rounded-sm flex-shrink-0 mr-1 text-muted-foreground hover:text-foreground hover:bg-accent-foreground/10 transition-colors"
        >
          <MoreVertical className="h-3.5 w-3.5" />
          <span className="sr-only">Session actions</span>
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" side="right">
        {actions.map((action) => (
          <DropdownMenuItem key={action.key} onClick={action.onClick} className={action.className}>
            <span className="mr-2">{action.icon}</span>
            {action.label}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
