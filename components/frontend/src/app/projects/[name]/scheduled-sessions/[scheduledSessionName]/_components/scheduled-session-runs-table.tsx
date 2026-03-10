import { formatDistanceToNow } from "date-fns";
import Link from "next/link";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { SessionPhaseBadge } from "@/components/status-badge";
import type { AgenticSession } from "@/types/api";

type ScheduledSessionRunsTableProps = {
  runs: AgenticSession[] | undefined;
  projectName: string;
};

export function ScheduledSessionRunsTable({
  runs,
  projectName,
}: ScheduledSessionRunsTableProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Runs</CardTitle>
      </CardHeader>
      <CardContent>
        {!runs || runs.length === 0 ? (
          <p className="text-sm text-muted-foreground py-4 text-center">
            No runs yet
          </p>
        ) : (
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Created</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {runs.map((run) => (
                  <TableRow key={run.metadata.uid || run.metadata.name}>
                    <TableCell>
                      <Link
                        href={`/projects/${encodeURIComponent(projectName)}/sessions/${run.metadata.name}`}
                        className="text-link hover:underline hover:text-link-hover transition-colors font-medium"
                      >
                        {run.spec.displayName || run.metadata.name}
                      </Link>
                    </TableCell>
                    <TableCell>
                      <SessionPhaseBadge
                        phase={run.status?.phase || "Pending"}
                        stoppedReason={run.status?.stoppedReason}
                      />
                    </TableCell>
                    <TableCell>
                      {run.metadata.creationTimestamp &&
                        formatDistanceToNow(new Date(run.metadata.creationTimestamp), {
                          addSuffix: true,
                        })}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
