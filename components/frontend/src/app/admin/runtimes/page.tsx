"use client";

import { useState } from "react";
import { RefreshCw, AlertTriangle, Server, ChevronDown, ChevronRight } from "lucide-react";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Skeleton } from "@/components/ui/skeleton";
import { Breadcrumbs } from "@/components/breadcrumbs";
import { PageHeader } from "@/components/page-header";
import { EmptyState } from "@/components/empty-state";
import { useRunnerTypesGlobal } from "@/services/queries/use-runner-types";
import type { RunnerType } from "@/services/api/runner-types";

function RuntimeStatusBadge() {
  // All runners returned by the API are already enabled (gated runners
  // are filtered out server-side by the backend's feature flag check).
  return <Badge variant="secondary">Enabled</Badge>;
}

function RuntimeDetailPanel({ runtime }: { runtime: RunnerType }) {
  const configJson = JSON.stringify(runtime, null, 2);

  return (
    <div className="p-4 bg-muted/30 border-t">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
        <div>
          <h4 className="text-sm font-medium mb-2">Authentication</h4>
          <div className="space-y-1 text-sm">
            <div className="flex gap-2">
              <span className="text-muted-foreground">Required keys:</span>
              <span className="font-mono">
                {(runtime.auth?.requiredSecretKeys ?? []).join(", ") || "None"}
              </span>
            </div>
            <div className="flex gap-2">
              <span className="text-muted-foreground">Key logic:</span>
              <Badge variant="outline" className="text-xs">
                {runtime.auth?.secretKeyLogic ?? "any"}
              </Badge>
            </div>
            <div className="flex gap-2">
              <span className="text-muted-foreground">Vertex AI:</span>
              <span>{runtime.auth?.vertexSupported ? "Supported" : "Not supported"}</span>
            </div>
          </div>
        </div>
        <div>
          <h4 className="text-sm font-medium mb-2">Provider</h4>
          <Badge variant="outline" className="text-xs">
            {runtime.provider}
          </Badge>
        </div>
      </div>
      <div>
        <h4 className="text-sm font-medium mb-2">Full Configuration</h4>
        <pre className="text-xs bg-muted rounded-md p-3 overflow-x-auto max-h-64 overflow-y-auto font-mono">
          {configJson}
        </pre>
      </div>
    </div>
  );
}

function LoadingSkeleton() {
  return (
    <div className="space-y-3">
      {Array.from({ length: 3 }).map((_, i) => (
        <div key={i} className="flex items-center gap-4 p-4 border rounded-lg">
          <Skeleton className="h-5 w-[150px]" />
          <Skeleton className="h-5 w-[200px]" />
          <Skeleton className="h-5 w-[40px]" />
          <Skeleton className="h-5 w-[80px]" />
        </div>
      ))}
    </div>
  );
}

export default function AdminRuntimesPage() {
  const { data: runtimes, isLoading, isError, error, refetch } = useRunnerTypesGlobal();
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const toggleExpanded = (id: string) => {
    setExpandedId((prev) => (prev === id ? null : id));
  };

  return (
    <div className="min-h-screen bg-background">
      <div className="sticky top-0 z-20 bg-card border-b">
        <div className="px-6 py-4">
          <Breadcrumbs
            items={[
              { label: "Admin", href: "/admin/runtimes" },
              { label: "Runtimes" },
            ]}
          />
        </div>
      </div>

      <div className="container mx-auto px-6 py-6 space-y-6">
        <PageHeader
          title="Agent Runtimes"
          description="Registered runtimes from the agent registry ConfigMap. Runtimes with a feature gate can be toggled per workspace in workspace settings."
        />

        <Card>
          <CardHeader>
            <div className="flex items-start justify-between">
              <div>
                <CardTitle className="flex items-center gap-2">
                  <Server className="h-5 w-5" />
                  Runtime Registry
                </CardTitle>
                <CardDescription>
                  {runtimes ? `${runtimes.length} runtime${runtimes.length !== 1 ? "s" : ""} registered` : "Loading..."}
                </CardDescription>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => refetch()}
                disabled={isLoading}
              >
                <RefreshCw className={`w-4 h-4 mr-2 ${isLoading ? "animate-spin" : ""}`} />
                Refresh
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <LoadingSkeleton />
            ) : isError ? (
              <Alert variant="destructive">
                <AlertTriangle className="h-4 w-4" />
                <AlertTitle>Failed to Load Runtimes</AlertTitle>
                <AlertDescription className="flex items-center justify-between">
                  <span>
                    {error instanceof Error ? error.message : "An unknown error occurred"}
                  </span>
                  <Button variant="outline" size="sm" onClick={() => refetch()}>
                    Retry
                  </Button>
                </AlertDescription>
              </Alert>
            ) : !runtimes || runtimes.length === 0 ? (
              <EmptyState
                icon={Server}
                title="No runtimes registered"
                description="No agent runtimes are configured in the registry ConfigMap."
              />
            ) : (
              <div className="border rounded-lg overflow-hidden">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-8" />
                      <TableHead>Runtime</TableHead>
                      <TableHead className="hidden md:table-cell">Description</TableHead>
                      <TableHead className="w-[80px] text-center">Provider</TableHead>
                      <TableHead className="w-[200px]">Status</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {runtimes.map((runtime) => {
                      const isExpanded = expandedId === runtime.id;
                      return (
                        <RuntimeRow
                          key={runtime.id}
                          runtime={runtime}
                          isExpanded={isExpanded}
                          onToggle={() => toggleExpanded(runtime.id)}
                        />
                      );
                    })}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function RuntimeRow({
  runtime,
  isExpanded,
  onToggle,
}: {
  runtime: RunnerType;
  isExpanded: boolean;
  onToggle: () => void;
}) {
  return (
    <>
      <TableRow
        className="cursor-pointer hover:bg-muted/50 transition-colors"
        onClick={onToggle}
        onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onToggle(); } }}
        tabIndex={0}
        role="button"
        aria-expanded={isExpanded}
      >
        <TableCell className="w-8">
          {isExpanded ? (
            <ChevronDown className="h-4 w-4 text-muted-foreground" />
          ) : (
            <ChevronRight className="h-4 w-4 text-muted-foreground" />
          )}
        </TableCell>
        <TableCell>
          <div className="font-medium">{runtime.displayName}</div>
          <div className="text-xs text-muted-foreground font-mono">{runtime.id}</div>
        </TableCell>
        <TableCell className="hidden md:table-cell text-sm text-muted-foreground">
          {runtime.description || "\u2014"}
        </TableCell>
        <TableCell className="text-center">
          <Badge variant="outline">{runtime.provider}</Badge>
        </TableCell>
        <TableCell>
          <RuntimeStatusBadge />
        </TableCell>
      </TableRow>
      {isExpanded && (
        <TableRow>
          <TableCell colSpan={5} className="p-0">
            <RuntimeDetailPanel runtime={runtime} />
          </TableCell>
        </TableRow>
      )}
    </>
  );
}
