"use client";

import { useState, useEffect, useMemo } from "react";
import { Flag, RefreshCw, Loader2, Info, AlertTriangle, Save } from "lucide-react";

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
import { Separator } from "@/components/ui/separator";
import { EmptyState } from "@/components/empty-state";
import { cn } from "@/lib/utils";

import { useFeatureFlags } from "@/services/queries/use-feature-flags-admin";
import * as featureFlagsApi from "@/services/api/feature-flags-admin";
import { successToast, errorToast } from "@/hooks/use-toast";
import { useQueryClient } from "@tanstack/react-query";

type FeatureFlagsSectionProps = {
  projectName: string;
};

// "default" = no override (use platform value), "on" = force enable, "off" = force disable
type OverrideValue = "default" | "on" | "off";

type LocalFlagState = {
  override: OverrideValue;
  serverOverride: OverrideValue; // what the server currently has
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
    const initial: Record<string, LocalFlagState> = {};
    for (const flag of flags) {
      const serverOverride = deriveServerOverride(flag.overrideEnabled);
      initial[flag.name] = {
        override: serverOverride,
        serverOverride,
      };
    }
    setLocalState(initial);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [flagsKey]);

  // Check if there are unsaved changes
  const hasChanges = useMemo(() => {
    return Object.values(localState).some((s) => s.override !== s.serverOverride);
  }, [localState]);

  // Get the count of changed flags
  const changedCount = useMemo(() => {
    return Object.values(localState).filter((s) => s.override !== s.serverOverride).length;
  }, [localState]);

  const handleOverrideChange = (flagName: string, value: OverrideValue) => {
    setLocalState((prev) => {
      const current = prev[flagName];
      if (!current) return prev;
      return {
        ...prev,
        [flagName]: { ...current, override: value },
      };
    });
  };

  const handleSave = async () => {
    const changed = Object.entries(localState).filter(
      ([, s]) => s.override !== s.serverOverride
    );

    if (changed.length === 0) return;

    setIsSaving(true);

    try {
      const promises: Promise<unknown>[] = [];

      for (const [flagName, state] of changed) {
        switch (state.override) {
          case "on":
            promises.push(featureFlagsApi.enableFeatureFlag(projectName, flagName));
            break;
          case "off":
            promises.push(featureFlagsApi.disableFeatureFlag(projectName, flagName));
            break;
          case "default":
            promises.push(featureFlagsApi.removeFeatureFlagOverride(projectName, flagName));
            break;
        }
      }

      await Promise.all(promises);

      successToast(`${changed.length} feature flag${changed.length > 1 ? "s" : ""} updated`);

      queryClient.invalidateQueries({ queryKey: ["feature-flags", "list", projectName] });
      queryClient.invalidateQueries({ queryKey: ["models", projectName] });
    } catch (err) {
      errorToast(err instanceof Error ? err.message : "Failed to save feature flags");
    } finally {
      setIsSaving(false);
    }
  };

  const handleDiscard = () => {
    const initial: Record<string, LocalFlagState> = {};
    for (const flag of flags) {
      const serverOverride = deriveServerOverride(flag.overrideEnabled);
      initial[flag.name] = {
        override: serverOverride,
        serverOverride,
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
              Override platform feature flags for this workspace. Changes are saved when you click Save.
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
              <AlertTitle>Workspace-Scoped Overrides</AlertTitle>
              <AlertDescription>
                Use the override control to force-enable or force-disable features for this workspace.
                Set to &quot;Default&quot; to inherit the platform setting.
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
                      <TableHead>Feature</TableHead>
                      <TableHead className="hidden lg:table-cell">Description</TableHead>
                      <TableHead className="w-[100px]">Default</TableHead>
                      <TableHead className="w-[200px]">Override</TableHead>
                      <TableHead className="hidden xl:table-cell">Type</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {flags.map((flag) => {
                      const state = localState[flag.name];
                      const currentOverride = state?.override ?? "default";
                      const isChanged = state ? state.override !== state.serverOverride : false;
                      return (
                        <TableRow key={flag.name} className={isChanged ? "bg-muted/50" : ""}>
                          <TableCell>
                            <div className="flex items-center gap-2">
                              <span className="font-medium font-mono text-sm">
                                {flag.name}
                              </span>
                              {isChanged && (
                                <Badge variant="outline" className="text-xs bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200">
                                  Unsaved
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
                          <TableCell>
                            <Badge variant={flag.enabled ? "secondary" : "outline"} className="text-xs w-fit">
                              {flag.enabled ? "On" : "Off"}
                            </Badge>
                          </TableCell>
                          <TableCell>
                            <OverrideControl
                              value={currentOverride}
                              onChange={(v) => handleOverrideChange(flag.name, v)}
                            />
                          </TableCell>
                          <TableCell className="hidden xl:table-cell">
                            {getTypeBadge(flag.type)}
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

/** Derive the server override state from the API's overrideEnabled field */
function deriveServerOverride(overrideEnabled?: boolean | null): OverrideValue {
  if (overrideEnabled === undefined || overrideEnabled === null) return "default";
  return overrideEnabled ? "on" : "off";
}

/** Segmented control with three states: Default | On | Off */
function OverrideControl({
  value,
  onChange,
}: {
  value: OverrideValue;
  onChange: (value: OverrideValue) => void;
}) {
  const options: { label: string; val: OverrideValue }[] = [
    { label: "Default", val: "default" },
    { label: "On", val: "on" },
    { label: "Off", val: "off" },
  ];

  return (
    <div className="inline-flex items-center rounded-md border border-input bg-background p-0.5 gap-0.5">
      {options.map((opt) => (
        <Button
          key={opt.val}
          variant="ghost"
          size="sm"
          onClick={() => onChange(opt.val)}
          className={cn(
            "h-auto px-2.5 py-1 text-xs font-medium rounded-sm transition-colors",
            value === opt.val
              ? opt.val === "on"
                ? "bg-green-600 text-white shadow-sm hover:bg-green-700 hover:text-white"
                : opt.val === "off"
                  ? "bg-red-600 text-white shadow-sm hover:bg-red-700 hover:text-white"
                  : "bg-muted text-foreground shadow-sm"
              : "text-muted-foreground hover:text-foreground hover:bg-muted/50"
          )}
        >
          {opt.label}
        </Button>
      ))}
    </div>
  );
}
