"use client";

import { useRouter } from "next/navigation";
import Link from "next/link";
import { cn } from "@/lib/utils";
import { useSessionsPaginated } from "@/services/queries/use-sessions";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Plus, ChevronRight } from "lucide-react";
import { getPhaseColor } from "@/utils/session-helpers";

type SessionTabsProps = {
  projectName: string;
  currentSessionName: string;
  onCreateSession?: () => void;
};

export function SessionTabs({
  projectName,
  currentSessionName,
  onCreateSession,
}: SessionTabsProps) {
  const router = useRouter();
  const { data: sessionsData, isLoading } = useSessionsPaginated(projectName, {
    limit: 10,
    sortBy: "metadata.creationTimestamp",
    sortOrder: "desc",
  });

  const sessions = sessionsData?.sessions || [];

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 h-10 px-1">
        <div className="h-8 w-24 bg-muted animate-pulse rounded-md" />
        <div className="h-8 w-24 bg-muted animate-pulse rounded-md" />
        <div className="h-8 w-24 bg-muted animate-pulse rounded-md" />
      </div>
    );
  }

  if (sessions.length === 0) {
    return null;
  }

  return (
    <div className="flex items-center gap-2">
      <div className="overflow-x-auto max-w-[calc(100vw-200px)] md:max-w-[600px] lg:max-w-[800px] scrollbar-hide">
        <div className="flex items-center gap-1 p-1">
          {sessions.map((session) => {
            const isActive = session.metadata.name === currentSessionName;
            const displayName =
              session.spec.displayName || session.metadata.name;
            const phase = session.status?.phase || "Pending";

            return (
              <Link
                key={session.metadata.name}
                href={`/projects/${projectName}/sessions/${session.metadata.name}`}
                className={cn(
                  "inline-flex items-center gap-2 px-3 py-1.5 rounded-md text-sm font-medium transition-colors whitespace-nowrap",
                  "hover:bg-accent hover:text-accent-foreground",
                  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                  isActive
                    ? "bg-background text-foreground shadow-sm border"
                    : "text-muted-foreground"
                )}
              >
                <span className="truncate max-w-[120px]">{displayName}</span>
                <Badge
                  variant="outline"
                  className={cn("text-[10px] px-1.5 py-0", getPhaseColor(phase))}
                >
                  {phase === "Running" ? "●" : phase.slice(0, 4)}
                </Badge>
              </Link>
            );
          })}
        </div>
      </div>

      {/* Quick actions */}
      <div className="flex items-center gap-1 border-l pl-2 ml-1">
        {onCreateSession && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onCreateSession}
            className="h-8 w-8 p-0"
            title="New session"
          >
            <Plus className="h-4 w-4" />
          </Button>
        )}
        <Button
          variant="ghost"
          size="sm"
          onClick={() => router.push(`/projects/${projectName}?section=sessions`)}
          className="h-8 px-2 text-xs text-muted-foreground"
          title="View all sessions"
        >
          All
          <ChevronRight className="h-3 w-3 ml-1" />
        </Button>
      </div>
    </div>
  );
}
