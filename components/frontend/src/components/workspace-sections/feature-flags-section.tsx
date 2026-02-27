"use client";

import { useState, useEffect, useMemo } from "react";
import { Flag, RefreshCw, Loader2, Info, AlertTriangle, Save, RotateCcw } from "lucide-react";

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
import { Switch } from "@/components/ui/switch";
import { Separator } from "@/components/ui/separator";
import { EmptyState } from "@/components/empty-state";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

import { useFeatureFlags } from "@/services/queries/use-feature-flags-admin";
import * as featureFlagsApi from "@/services/api/feature-flags-admin";
import { successToast, errorToast } from "@/hooks/use-toast";
import { useQueryClient } from "@tanstack/react-query";

type FeatureFlagsSectionProps = {
  projectName: string;
};

type LocalFlagState = {
  enabled: boolean;
  changed: boolean; // true if user toggled this flag
  markedForReset: boolean; // true if user wants to remove the workspace override
};

export function FeatureFlagsSection({ projectName }: FeatureFlagsSectionProps) {
  const queryClient = useQueryClient();
  const {
    data: flags = [],
    isLoading,
    isError,
    error,
    refetch,
  } = useFeatureFlags(projectName);

  // Local state to track pending changes
  const [localState, setLocalState] = useState<Record<string, LocalFlagState>>({});
  const [isSaving, setIsSaving] = useState(false);

  // Stable serialization of flags to detect actual data changes
  const flagsKey = useMemo(() => {
    return flags.map(f => `${f.name}:${f.enabled}:${f.overrideEnabled}`).join('|');
  }, [flags]);

  // Reset local state when flags data changes
  useEffect(() => {
    // Initialize local state from server state
    const initial: Record<string, LocalFlagState> = {};
    for (const flag of flags) {
      initial[flag.name] = {
        enabled: flag.enabled,
        changed: false,
        markedForReset: false,
      };
    }
    setLocalState(initial);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [flagsKey]);

  // Check if there are unsaved changes
  const hasChanges = useMemo(() => {
    return Object.values(localState).some((s) => s.changed || s.markedForReset);
  }, [localState]);

  // Get the count of changed flags
  const changedCount = useMemo(() => {
    return Object.values(localState).filter((s) => s.changed || s.markedForReset).length;
  }, [localState]);

  const handleToggle = (flagName: string) => {
    setLocalState((prev) => {
      const current = prev[flagName];
      if (!current) return prev;

      // If marked for reset, clear that first
      if (current.markedForReset) {
        return {
          ...prev,
          [flagName]: {
            ...current,
            markedForReset: false,
          },
        };
      }

      // Find the original server state
      const serverFlag = flags.find((f) => f.name === flagName);
      const serverEnabled = serverFlag?.enabled ?? false;

      const newEnabled = !current.enabled;
      // Mark as changed only if different from server state
      const isChanged = newEnabled !== serverEnabled;

      return {
        ...prev,
        [flagName]: {
          enabled: newEnabled,
          changed: isChanged,
          markedForReset: false,
        },
      };
    });
  };

  const handleMarkForReset = (flagName: string) => {
    setLocalState((prev) => {
      const current = prev[flagName];
      if (!current) return prev;

      return {
        ...prev,
        [flagName]: {
          ...current,
          markedForReset: true,
          changed: false, // Clear toggle change since we're resetting
        },
      };
    });
  };

  const handleUndoReset = (flagName: string) => {
    setLocalState((prev) => {
      const current = prev[flagName];
      if (!current) return prev;

      // Restore to server state
      const serverFlag = flags.find((f) => f.name === flagName);

      return {
        ...prev,
        [flagName]: {
          enabled: serverFlag?.enabled ?? false,
          changed: false,
          markedForReset: false,
        },
      };
    });
  };

  const handleSave = async () => {
    const changedFlags = Object.entries(localState).filter(([, s]) => s.changed);
    const resetFlags = Object.entries(localState).filter(([, s]) => s.markedForReset);

    if (changedFlags.length === 0 && resetFlags.length === 0) {
      return;
    }

    setIsSaving(true);

    try {
      const promises: Promise<unknown>[] = [];

      // Save all changed flags (set overrides)
      for (const [flagName, state] of changedFlags) {
        if (state.enabled) {
          promises.push(featureFlagsApi.enableFeatureFlag(projectName, flagName));
        } else {
          promises.push(featureFlagsApi.disableFeatureFlag(projectName, flagName));
        }
      }

      // Reset flags (remove overrides)
      for (const [flagName] of resetFlags) {
        promises.push(featureFlagsApi.removeFeatureFlagOverride(projectName, flagName));
      }

      await Promise.all(promises);

      const totalChanges = changedFlags.length + resetFlags.length;
      successToast(`${totalChanges} feature flag${totalChanges > 1 ? "s" : ""} updated`);

      // Invalidate queries to refetch fresh data
      queryClient.invalidateQueries({ queryKey: ["feature-flags", "list", projectName] });
    } catch (err) {
      errorToast(err instanceof Error ? err.message : "Failed to save feature flags");
    } finally {
      setIsSaving(false);
    }
  };

  const handleDiscard = () => {
    // Reset to server state
    const initial: Record<string, LocalFlagState> = {};
    for (const flag of flags) {
      initial[flag.name] = {
        enabled: flag.enabled,
        changed: false,
        markedForReset: false,
      };
    }
    setLocalState(initial);
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

  const getSourceBadge = (source?: string, hasOverride?: boolean, markedForReset?: boolean) => {
    if (markedForReset) {
      return (
        <Badge variant="outline" className="text-xs bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200">
          Will Reset
        </Badge>
      );
    }
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
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <Flag className="h-5 w-5" />
              Feature Flags
            </CardTitle>
            <CardDescription>
              Enable or disable features for this workspace. Changes are saved when you click Save.
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
      <Separator />
      <CardContent className="space-y-4 pt-4">
        {isNotConfigured ? (
          <Alert variant="warning">
            <AlertTriangle className="h-4 w-4" />
            <AlertTitle>Feature Flags Not Available</AlertTitle>
            <AlertDescription>
              Feature flag management requires Unleash to be configured.
              Contact your platform administrator to enable this feature.
            </AlertDescription>
          </Alert>
        ) : (
          <>
            <Alert>
              <Info className="h-4 w-4" />
              <AlertTitle>Workspace-Scoped Feature Flags</AlertTitle>
              <AlertDescription>
                Toggle switches to enable or disable features for this workspace only.
                Use the reset button to revert to the platform default.
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
                description="No feature toggles are configured for this project"
              />
            ) : (
              <>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-[80px]">Enabled</TableHead>
                      <TableHead>Feature</TableHead>
                      <TableHead className="hidden lg:table-cell">Description</TableHead>
                      <TableHead className="hidden md:table-cell">Source</TableHead>
                      <TableHead className="hidden xl:table-cell">Type</TableHead>
                      <TableHead className="w-[80px]">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {flags.map((flag) => {
                      const state = localState[flag.name];
                      const isEnabled = state?.enabled ?? flag.enabled;
                      const isChanged = state?.changed ?? false;
                      const isMarkedForReset = state?.markedForReset ?? false;
                      const hasOverride = flag.overrideEnabled !== undefined && flag.overrideEnabled !== null;
                      const hasUnsavedChange = isChanged || isMarkedForReset;

                      return (
                        <TableRow key={flag.name} className={hasUnsavedChange ? "bg-muted/50" : ""}>
                          <TableCell>
                            <Switch
                              checked={isMarkedForReset ? false : isEnabled}
                              onCheckedChange={() => handleToggle(flag.name)}
                              disabled={isMarkedForReset}
                              aria-label={`Toggle ${flag.name}`}
                            />
                          </TableCell>
                          <TableCell>
                            <div className="flex items-center gap-2">
                              <span className={`font-medium font-mono text-sm ${isMarkedForReset ? "line-through text-muted-foreground" : ""}`}>
                                {flag.name}
                              </span>
                              {isChanged && (
                                <Badge variant="outline" className="text-xs bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200">
                                  Unsaved
                                </Badge>
                              )}
                              {isMarkedForReset && (
                                <Badge variant="outline" className="text-xs bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200">
                                  Will Reset
                                </Badge>
                              )}
                            </div>
                            {flag.stale && (
                              <Badge variant="outline" className="mt-1 text-xs">
                                Stale
                              </Badge>
                            )}
                          </TableCell>
                          <TableCell className="hidden lg:table-cell text-sm text-muted-foreground">
                            <div className="max-w-[200px] whitespace-normal">
                              {flag.description || "\u2014"}
                            </div>
                          </TableCell>
                          <TableCell className="hidden md:table-cell">
                            {getSourceBadge(flag.source, hasOverride, isMarkedForReset)}
                          </TableCell>
                          <TableCell className="hidden xl:table-cell">
                            {getTypeBadge(flag.type)}
                          </TableCell>
                          <TableCell>
                            {isMarkedForReset ? (
                              <TooltipProvider>
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <Button
                                      variant="ghost"
                                      size="icon"
                                      onClick={() => handleUndoReset(flag.name)}
                                      aria-label={`Undo reset for ${flag.name}`}
                                    >
                                      <RotateCcw className="h-4 w-4 text-blue-600" />
                                    </Button>
                                  </TooltipTrigger>
                                  <TooltipContent>
                                    Undo reset
                                  </TooltipContent>
                                </Tooltip>
                              </TooltipProvider>
                            ) : hasOverride ? (
                              <TooltipProvider>
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <Button
                                      variant="ghost"
                                      size="icon"
                                      onClick={() => handleMarkForReset(flag.name)}
                                      aria-label={`Reset ${flag.name} to platform default`}
                                    >
                                      <RotateCcw className="h-4 w-4" />
                                    </Button>
                                  </TooltipTrigger>
                                  <TooltipContent>
                                    Reset to platform default
                                  </TooltipContent>
                                </Tooltip>
                              </TooltipProvider>
                            ) : null}
                          </TableCell>
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>

                {/* Save/Discard buttons */}
                <div className="flex items-center justify-between pt-4 border-t">
                  <div className="flex gap-2">
                    <Button
                      onClick={handleSave}
                      disabled={!hasChanges || isSaving}
                    >
                      {isSaving ? (
                        <>
                          <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                          Saving...
                        </>
                      ) : (
                        <>
                          <Save className="w-4 h-4 mr-2" />
                          Save Feature Flags
                        </>
                      )}
                    </Button>
                    {hasChanges && (
                      <Button
                        variant="outline"
                        onClick={handleDiscard}
                        disabled={isSaving}
                      >
                        Discard
                      </Button>
                    )}
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {hasChanges ? (
                      <span className="text-yellow-600 dark:text-yellow-400">
                        {changedCount} unsaved change{changedCount > 1 ? "s" : ""}
                      </span>
                    ) : (
                      "No unsaved changes"
                    )}
                  </div>
                </div>
              </>
            )}
          </>
        )}
      </CardContent>
    </Card>
  );
}
