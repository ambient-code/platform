"use client";

import { useEffect, useState, useMemo, useRef } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import Link from "next/link";
import { AlertCircle, AlertTriangle, CheckCircle2, ChevronsUpDown, FileUp, Loader2, X } from "lucide-react";
import { useRouter } from "next/navigation";
import { useFlag } from "@/lib/feature-flags";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
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
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { CreateAgenticSessionRequest } from "@/types/agentic-session";
import type { WorkflowSelection } from "@/types/workflow";
import { useCreateSession } from "@/services/queries/use-sessions";
import { useRunnerTypes } from "@/services/queries/use-runner-types";
import {
  AgentOptionsFields,
  claudeAgentOptionsDefaults,
  claudeAgentOptionsSchema,
  type ClaudeAgentOptionsForm,
} from "@/components/claude-agent-options";
import { DEFAULT_RUNNER_TYPE_ID } from "@/services/api/runner-types";
import { useIntegrationsStatus } from "@/services/queries/use-integrations";
import { useModels } from "@/services/queries/use-models";
import { useOOTBWorkflows } from "@/services/queries/use-workflows";
import { toast } from "sonner";

// Static default used for form initialization before the API responds.
const DEFAULT_MODEL = "";

const formSchema = z.object({
  displayName: z.string().max(50).optional(),
  initialPrompt: z.string().max(5000).optional(),
  runnerType: z.string().min(1, "Please select a runner type"),
  model: z.string().min(1, "Please select a model"),
  temperature: z.number().min(0).max(2),
  maxTokens: z.number().min(100).max(8000),
  timeout: z.number().min(60).max(1800),
});

type FormValues = z.infer<typeof formSchema>;

type CreateSessionDialogProps = {
  projectName: string;
  trigger: React.ReactNode;
  onSuccess?: () => void;
};

// Maximum file size for pre-uploads: 10MB
const MAX_PRE_UPLOAD_SIZE = 10 * 1024 * 1024;

export function CreateSessionDialog({
  projectName,
  trigger,
  onSuccess,
}: CreateSessionDialogProps) {
  const [open, setOpen] = useState(false);
  const [selectedWorkflow, setSelectedWorkflow] = useState("none");
  const [workflowSelection, setWorkflowSelection] = useState<WorkflowSelection | null>(null);
  const [customGitUrl, setCustomGitUrl] = useState("");
  const [customBranch, setCustomBranch] = useState("main");
  const [customPath, setCustomPath] = useState("");
  const [pendingFiles, setPendingFiles] = useState<File[]>([]);
  const [uploadingFiles, setUploadingFiles] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const router = useRouter();
  const advancedAgentOptions = useFlag("advanced-agent-options") ?? false;
  const createSessionMutation = useCreateSession();

  const agentOptionsForm = useForm<ClaudeAgentOptionsForm>({
    resolver: zodResolver(claudeAgentOptionsSchema),
    defaultValues: claudeAgentOptionsDefaults,
  });
  const { data: runnerTypes, isLoading: runnerTypesLoading, isError: runnerTypesError, refetch: refetchRunnerTypes } = useRunnerTypes(projectName);
  const { data: integrationsStatus } = useIntegrationsStatus();
  const { data: ootbWorkflows = [], isLoading: workflowsLoading } = useOOTBWorkflows(projectName);

  const githubConfigured = integrationsStatus?.github?.active != null;
  const gitlabConfigured = integrationsStatus?.gitlab?.connected ?? false;
  const atlassianConfigured = integrationsStatus?.jira?.connected ?? false;
  const googleConfigured = integrationsStatus?.google?.connected ?? false;

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      displayName: "",
      initialPrompt: "",
      runnerType: DEFAULT_RUNNER_TYPE_ID,
      model: DEFAULT_MODEL,
      temperature: 0.7,
      maxTokens: 4000,
      timeout: 300,
    },
  });

  const selectedRunnerType = form.watch("runnerType");

  const selectedRunner = useMemo(
    () => runnerTypes?.find((rt) => rt.id === selectedRunnerType),
    [runnerTypes, selectedRunnerType]
  );

  // Fetch models filtered by the selected runner's provider.
  // models.json is the single source of truth — no hardcoded fallback lists.
  // Wait for runner types to load so we know the provider before fetching.
  const { data: modelsData, isLoading: modelsLoading, isError: modelsError } = useModels(
    projectName, open && !runnerTypesLoading && !runnerTypesError, selectedRunner?.provider
  );

  const models = modelsData
    ? modelsData.models.map((m) => ({ value: m.id, label: m.label }))
    : [];

  // Update form model when API response arrives or provider changes
  useEffect(() => {
    if (modelsData?.defaultModel && !form.formState.dirtyFields.model) {
      form.setValue("model", modelsData.defaultModel, { shouldDirty: false });
    }
  }, [modelsData?.defaultModel, form]);

  const handleRunnerTypeChange = (value: string, onChange: (v: string) => void) => {
    onChange(value);
    // Model list will refetch via useModels when provider changes.
    // resetField clears both value AND dirty state so the useEffect
    // above will set the new provider's default model.
    form.resetField("model", { defaultValue: "" });
  };

  const selectedWorkflowDescription = useMemo(() => {
    if (selectedWorkflow === "none") return "A general chat session with no structured workflow.";
    if (selectedWorkflow === "custom") return "Load a workflow from a custom Git repository.";
    const wf = ootbWorkflows.find(w => w.id === selectedWorkflow);
    return wf?.description ?? "";
  }, [selectedWorkflow, ootbWorkflows]);

  const handleWorkflowChange = (value: string) => {
    setSelectedWorkflow(value);
    if (value === "custom") {
      // Custom fields will show inline; update selection when user fills them
      setWorkflowSelection(
        customGitUrl.trim()
          ? { gitUrl: customGitUrl.trim(), branch: customBranch || "main", path: customPath || undefined }
          : null
      );
      return;
    }
    if (value === "none") {
      setWorkflowSelection(null);
      return;
    }
    const workflow = ootbWorkflows.find(w => w.id === value);
    if (workflow) {
      setWorkflowSelection({
        gitUrl: workflow.gitUrl,
        branch: workflow.branch,
        path: workflow.path,
      });
    }
  };

  // Keep workflowSelection in sync with custom fields
  useEffect(() => {
    if (selectedWorkflow === "custom" && customGitUrl.trim()) {
      setWorkflowSelection({
        gitUrl: customGitUrl.trim(),
        branch: customBranch || "main",
        path: customPath || undefined,
      });
    } else if (selectedWorkflow === "custom") {
      setWorkflowSelection(null);
    }
  }, [customGitUrl, customBranch, customPath, selectedWorkflow]);

  const onSubmit = async (values: FormValues) => {
    if (!projectName) return;

    // Validate agent options form before including it in the request
    if (advancedAgentOptions) {
      const agentOptionsValid = await agentOptionsForm.trigger();
      if (!agentOptionsValid) return;
    }

    const request: CreateAgenticSessionRequest = {
      runnerType: values.runnerType,
      llmSettings: {
        model: values.model,
        temperature: values.temperature,
        maxTokens: values.maxTokens,
      },
      timeout: values.timeout,
    };
    const trimmedName = values.displayName?.trim();
    if (trimmedName) {
      request.displayName = trimmedName;
    }
    const trimmedPrompt = values.initialPrompt?.trim();
    if (trimmedPrompt) {
      request.initialPrompt = trimmedPrompt;
    }
    if (workflowSelection) {
      request.activeWorkflow = workflowSelection;
    }

    if (advancedAgentOptions) {
      request.sdkOptions = agentOptionsForm.getValues();
    }

    createSessionMutation.mutate(
      { projectName, data: request },
      {
        onSuccess: async (session) => {
          const sessionName = session.metadata.name;

          // Upload pending files via pre-upload endpoint (direct to S3)
          if (pendingFiles.length > 0) {
            setUploadingFiles(true);
            try {
              for (const file of pendingFiles) {
                const formData = new FormData();
                formData.append("type", "local");
                formData.append("file", file);
                formData.append("filename", file.name);

                await fetch(
                  `/api/projects/${projectName}/agentic-sessions/${sessionName}/workspace/upload`,
                  { method: "POST", body: formData }
                );
              }
            } catch {
              toast.error("Some files failed to upload. You can upload them after the session starts.");
            } finally {
              setUploadingFiles(false);
            }
          }

          setOpen(false);
          form.reset();
          setPendingFiles([]);
          router.push(`/projects/${encodeURIComponent(projectName)}/sessions/${sessionName}`);
          onSuccess?.();
        },
        onError: (error) => {
          toast.error(error.message || "Failed to create session");
        },
      }
    );
  };

  const handleFileAdd = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (!files) return;

    const newFiles: File[] = [];
    for (let i = 0; i < files.length; i++) {
      const file = files[i];
      if (file.size > MAX_PRE_UPLOAD_SIZE) {
        toast.error(`File "${file.name}" exceeds 10MB limit`);
        continue;
      }
      // Avoid duplicates by name
      if (!pendingFiles.some(f => f.name === file.name)) {
        newFiles.push(file);
      }
    }
    setPendingFiles(prev => [...prev, ...newFiles]);

    // Reset input so the same file can be re-added after removal
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
  };

  const handleFileRemove = (index: number) => {
    setPendingFiles(prev => prev.filter((_, i) => i !== index));
  };

  const handleOpenChange = (newOpen: boolean) => {
    setOpen(newOpen);
    if (!newOpen) {
      form.reset();
      setSelectedWorkflow("none");
      setWorkflowSelection(null);
      setCustomGitUrl("");
      setCustomBranch("main");
      setCustomPath("");
      setPendingFiles([]);
      agentOptionsForm.reset();
    }
  };

  const handleTriggerClick = () => {
    setOpen(true);
  };

  return (
    <>
      <div onClick={handleTriggerClick}>{trigger}</div>
      <Dialog open={open} onOpenChange={handleOpenChange}>
        <DialogContent className="w-full max-w-3xl sm:min-w-[650px] max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Create Session</DialogTitle>
          </DialogHeader>

          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
              {/* Session name (optional; same as Edit name in kebab menu) */}
              <FormField
                control={form.control}
                name="displayName"
                render={({ field }) => (
                  <FormItem className="w-full">
                    <FormLabel>Session name</FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        placeholder="Enter a display name..."
                        maxLength={50}
                        disabled={createSessionMutation.isPending}
                      />
                    </FormControl>
                    <p className="text-xs text-muted-foreground">
                      {(field.value ?? "").length}/50 characters. Optional; you can rename later from the session menu.
                    </p>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* Initial message (optional) */}
              <FormField
                control={form.control}
                name="initialPrompt"
                render={({ field }) => (
                  <FormItem className="w-full">
                    <FormLabel>Initial message (optional)</FormLabel>
                    <FormControl>
                      <Textarea
                        {...field}
                        placeholder="Enter an initial message to send to the agent when the session starts..."
                        maxLength={5000}
                        rows={3}
                        disabled={createSessionMutation.isPending}
                      />
                    </FormControl>
                    <p className="text-xs text-muted-foreground">
                      {(field.value ?? "").length}/5000 characters. If a workflow is selected, its startup prompt will be sent first, then this message.
                    </p>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* Workflow Selection — standard shadcn Select with descriptions */}
              <div className="space-y-2">
                <FormLabel>Workflow</FormLabel>
                {workflowsLoading ? (
                  <Skeleton className="h-10 w-full" />
                ) : (
                  <Select
                    value={selectedWorkflow}
                    onValueChange={handleWorkflowChange}
                    disabled={createSessionMutation.isPending}
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
                {selectedWorkflowDescription && (
                  <p className="text-xs text-muted-foreground">
                    {selectedWorkflowDescription}
                  </p>
                )}
                {/* Custom workflow fields — shown inline when "Custom workflow..." selected */}
                {selectedWorkflow === "custom" && (
                  <>
                    <div className="space-y-1">
                      <FormLabel className="text-xs">Git Repository URL *</FormLabel>
                      <Input
                        value={customGitUrl}
                        onChange={(e) => setCustomGitUrl(e.target.value)}
                        placeholder="https://github.com/org/workflow-repo.git"
                        disabled={createSessionMutation.isPending}
                      />
                    </div>
                    <div className="space-y-1">
                      <FormLabel className="text-xs">Branch</FormLabel>
                      <Input
                        value={customBranch}
                        onChange={(e) => setCustomBranch(e.target.value)}
                        placeholder="main"
                        disabled={createSessionMutation.isPending}
                      />
                    </div>
                    <div className="space-y-1">
                      <FormLabel className="text-xs">Path (optional)</FormLabel>
                      <Input
                        value={customPath}
                        onChange={(e) => setCustomPath(e.target.value)}
                        placeholder="workflows/my-workflow"
                        disabled={createSessionMutation.isPending}
                      />
                    </div>
                  </>
                )}
              </div>

              {/* Runner Type Selection */}
              <FormField
                control={form.control}
                name="runnerType"
                render={({ field }) => (
                  <FormItem className="w-full">
                    <FormLabel>Runner Type</FormLabel>
                    {runnerTypesLoading ? (
                      <Skeleton className="h-10 w-full" />
                    ) : runnerTypesError ? (
                      <Alert variant="destructive">
                        <AlertCircle className="h-4 w-4" />
                        <AlertDescription className="flex items-center justify-between">
                          <span>Failed to load runner types.</span>
                          <Button type="button" variant="outline" size="sm" onClick={() => refetchRunnerTypes()}>
                            Retry
                          </Button>
                        </AlertDescription>
                      </Alert>
                    ) : (
                      <Select
                        onValueChange={(v) => handleRunnerTypeChange(v, field.onChange)}
                        defaultValue={field.value}
                      >
                        <FormControl>
                          <SelectTrigger className="w-full">
                            <SelectValue placeholder="Select a runner type" />
                          </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                          {runnerTypes?.map((rt) => (
                            <SelectItem key={rt.id} value={rt.id}>
                              {rt.displayName}
                            </SelectItem>
                          )) ?? (
                            <SelectItem value={DEFAULT_RUNNER_TYPE_ID}>Claude Agent SDK</SelectItem>
                          )}
                        </SelectContent>
                      </Select>
                    )}
                    {selectedRunner && (
                      <p className="text-xs text-muted-foreground">
                        {selectedRunner.description}
                      </p>
                    )}
                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* Model Selection */}
              <FormField
                control={form.control}
                name="model"
                render={({ field }) => (
                  <FormItem className="w-full">
                    <FormLabel>Model</FormLabel>
                    <Select
                      onValueChange={field.onChange}
                      value={field.value}
                      disabled={modelsLoading}
                    >
                      <FormControl>
                        <SelectTrigger className="w-full">
                          <SelectValue
                            placeholder={modelsLoading ? "Loading models..." : "Select a model"}
                          />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        {models.length === 0 && !modelsLoading ? (
                          <div className="p-2 text-sm text-muted-foreground">
                            No models available for this runner
                          </div>
                        ) : (
                          models.map((m) => (
                            <SelectItem key={m.value} value={m.value}>
                              {m.label}
                            </SelectItem>
                          ))
                        )}
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* File Attachments (pre-upload to S3) */}
              <div className="space-y-2">
                <FormLabel>Files (optional)</FormLabel>
                <input
                  ref={fileInputRef}
                  type="file"
                  multiple
                  onChange={handleFileAdd}
                  disabled={createSessionMutation.isPending || uploadingFiles}
                  className="sr-only"
                  aria-label="Attach files"
                />
                <button
                  type="button"
                  onClick={() => fileInputRef.current?.click()}
                  disabled={createSessionMutation.isPending || uploadingFiles}
                  className="w-full border-2 border-dashed border-muted-foreground/25 hover:border-primary/50 rounded-lg p-4 flex flex-col items-center gap-1 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <FileUp className="h-6 w-6 text-muted-foreground/60" />
                  <span className="text-sm font-medium">Click to attach files</span>
                  <span className="text-xs text-muted-foreground">
                    Files will be available in the session workspace when it starts. Max 10MB per file.
                  </span>
                </button>
                {pendingFiles.length > 0 && (
                  <div className="space-y-1">
                    {pendingFiles.map((file, index) => (
                      <div
                        key={`${file.name}-${index}`}
                        className="flex items-center justify-between text-sm p-2 bg-muted/50 rounded"
                      >
                        <span className="truncate mr-2">
                          {file.name} ({(file.size / 1024).toFixed(1)} KB)
                        </span>
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          className="h-6 w-6 p-0 flex-shrink-0"
                          onClick={() => handleFileRemove(index)}
                          disabled={createSessionMutation.isPending || uploadingFiles}
                        >
                          <X className="h-3 w-3" />
                        </Button>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Advanced Agent Options (behind feature flag) */}
              {advancedAgentOptions && (
                <Collapsible className="w-full space-y-2">
                  <CollapsibleTrigger className="flex items-center justify-between w-full">
                    <FormLabel className="cursor-pointer">Advanced Agent Options</FormLabel>
                    <ChevronsUpDown className="h-4 w-4 text-muted-foreground" />
                  </CollapsibleTrigger>
                  <CollapsibleContent className="space-y-2 pt-2">
                    <Form {...agentOptionsForm}>
                      <AgentOptionsFields
                        form={agentOptionsForm}
                        disabled={createSessionMutation.isPending}
                      />
                    </Form>
                  </CollapsibleContent>
                </Collapsible>
              )}

              {/* Integration auth status */}
              <Collapsible className="w-full space-y-2">
                <CollapsibleTrigger className="flex items-center justify-between w-full">
                  <FormLabel className="cursor-pointer">Integrations</FormLabel>
                  <ChevronsUpDown className="h-4 w-4 text-muted-foreground" />
                </CollapsibleTrigger>
                <CollapsibleContent className="space-y-2">
                {/* GitHub card */}
                {githubConfigured ? (
                  <div className="flex items-start justify-between gap-3 p-3 border rounded-lg bg-background/50">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <div className="flex-shrink-0">
                          <CheckCircle2 className="h-4 w-4 text-green-600" />
                        </div>
                        <h4 className="font-medium text-sm">GitHub</h4>
                      </div>
                      <p className="text-xs text-muted-foreground mt-0.5">
                        Authenticated. Git push and repository access enabled.
                      </p>
                    </div>
                  </div>
                ) : (
                  <div className="flex items-start gap-3 p-3 border rounded-lg bg-background/50">
                    <div className="flex-shrink-0">
                      <AlertTriangle className="h-4 w-4 text-amber-500" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <h4 className="font-medium text-sm">GitHub</h4>
                      <p className="text-xs text-muted-foreground mt-0.5">
                        Not connected.{" "}
                        <Link href="/integrations" className="text-primary hover:underline">
                          Set up
                        </Link>{" "}
                        to enable repository access.
                      </p>
                    </div>
                  </div>
                )}
                {/* GitLab card */}
                {gitlabConfigured ? (
                  <div className="flex items-start justify-between gap-3 p-3 border rounded-lg bg-background/50">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <div className="flex-shrink-0">
                          <CheckCircle2 className="h-4 w-4 text-green-600" />
                        </div>
                        <h4 className="font-medium text-sm">GitLab</h4>
                      </div>
                      <p className="text-xs text-muted-foreground mt-0.5">
                        Authenticated. Git push and repository access enabled.
                      </p>
                    </div>
                  </div>
                ) : (
                  <div className="flex items-start gap-3 p-3 border rounded-lg bg-background/50">
                    <div className="flex-shrink-0">
                      <AlertTriangle className="h-4 w-4 text-amber-500" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <h4 className="font-medium text-sm">GitLab</h4>
                      <p className="text-xs text-muted-foreground mt-0.5">
                        Not connected.{" "}
                        <Link href="/integrations" className="text-primary hover:underline">
                          Set up
                        </Link>{" "}
                        to enable repository access.
                      </p>
                    </div>
                  </div>
                )}
                {/* Google Workspace card */}
                {googleConfigured ? (
                  <div className="flex items-start justify-between gap-3 p-3 border rounded-lg bg-background/50">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <div className="flex-shrink-0">
                          <CheckCircle2 className="h-4 w-4 text-green-600" />
                        </div>
                        <h4 className="font-medium text-sm">Google Workspace</h4>
                      </div>
                      <p className="text-xs text-muted-foreground mt-0.5">
                        Authenticated. Drive, Calendar, and Gmail access enabled.
                      </p>
                    </div>
                  </div>
                ) : (
                  <div className="flex items-start gap-3 p-3 border rounded-lg bg-background/50">
                    <div className="flex-shrink-0">
                      <AlertTriangle className="h-4 w-4 text-amber-500" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <h4 className="font-medium text-sm">Google Workspace</h4>
                      <p className="text-xs text-muted-foreground mt-0.5">
                        Not connected.{" "}
                        <Link href="/integrations" className="text-primary hover:underline">
                          Set up
                        </Link>{" "}
                        to enable Drive, Calendar, and Gmail access.
                      </p>
                    </div>
                  </div>
                )}
                {/* Jira card */}
                {atlassianConfigured ? (
                  <div className="flex items-start justify-between gap-3 p-3 border rounded-lg bg-background/50">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <div className="flex-shrink-0">
                          <CheckCircle2 className="h-4 w-4 text-green-600" />
                        </div>
                        <h4 className="font-medium text-sm">Jira</h4>
                      </div>
                      <p className="text-xs text-muted-foreground mt-0.5">
                        Authenticated. Issue and project access enabled.
                      </p>
                    </div>
                  </div>
                ) : (
                  <div className="flex items-start gap-3 p-3 border rounded-lg bg-background/50">
                    <div className="flex-shrink-0">
                      <AlertTriangle className="h-4 w-4 text-amber-500" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <h4 className="font-medium text-sm">Jira</h4>
                      <p className="text-xs text-muted-foreground mt-0.5">
                        Not connected.{" "}
                        <Link
                          href="/integrations"
                          className="text-primary hover:underline"
                        >
                          Set up
                        </Link>{" "}
                        to enable issue and project access.
                      </p>
                    </div>
                  </div>
                )}
                </CollapsibleContent>
              </Collapsible>

              <DialogFooter>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setOpen(false)}
                  disabled={createSessionMutation.isPending}
                >
                  Cancel
                </Button>
                <Button type="submit" data-testid="create-session-submit" disabled={createSessionMutation.isPending || uploadingFiles || runnerTypesLoading || runnerTypesError || modelsLoading || (modelsError && models.length === 0)}>
                  {(createSessionMutation.isPending || uploadingFiles) && (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  )}
                  {uploadingFiles ? "Uploading files..." : "Create Session"}
                </Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>
    </>
  );
}
