"use client";

import { useParams } from "next/navigation";
import {
  BookOpen,
  GitPullRequestArrow,
  Brain,
  MessageSquareWarning,
  Sparkles,
} from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  useLearningSummary,
  useLearningTimeline,
} from "@/services/queries/use-learning";
import { useWorkspaceFlag } from "@/services/queries/use-feature-flags-admin";
import type { TimelineEntry } from "@/types/learning";
import { CORRECTION_TYPE_LABELS } from "@/services/api/corrections";

/** Colors for correction type breakdown bars. */
const CORRECTION_TYPE_COLORS: Record<string, string> = {
  incomplete: "bg-amber-500",
  incorrect: "bg-red-500",
  out_of_scope: "bg-blue-500",
  style: "bg-purple-500",
};

function SummaryCard({
  title,
  value,
  icon: Icon,
  description,
  isLoading,
}: {
  title: string;
  value: number;
  icon: React.ComponentType<{ className?: string }>;
  description: string;
  isLoading: boolean;
}) {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <Icon className="w-4 h-4 text-muted-foreground" />
          <CardTitle className="text-sm">{title}</CardTitle>
        </div>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-8 w-16" />
        ) : (
          <div className="text-3xl font-bold">{value}</div>
        )}
        <CardDescription className="mt-1">{description}</CardDescription>
      </CardContent>
    </Card>
  );
}

function CorrectionBreakdown({
  correctionsByType,
  total,
  isLoading,
}: {
  correctionsByType: Record<string, number>;
  total: number;
  isLoading: boolean;
}) {
  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Corrections by Type</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-6 w-full" />
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }

  const types = Object.entries(correctionsByType);

  if (types.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Corrections by Type</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            No correction data yet
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Corrections by Type</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {types.map(([type, count]) => {
            const pct = total > 0 ? (count / total) * 100 : 0;
            return (
              <div key={type}>
                <div className="flex items-center justify-between text-sm mb-1">
                  <span>
                    {CORRECTION_TYPE_LABELS[type as keyof typeof CORRECTION_TYPE_LABELS] || type}
                  </span>
                  <span className="text-muted-foreground">
                    {count} ({Math.round(pct)}%)
                  </span>
                </div>
                <div className="h-2 rounded-full bg-muted overflow-hidden">
                  <div
                    className={`h-full rounded-full ${CORRECTION_TYPE_COLORS[type] || "bg-primary"}`}
                    style={{ width: `${pct}%` }}
                  />
                </div>
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
}

function TimelineItem({ entry }: { entry: TimelineEntry }) {
  return (
    <div className="flex gap-3 py-3 border-b last:border-b-0">
      <div className="flex-shrink-0 mt-0.5">
        <div className="w-2 h-2 rounded-full bg-primary" />
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-sm">{entry.summary}</p>
        <div className="flex items-center gap-2 mt-1 text-xs text-muted-foreground">
          <span>{new Date(entry.timestamp).toLocaleDateString()}</span>
          {entry.correctionType && (
            <span className="px-1.5 py-0.5 rounded bg-muted">
              {CORRECTION_TYPE_LABELS[entry.correctionType as keyof typeof CORRECTION_TYPE_LABELS] ||
                entry.correctionType}
            </span>
          )}
          {entry.memoryId && (
            <span className="text-primary">{entry.memoryId}</span>
          )}
        </div>
      </div>
    </div>
  );
}

function EmptyState() {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <Sparkles className="w-12 h-12 text-muted-foreground/30 mb-4" />
      <h3 className="text-lg font-medium mb-2">No learning data yet</h3>
      <p className="text-sm text-muted-foreground max-w-md">
        As your team works with agents, corrections and feedback are captured
        automatically. The learning pipeline transforms these into project
        memories that improve future sessions.
      </p>
      <p className="text-sm text-muted-foreground mt-4 max-w-md">
        Start by running sessions and providing corrections when the agent makes
        mistakes. Each correction feeds the improvement loop.
      </p>
    </div>
  );
}

export default function LearningPage() {
  const params = useParams();
  const projectName = params?.name as string;

  const { enabled: flagEnabled, isLoading: flagLoading } = useWorkspaceFlag(
    projectName,
    "learning-agent-loop"
  );

  const { data: summary, isLoading: summaryLoading } =
    useLearningSummary(projectName);

  const { data: timeline, isLoading: timelineLoading } =
    useLearningTimeline(projectName);

  // Hide the page entirely when the flag is off (not just loading)
  if (!flagLoading && !flagEnabled) {
    return (
      <div className="p-6">
        <p className="text-sm text-muted-foreground">
          The learning agent loop feature is not enabled for this workspace.
        </p>
      </div>
    );
  }

  const isLoading = summaryLoading || timelineLoading;
  const isEmpty =
    !isLoading &&
    summary?.totalCorrections === 0 &&
    (!timeline?.items || timeline.items.length === 0);

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div>
        <h1 className="text-xl font-semibold">Learning</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Track how corrections improve your agents over time.
        </p>
      </div>

      {isEmpty ? (
        <EmptyState />
      ) : (
        <>
          {/* Summary cards */}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            <SummaryCard
              title="Corrections"
              value={summary?.totalCorrections ?? 0}
              icon={MessageSquareWarning}
              description="Total corrections submitted"
              isLoading={summaryLoading}
            />
            <SummaryCard
              title="Improvement Sessions"
              value={summary?.improvementSessions ?? 0}
              icon={GitPullRequestArrow}
              description="Automated improvement runs"
              isLoading={summaryLoading}
            />
            <SummaryCard
              title="Memories Created"
              value={summary?.memoriesCreated ?? 0}
              icon={Brain}
              description="Lessons extracted"
              isLoading={summaryLoading}
            />
            <SummaryCard
              title="Citations"
              value={summary?.memoryCitations ?? 0}
              icon={BookOpen}
              description="Times memories were cited"
              isLoading={summaryLoading}
            />
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Correction breakdown */}
            <CorrectionBreakdown
              correctionsByType={summary?.correctionsByType ?? {}}
              total={summary?.totalCorrections ?? 0}
              isLoading={summaryLoading}
            />

            {/* Timeline */}
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Recent Activity</CardTitle>
              </CardHeader>
              <CardContent>
                {timelineLoading ? (
                  <div className="space-y-3">
                    {Array.from({ length: 5 }).map((_, i) => (
                      <Skeleton key={i} className="h-12 w-full" />
                    ))}
                  </div>
                ) : timeline?.items && timeline.items.length > 0 ? (
                  <div>
                    {timeline.items.map((entry) => (
                      <TimelineItem key={entry.id} entry={entry} />
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">
                    No recent activity
                  </p>
                )}
              </CardContent>
            </Card>
          </div>
        </>
      )}
    </div>
  );
}
