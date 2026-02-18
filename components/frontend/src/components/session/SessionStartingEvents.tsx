"use client";

import { useState } from "react";
import { Loader2, CheckCircle2, AlertTriangle, Info, Clock, ChevronDown, ChevronUp } from "lucide-react";
import { useSessionPodEvents } from "@/services/queries/use-sessions";
import type { PodEvent } from "@/services/api/sessions";
import { cn } from "@/lib/utils";

type SessionStartingEventsProps = {
  projectName: string;
  sessionName: string;
};

function EventIcon({ type, reason }: { type: string; reason: string }) {
  if (type === "Warning") {
    return <AlertTriangle className="h-3.5 w-3.5 text-amber-500 shrink-0" />;
  }
  if (["Pulled", "Created", "Started", "Scheduled"].includes(reason)) {
    return <CheckCircle2 className="h-3.5 w-3.5 text-green-500 shrink-0" />;
  }
  if (reason === "Pulling") {
    return <Loader2 className="h-3.5 w-3.5 text-blue-500 animate-spin shrink-0" />;
  }
  return <Info className="h-3.5 w-3.5 text-muted-foreground shrink-0" />;
}

function formatTimestamp(ts: string): string {
  try {
    const date = new Date(ts);
    return date.toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  } catch {
    return ts;
  }
}

function EventRow({ event }: { event: PodEvent }) {
  return (
    <div
      className={cn(
        "flex items-start gap-2 text-xs",
        event.type === "Warning"
          ? "text-amber-600 dark:text-amber-400"
          : "text-muted-foreground",
      )}
    >
      <EventIcon type={event.type} reason={event.reason} />
      <span className="flex-1 break-words">{event.message}</span>
      <span className="text-[10px] tabular-nums whitespace-nowrap opacity-60 flex items-center gap-0.5">
        <Clock className="h-2.5 w-2.5" />
        {formatTimestamp(event.timestamp)}
      </span>
    </div>
  );
}

export function SessionStartingEvents({
  projectName,
  sessionName,
}: SessionStartingEventsProps) {
  const [expanded, setExpanded] = useState(false);

  const { data } = useSessionPodEvents(
    projectName,
    sessionName,
    2000, // Poll every 2s during startup
  );

  const events: PodEvent[] = data?.events ?? [];
  const latestEvent = events.length > 0 ? events[events.length - 1] : null;
  const olderEvents = events.length > 1 ? events.slice(0, -1) : [];

  return (
    <div className="flex flex-col items-center justify-center h-full px-4">
      <div className="w-full max-w-md">
        {/* Header */}
        <div className="flex flex-col items-center mb-6">
          <Loader2 className="h-10 w-10 animate-spin text-blue-600 mb-3" />
          <h3 className="font-semibold text-lg">Starting Session</h3>
          <p className="text-sm text-muted-foreground mt-1">
            Setting up your workspace...
          </p>
        </div>

        {/* Latest event */}
        {latestEvent && (
          <div className="border rounded-lg bg-muted/30 overflow-hidden">
            <div className="p-3">
              <EventRow event={latestEvent} />
            </div>

            {/* Expand toggle for older events */}
            {olderEvents.length > 0 && (
              <>
                <button
                  type="button"
                  onClick={() => setExpanded((v) => !v)}
                  className="w-full flex items-center justify-center gap-1 px-3 py-1.5 text-[10px] text-muted-foreground hover:text-foreground hover:bg-muted/50 border-t transition-colors"
                >
                  {expanded ? (
                    <>
                      <ChevronUp className="h-3 w-3" />
                      Hide earlier events
                    </>
                  ) : (
                    <>
                      <ChevronDown className="h-3 w-3" />
                      {olderEvents.length} earlier event{olderEvents.length !== 1 ? "s" : ""}
                    </>
                  )}
                </button>

                {expanded && (
                  <div className="border-t px-3 py-2 max-h-48 overflow-y-auto space-y-2">
                    {olderEvents.map((event, idx) => (
                      <EventRow key={`${event.reason}-${idx}`} event={event} />
                    ))}
                  </div>
                )}
              </>
            )}
          </div>
        )}

        {/* No events yet */}
        {!latestEvent && (
          <div className="text-center text-xs text-muted-foreground">
            <p>Waiting for pod events...</p>
          </div>
        )}
      </div>
    </div>
  );
}
