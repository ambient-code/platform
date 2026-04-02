"use client";

import { useState, useMemo, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { ArrowLeft, Loader2, AlertCircle, X, Info } from "lucide-react";
import { getCronDescription, getNextRuns } from "@/lib/cron";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Checkbox } from "@/components/ui/checkbox";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { Separator } from "@/components/ui/separator";
import { useCreateScheduledSession } from "@/services/queries/use-scheduled-sessions";
import { useRunnerTypes } from "@/services/queries/use-runner-types";
import { useModels } from "@/services/queries/use-models";
import { useOOTBWorkflows } from "@/services/queries/use-workflows";
import { DEFAULT_RUNNER_TYPE_ID } from "@/services/api/runner-types";
import type { WorkflowSelection } from "@/types/workflow";
import type { SessionRepo } from "@/types/api/sessions";
import { INACTIVITY_TIMEOUT_TOOLTIP } from "@/lib/constants";
import { Label } from "@/components/ui/label";
import { useWorkspaceFlag } from "@/services/queries/use-feature-flags-admin";
import { toast } from "sonner";

const SCHEDULE_PRESETS = [
  { label: "Every hour", value: "0 * * * *" },
  { label: "Daily at 9:00 AM", value: "0 9 * * *" },
  { label: "Every weekday at 9:00 AM", value: "0 9 * * 1-5" },
  { label: "Weekly on Monday", value: "0 9 * * 1" },
  { label: "Custom", value: "custom" },
] as const;

const formSchema = z.object({
  displayName: z.string().max(50).optional(),
  schedulePreset: z.string().min(1, "Please select a schedule"),
  customCron: z.string().optional(),
  initialPrompt: z.string().refine(
    (value) => value.trim().length > 0,
    { message: "Initial prompt is required" }
  ),
  runnerType: z.string().min(1, "Please select a runner type"),
  model: z.string().min(1, "Please select a model"),
  inactivityTimeout: z.string().optional()
    .refine(
      (val) => !val?.trim() || (!isNaN(Number(val)) && Number(val) >= 0 && Number.isInteger(Number(val))),
      { message: "Must be a non-negative integer" }
    ),
  reuseLastSession: z.boolean().optional(),
}).refine(
  (data) => {
    if (data.schedulePreset === "custom") {
      return !!data.customCron?.trim();
    }
    return true;
  },
  { message: "Cron expression is required", path: ["customCron"] }
);

type FormValues = z.infer<typeof formSchema>;

export default function CreateScheduledSessionPage() {
  const params = useParams<{ name: string }>();
  const router = useRouter();
  const projectName = params.name;

  const [selectedWorkflow, setSelectedWorkflow] = useState("none");
  const [customGitUrl, setCustomGitUrl] = useState("");
  const [customBranch, setCustomBranch] = useState("main");
  const [customPath, setCustomPath] = useState("");
  const [repos, setRepos] = useState<SessionRepo[]>([]);

  const createMutation = useCreateScheduledSession();
  const { data: runnerTypes, isLoading: runnerTypesLoading, isError: runnerTypesError, refetch: refetchRunnerTypes } = useRunnerTypes(projectName);
  const { data: ootbWorkflows = [], isLoading: workflowsLoading } = useOOTBWorkflows(projectName);
  const { enabled: reuseFeatureEnabled } = useWorkspaceFlag(projectName, "scheduled-session.reuse.enabled");

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      displayName: "",
      schedulePreset: "0 * * * *",
      customCron: "",
      initialPrompt: "",
      runnerType: DEFAULT_RUNNER_TYPE_ID,
      model: "",
      inactivityTimeout: "",
      reuseLastSession: false,
    },
  });

  const schedulePreset = form.watch("schedulePreset");
  const customCron = form.watch("customCron");
  const selectedRunnerType = form.watch("runnerType");

  const selectedRunner = useMemo(
    () => runnerTypes?.find((rt) => rt.id === selectedRunnerType),
    [runnerTypes, selectedRunnerType]
  );

  const { data: modelsData, isLoading: modelsLoading, isError: modelsError } = useModels(
    projectName, !runnerTypesLoading && !runnerTypesError, selectedRunner?.provider
  );

  const models = modelsData
    ? modelsData.models.map((m) => ({ value: m.id, label: m.label }))
    : [];

  useEffect(() => {
    if (modelsData?.defaultModel && !form.formState.dirtyFields.model) {
      form.setValue("model", modelsData.defaultModel, { shouldDirty: false });
    }
  }, [modelsData?.defaultModel, form]);

  const effectiveCron = schedulePreset === "custom" ? (customCron ?? "") : schedulePreset;
  const nextRuns = useMemo(() => getNextRuns(effectiveCron, 3), [effectiveCron]);
  const cronDescription = useMemo(() => effectiveCron ? getCronDescription(effectiveCron) : "", [effectiveCron]);

  const handleRunnerTypeChange = (value: string, onChange: (v: string) => void) => {
    onChange(value);
    form.resetField("model", { defaultValue: "" });
  };

  const handleWorkflowChange = (value: string) => {
    setSelectedWorkflow(value);
  };

  const addRepo = () => setRepos([...repos, { url: "", branch: "", autoPush: false }]);
  const removeRepo = (index: number) => setRepos(repos.filter((_, i) => i !== index));
  const updateRepo = (index: number, field: keyof SessionRepo, value: string | boolean) => {
    setRepos(repos.map((r, i) => i === index ? { ...r, [field]: value } : r));
  };

  const backUrl = `/projects/${encodeURIComponent(projectName)}?section=schedules`;

  const onSubmit = (values: FormValues) => {
    let activeWorkflow: WorkflowSelection | undefined;
    if (selectedWorkflow === "custom") {
      const gitUrl = customGitUrl.trim();
      if (!gitUrl) {
        toast.error("Git repository URL is required for a custom workflow");
        return;
      }
      const branch = customBranch.trim() || "main";
      const path = customPath.trim();
      activeWorkflow = { gitUrl, branch, ...(path ? { path } : {}) };
    } else if (selectedWorkflow !== "none") {
      const workflow = ootbWorkflows.find((w) => w.id === selectedWorkflow);
      if (workflow) {
        activeWorkflow = { gitUrl: workflow.gitUrl, branch: workflow.branch, path: workflow.path };
      }
    }

    const schedule = values.schedulePreset === "custom"
      ? (values.customCron ?? "").trim()
      : values.schedulePreset;

    const validRepos = repos
      .map((r) => ({ ...r, url: r.url.trim(), branch: r.branch?.trim() ?? "" }))
      .filter((r) => r.url.length > 0);
    const parsedInactivityTimeout = values.inactivityTimeout?.trim()
      ? parseInt(values.inactivityTimeout.trim(), 10)
      : undefined;

    createMutation.mutate(
      {
        projectName,
        data: {
          displayName: values.displayName?.trim() || undefined,
          schedule,
          reuseLastSession: values.reuseLastSession,
          sessionTemplate: {
            initialPrompt: values.initialPrompt,
            runnerType: values.runnerType,
            llmSettings: { model: values.model, temperature: 0.7, maxTokens: 4000 },
            timeout: 300,
            ...(parsedInactivityTimeout !== undefined ? { inactivityTimeout: parsedInactivityTimeout } : {}),
            ...(activeWorkflow ? { activeWorkflow } : {}),
            ...(validRepos.length > 0 ? { repos: validRepos } : {}),
          },
        },
      },
      {
        onSuccess: () => {
          toast.success("Scheduled session created");
          router.push(backUrl);
        },
        onError: (error) => {
          toast.error(error.message || "Failed to create scheduled session");
        },
      }
    );
  };

  return (
    <div className="space-y-6 p-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="sm" onClick={() => router.push(backUrl)}>
          <ArrowLeft className="h-4 w-4 mr-1" />
          Back
        </Button>
        <h1 className="text-2xl font-bold">Create Scheduled Session</h1>
      </div>

      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6 w-full lg:w-3/4 mx-auto">
          {/* Basics */}
          <Card>
            <CardHeader>
              <CardTitle>Basics</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <FormField
                control={form.control}
                name="displayName"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Name</FormLabel>
                    <FormControl>
                      <Input {...field} placeholder="Enter a display name..." maxLength={50} disabled={createMutation.isPending} />
                    </FormControl>
                    <p className="text-xs text-muted-foreground">{(field.value ?? "").length}/50 characters</p>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="schedulePreset"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Schedule</FormLabel>
                    <Select onValueChange={field.onChange} defaultValue={field.value}>
                      <FormControl>
                        <SelectTrigger className="w-full">
                          <SelectValue placeholder="Select a schedule" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        {SCHEDULE_PRESETS.map((preset) => (
                          <SelectItem key={preset.value} value={preset.value}>{preset.label}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {schedulePreset === "custom" && (
                <FormField
                  control={form.control}
                  name="customCron"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Cron Expression</FormLabel>
                      <FormControl>
                        <Input {...field} placeholder="*/15 * * * *" disabled={createMutation.isPending} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}

              {effectiveCron && (
                <div className="rounded-md border p-3 space-y-2">
                  <p className="text-sm font-medium">{cronDescription}</p>
                  {nextRuns.length > 0 && (
                    <div className="text-xs text-muted-foreground space-y-0.5">
                      <p className="font-medium">Next 3 runs:</p>
                      {nextRuns.map((date, i) => (
                        <p key={i}>{date.toLocaleString()}</p>
                      ))}
                    </div>
                  )}
                </div>
              )}

              {reuseFeatureEnabled && (
                <FormField
                  control={form.control}
                  name="reuseLastSession"
                  render={({ field }) => (
                    <FormItem>
                      <div className="flex items-center space-x-2">
                        <FormControl>
                          <Checkbox
                            id="reuse-last-session"
                            checked={field.value}
                            onCheckedChange={field.onChange}
                            disabled={createMutation.isPending}
                          />
                        </FormControl>
                        <div className="flex items-center gap-1.5">
                          <Label htmlFor="reuse-last-session" className="cursor-pointer">
                            Reuse last session
                          </Label>
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger type="button" aria-label="Reuse last session help">
                                <Info className="h-3.5 w-3.5 text-muted-foreground" />
                              </TooltipTrigger>
                              <TooltipContent side="right" className="max-w-xs">
                                <p>Instead of creating a new session each time, reuse the most recent one. If the session is still running, the prompt is sent as a follow-up message. If it has stopped, it is resumed with the prompt.</p>
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        </div>
                      </div>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}
            </CardContent>
          </Card>

          {/* Prompt & Workflow */}
          <Card>
            <CardHeader>
              <CardTitle>Prompt & Workflow</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <FormField
                control={form.control}
                name="initialPrompt"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Initial Prompt</FormLabel>
                    <FormControl>
                      <Textarea {...field} placeholder="Enter the prompt for each scheduled session..." rows={4} disabled={createMutation.isPending} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <div className="space-y-2">
                <FormLabel>Workflow</FormLabel>
                {workflowsLoading ? (
                  <Skeleton className="h-10 w-full" />
                ) : (
                  <Select
                    value={selectedWorkflow}
                    onValueChange={handleWorkflowChange}
                    disabled={createMutation.isPending}
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue placeholder="Select workflow..." />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="none">General chat</SelectItem>
                      {ootbWorkflows
                        .filter(w => w.enabled)
                        .sort((a, b) => a.name.localeCompare(b.name))
                        .map((workflow) => (
                          <SelectItem key={workflow.id} value={workflow.id}>
                            {workflow.name}
                          </SelectItem>
                        ))}
                      <SelectItem value="custom">Custom workflow...</SelectItem>
                    </SelectContent>
                  </Select>
                )}
                {selectedWorkflow === "custom" && (
                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 pt-1">
                    <div className="sm:col-span-2 space-y-1">
                      <FormLabel className="text-xs">Git Repository URL *</FormLabel>
                      <Input
                        value={customGitUrl}
                        onChange={(e) => setCustomGitUrl(e.target.value)}
                        placeholder="https://github.com/org/workflow-repo.git"
                        disabled={createMutation.isPending}
                      />
                    </div>
                    <div className="space-y-1">
                      <FormLabel className="text-xs">Branch</FormLabel>
                      <Input
                        value={customBranch}
                        onChange={(e) => setCustomBranch(e.target.value)}
                        placeholder="main"
                        disabled={createMutation.isPending}
                      />
                    </div>
                    <div className="sm:col-span-3 space-y-1">
                      <FormLabel className="text-xs">Path (optional)</FormLabel>
                      <Input
                        value={customPath}
                        onChange={(e) => setCustomPath(e.target.value)}
                        placeholder="workflows/my-workflow"
                        disabled={createMutation.isPending}
                      />
                    </div>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Runner & Model */}
          <Card>
            <CardHeader>
              <CardTitle>Runner & Model</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <FormField
                  control={form.control}
                  name="runnerType"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Runner Type</FormLabel>
                      {runnerTypesLoading ? (
                        <Skeleton className="h-10 w-full" />
                      ) : runnerTypesError ? (
                        <Alert variant="destructive">
                          <AlertCircle className="h-4 w-4" />
                          <AlertDescription className="flex items-center justify-between">
                            <span>Failed to load.</span>
                            <Button type="button" variant="outline" size="sm" onClick={() => refetchRunnerTypes()}>Retry</Button>
                          </AlertDescription>
                        </Alert>
                      ) : (
                        <Select onValueChange={(v) => handleRunnerTypeChange(v, field.onChange)} defaultValue={field.value}>
                          <FormControl>
                            <SelectTrigger className="w-full">
                              <SelectValue placeholder="Select a runner type" />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            {runnerTypes?.map((rt) => (
                              <SelectItem key={rt.id} value={rt.id}>{rt.displayName}</SelectItem>
                            )) ?? (
                              <SelectItem value={DEFAULT_RUNNER_TYPE_ID}>Claude Agent SDK</SelectItem>
                            )}
                          </SelectContent>
                        </Select>
                      )}
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name="model"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Model</FormLabel>
                      <Select onValueChange={field.onChange} value={field.value} disabled={modelsLoading || (modelsError && models.length === 0)}>
                        <FormControl>
                          <SelectTrigger className="w-full">
                            <SelectValue placeholder={modelsLoading ? "Loading models..." : "Select a model"} />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          {models.length === 0 && !modelsLoading ? (
                            <div className="p-2 text-sm text-muted-foreground">No models available for this runner</div>
                          ) : (
                            models.map((m) => (
                              <SelectItem key={m.value} value={m.value}>{m.label}</SelectItem>
                            ))
                          )}
                        </SelectContent>
                      </Select>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>

              <Separator />

              <FormField
                control={form.control}
                name="inactivityTimeout"
                render={({ field }) => (
                  <FormItem>
                    <div className="flex items-center gap-1.5">
                      <FormLabel>Inactivity Timeout (seconds)</FormLabel>
                      <TooltipProvider>
                        <Tooltip>
                          <TooltipTrigger type="button" aria-label="Inactivity timeout help">
                            <Info className="h-3.5 w-3.5 text-muted-foreground" />
                          </TooltipTrigger>
                          <TooltipContent side="right" className="max-w-xs">
                            <p>{INACTIVITY_TIMEOUT_TOOLTIP}</p>
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    </div>
                    <FormControl>
                      <Input
                        className="max-w-xs [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                        {...field}
                        placeholder="Enter timeout value..."
                        disabled={createMutation.isPending}
                      />
                    </FormControl>
                    <p className="text-xs text-muted-foreground">Default: 24 hours (86400s). Set to 0 to disable auto-stop.</p>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </CardContent>
          </Card>

          {/* Context Repositories */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle>Context Repositories</CardTitle>
                <Button type="button" variant="outline" size="sm" onClick={addRepo} disabled={createMutation.isPending}>
                  Add Repository
                </Button>
              </div>
            </CardHeader>
            <CardContent className="space-y-3">
              {repos.length === 0 && (
                <p className="text-sm text-muted-foreground">No repositories added. Sessions will run without repository context.</p>
              )}
              {repos.map((repo, index) => (
                <div key={index} className="rounded-md border p-4 space-y-3">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium text-muted-foreground">Repository {index + 1}</span>
                    <Button type="button" variant="ghost" size="sm" aria-label={`Remove repository ${index + 1}`} onClick={() => removeRepo(index)} disabled={createMutation.isPending}>
                      <X className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                    <div className="sm:col-span-2 space-y-1">
                      <FormLabel className="text-xs">Repository URL</FormLabel>
                      <Input
                        value={repo.url}
                        onChange={(e) => updateRepo(index, "url", e.target.value)}
                        placeholder="https://github.com/org/repo"
                        disabled={createMutation.isPending}
                      />
                      <p className="text-xs text-muted-foreground">Currently supports GitHub repositories for code context</p>
                    </div>
                    <div className="space-y-1">
                      <FormLabel className="text-xs">Branch (optional)</FormLabel>
                      <Input
                        value={repo.branch ?? ""}
                        onChange={(e) => updateRepo(index, "branch", e.target.value)}
                        placeholder="Enter branch name..."
                        disabled={createMutation.isPending}
                      />
                      <p className="text-xs text-muted-foreground">If empty, a unique feature branch is created</p>
                    </div>
                  </div>
                  <div className="space-y-1">
                    <div className="flex items-center space-x-2">
                      <Checkbox
                        id={`auto-push-${index}`}
                        checked={repo.autoPush ?? false}
                        onCheckedChange={(checked) => updateRepo(index, "autoPush", !!checked)}
                        disabled={createMutation.isPending}
                      />
                      <label htmlFor={`auto-push-${index}`} className="text-sm text-muted-foreground cursor-pointer">
                        Enable auto-push
                      </label>
                    </div>
                    <p className="text-xs text-muted-foreground pl-6">
                      Instructs Claude to commit and push changes made to this repository during the session. Requires git credentials to be configured.
                    </p>
                  </div>
                </div>
              ))}
            </CardContent>
          </Card>

          {/* Actions */}
          <div className="flex justify-end gap-3 pb-6">
            <Button type="button" variant="outline" onClick={() => router.push(backUrl)} disabled={createMutation.isPending}>
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={createMutation.isPending || runnerTypesLoading || runnerTypesError || modelsLoading || (modelsError && models.length === 0)}
            >
              {createMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Create Scheduled Session
            </Button>
          </div>
        </form>
      </Form>
    </div>
  );
}
