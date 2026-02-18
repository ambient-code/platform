"use client";

import { useState } from "react";
import { Flag, RefreshCw, Loader2, Info, AlertTriangle, RotateCcw } from "lucide-react";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
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
import { Switch } from "@/components/ui/switch";
import { EmptyState } from "@/components/empty-state";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

import {
  useFeatureFlags,
  useToggleFeatureFlag,
  useRemoveFeatureFlagOverride,
} from "@/services/queries/use-feature-flags-admin";
import { successToast, errorToast } from "@/hooks/use-toast";

type FeatureFlagsSectionProps = {
  projectName: string;
};

export function FeatureFlagsSection({ projectName }: FeatureFlagsSectionProps) {
  const {
    data: flags = [],
    isLoading,
    isError,
    error,
    refetch,
  } = useFeatureFlags(projectName);
  const toggleMutation = useToggleFeatureFlag();
  const removeOverrideMutation = useRemoveFeatureFlagOverride();

  const [pendingFlags, setPendingFlags] = useState<Set<string>>(new Set());

  const handleToggle = (flagName: string, currentEnabled: boolean) => {
    const newEnabled = !currentEnabled;
    setPendingFlags((prev) => new Set(prev).add(flagName));

    toggleMutation.mutate(
      { projectName, flagName, enable: newEnabled },
      {
        onSuccess: () => {
          successToast(
            `Feature "${flagName}" ${newEnabled ? "enabled" : "disabled"} for this workspace`
          );
          setPendingFlags((prev) => {
            const next = new Set(prev);
            next.delete(flagName);
            return next;
          });
        },
        onError: (err) => {
          errorToast(
            err instanceof Error ? err.message : "Failed to update feature flag"
          );
          setPendingFlags((prev) => {
            const next = new Set(prev);
            next.delete(flagName);
            return next;
          });
        },
      }
    );
  };

  const handleResetToDefault = (flagName: string) => {
    setPendingFlags((prev) => new Set(prev).add(flagName));

    removeOverrideMutation.mutate(
      { projectName, flagName },
      {
        onSuccess: () => {
          successToast(`Feature "${flagName}" reset to platform default`);
          setPendingFlags((prev) => {
            const next = new Set(prev);
            next.delete(flagName);
            return next;
          });
        },
        onError: (err) => {
          errorToast(
            err instanceof Error ? err.message : "Failed to reset feature flag"
          );
          setPendingFlags((prev) => {
            const next = new Set(prev);
            next.delete(flagName);
            return next;
          });
        },
      }
    );
  };

  const getTypeBadge = (type?: string) => {
    switch (type) {
      case "experiment":
        return <Badge variant="secondary">Experiment</Badge>;
      case "operational":
        return <Badge variant="outline">Operational</Badge>;
      case "kill-switch":
        return <Badge variant="destructive">Kill Switch</Badge>;
      case "permission":
        return <Badge>Permission</Badge>;
      default:
        return <Badge variant="outline">Release</Badge>;
    }
  };

  const getSourceBadge = (source?: string, hasOverride?: boolean) => {
    if (source === "workspace-override" || hasOverride) {
      return (
        <Badge variant="default" className="text-xs">
          Workspace Override
        </Badge>
      );
    }
    return (
      <Badge variant="secondary" className="text-xs">
        Platform Default
      </Badge>
    );
  };

  // Check if Unleash is not configured (service unavailable error)
  const isNotConfigured =
    isError &&
    error instanceof Error &&
    error.message.includes("not configured");

  return (
    <Card className="flex-1">
      <CardHeader>
        <div className="flex items-start justify-between">
          <div>
            <CardTitle>Feature Flags</CardTitle>
            <CardDescription>
              Manage feature toggles for this workspace. Changes only affect this workspace.
            </CardDescription>
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              onClick={() => refetch()}
              disabled={isLoading}
            >
              <RefreshCw
                className={`w-4 h-4 mr-2 ${isLoading ? "animate-spin" : ""}`}
              />
              Refresh
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {isNotConfigured ? (
          <Alert variant="warning">
            <AlertTriangle className="h-4 w-4" />
            <AlertTitle>Unleash Not Configured</AlertTitle>
            <AlertDescription>
              Feature flag management requires Unleash Admin API configuration.
              Set the <code>UNLEASH_ADMIN_URL</code> and{" "}
              <code>UNLEASH_ADMIN_TOKEN</code> environment variables on the
              backend to enable this feature.
            </AlertDescription>
          </Alert>
        ) : (
          <>
            <Alert>
              <Info className="h-4 w-4" />
              <AlertTitle>Workspace-Scoped Feature Flags</AlertTitle>
              <AlertDescription>
                Toggle switches set workspace-specific overrides. Use the reset button
                to revert to the platform default (controlled by Unleash).
              </AlertDescription>
            </Alert>

            {isLoading ? (
              <div className="flex items-center justify-center py-8">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              </div>
            ) : isError ? (
              <Alert variant="destructive">
                <AlertTriangle className="h-4 w-4" />
                <AlertTitle>Error Loading Feature Flags</AlertTitle>
                <AlertDescription>
                  {error instanceof Error
                    ? error.message
                    : "Failed to load feature flags"}
                </AlertDescription>
              </Alert>
            ) : flags.length === 0 ? (
              <EmptyState
                icon={Flag}
                title="No feature flags found"
                description="No feature toggles are configured in Unleash for this project"
              />
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-[80px]">Enabled</TableHead>
                    <TableHead>Feature</TableHead>
                    <TableHead className="hidden md:table-cell">Source</TableHead>
                    <TableHead className="hidden lg:table-cell">Type</TableHead>
                    <TableHead className="hidden xl:table-cell">
                      Description
                    </TableHead>
                    <TableHead className="w-[80px]">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {flags.map((flag) => {
                    const isPending = pendingFlags.has(flag.name);
                    const hasOverride = flag.overrideEnabled !== undefined && flag.overrideEnabled !== null;

                    return (
                      <TableRow key={flag.name}>
                        <TableCell>
                          {isPending ? (
                            <Loader2 className="h-4 w-4 animate-spin" />
                          ) : (
                            <Switch
                              checked={flag.enabled}
                              onCheckedChange={() =>
                                handleToggle(flag.name, flag.enabled)
                              }
                              aria-label={`Toggle ${flag.name}`}
                            />
                          )}
                        </TableCell>
                        <TableCell>
                          <div className="font-medium font-mono text-sm">
                            {flag.name}
                          </div>
                          {flag.stale && (
                            <Badge variant="outline" className="mt-1 text-xs">
                              Stale
                            </Badge>
                          )}
                        </TableCell>
                        <TableCell className="hidden md:table-cell">
                          {getSourceBadge(flag.source, hasOverride)}
                        </TableCell>
                        <TableCell className="hidden lg:table-cell">
                          {getTypeBadge(flag.type)}
                        </TableCell>
                        <TableCell className="hidden xl:table-cell text-sm text-muted-foreground">
                          {flag.description || "—"}
                        </TableCell>
                        <TableCell>
                          {hasOverride && (
                            <TooltipProvider>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <Button
                                    variant="ghost"
                                    size="icon"
                                    onClick={() => handleResetToDefault(flag.name)}
                                    disabled={isPending}
                                    aria-label={`Reset ${flag.name} to default`}
                                  >
                                    <RotateCcw className="h-4 w-4" />
                                  </Button>
                                </TooltipTrigger>
                                <TooltipContent>
                                  Reset to platform default
                                </TooltipContent>
                              </Tooltip>
                            </TooltipProvider>
                          )}
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            )}
          </>
        )}
      </CardContent>
    </Card>
  );
}
