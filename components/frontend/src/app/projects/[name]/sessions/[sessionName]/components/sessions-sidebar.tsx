"use client";

import { useMemo, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { AgentStatusIndicator, agentStatusLabel } from "@/components/agent-status-indicator";
import { deriveAgentStatusFromPhase } from "@/hooks/use-agent-status";
import { SessionStatusDot, sessionPhaseLabel } from "@/components/session-status-dot";
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
  Cpu,
  Clock,
  MessageSquare,
  RefreshCw,
} from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import { useSessionsPaginated } from "@/services/queries/use-sessions";
import { useVersion } from "@/services/queries/use-version";
import { cn } from "@/lib/utils";
import type { AgenticSession } from "@/types/api";

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

  const sessions = useMemo(() => {
    return data?.items ?? [];
  }, [data?.items]);

  const visibleSessions = useMemo(() => {
    if (showAll) return sessions;
    return sessions.slice(0, INITIAL_RECENTS_COUNT);
  }, [sessions, showAll]);

  const hasMore = sessions.length > INITIAL_RECENTS_COUNT && !showAll;

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
        label: "Sharing",
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

  const handleNavigate = (sessionName: string) => {
    onSessionSelect?.();
    router.push(`/projects/${projectName}/sessions/${sessionName}`);
  };

  const handleNewSession = () => {
    if (onNewSession) {
      onNewSession();
    } else {
      router.push(`/projects/${projectName}/new`);
    }
  };

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
            className="w-full justify-start text-muted-foreground hover:text-foreground"
          >
            <ChevronLeft className="w-4 h-4 mr-2" />
            Workspaces
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
          <button
            type="button"
            onClick={() => refetch()}
            disabled={isFetching}
            className="text-muted-foreground/60 hover:text-muted-foreground transition-colors disabled:opacity-50"
            title={dataUpdatedAt ? `Last updated ${formatDistanceToNow(new Date(dataUpdatedAt), { addSuffix: true })}` : "Refresh"}
          >
            <RefreshCw className="h-3 w-3" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto">
          {isLoading ? (
            <div className="space-y-2 p-2">
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-10 w-full rounded-md" />
              ))}
            </div>
          ) : sessions.length === 0 ? (
            <div className="p-4 text-center text-sm text-muted-foreground">
              No sessions yet
            </div>
          ) : (
              <div className="space-y-0.5 p-1">
                {visibleSessions.map((session: AgenticSession) => {
                  const name =
                    session.spec.displayName || session.metadata.name;
                  const phase = session.status?.phase || "Pending";
                  const isActive =
                    session.metadata.name === currentSessionName;
                  const createdAt = session.metadata.creationTimestamp;

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
                    <HoverCard key={session.metadata.uid} openDelay={300} closeDelay={100}>
                      <HoverCardTrigger asChild>
                        {/* div used instead of button to avoid nesting with SessionStatusDot's tooltip button */}
                        <div
                          role="button"
                          tabIndex={0}
                          onClick={() =>
                            handleNavigate(session.metadata.name)
                          }
                          onKeyDown={(e) => {
                            if (e.key === "Enter" || e.key === " ") {
                              e.preventDefault();
                              handleNavigate(session.metadata.name);
                            }
                          }}
                          className={cn(
                            "w-full flex items-center gap-2 px-2 py-2 rounded-md text-left text-sm transition-colors cursor-pointer",
                            "border-l-2",
                            borderColor,
                            "hover:bg-accent hover:text-accent-foreground",
                            isActive &&
                              "bg-accent text-accent-foreground font-medium"
                          )}
                        >
                          <AgentStatusIndicator
                            status={agentStatus}
                            compact
                            className="flex-shrink-0"
                          />
                          <span className="flex-1 truncate">{name}</span>
                          <span className="text-xs text-muted-foreground flex-shrink-0">
                            {createdAt
                              ? formatDistanceToNow(
                                  new Date(createdAt),
                                  { addSuffix: false }
                                )
                              : ""}
                          </span>
                        </div>
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
                            {createdAt && (
                              <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                                <Clock className="h-3 w-3" />
                                <span>{formatDistanceToNow(new Date(createdAt), { addSuffix: true })}</span>
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
    </div>
  );
}
