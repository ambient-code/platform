"use client";

import { Pencil } from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import { Button } from "@/components/ui/button";
import { SessionPhaseBadge } from "@/components/status-badge";
import type { AgenticSession } from "@/types/agentic-session";

type SessionDetailsProps = {
  session: AgenticSession;
  onEditName?: () => void;
};

export function SessionDetails({ session, onEditName }: SessionDetailsProps) {
  const phase = session.status?.phase || "Pending";
  const stoppedReason = session.status?.stoppedReason;
  const displayName = session.spec.displayName || session.metadata.name;
  const model = session.spec.llmSettings?.model || "—";
  const createdAt = session.metadata.creationTimestamp;

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
          <span className="text-sm text-muted-foreground truncate max-w-[200px]">
            {model}
          </span>
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
