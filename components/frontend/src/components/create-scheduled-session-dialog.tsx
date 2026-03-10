"use client";

import { useState, useMemo, useEffect } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { Loader2, AlertCircle } from "lucide-react";
import { getCronDescription, getNextRuns } from "@/lib/cron";

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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useCreateScheduledSession } from "@/services/queries/use-scheduled-sessions";
import { useRunnerTypes } from "@/services/queries/use-runner-types";
import { useModels } from "@/services/queries/use-models";
import { DEFAULT_RUNNER_TYPE_ID } from "@/services/api/runner-types";
import { toast } from "sonner";

const SCHEDULE_PRESETS = [
  { label: "Every hour", value: "0 * * * *" },
  { label: "Daily at 9:00 AM", value: "0 9 * * *" },
  { label: "Every weekday at 9:00 AM", value: "0 9 * * 1-5" },
  { label: "Weekly on Monday", value: "0 9 * * 1" },
  { label: "Custom", value: "custom" },
] as const;

const DEFAULT_MODEL = "claude-sonnet-4-5";

const formSchema = z.object({
  displayName: z.string().max(50).optional(),
  schedulePreset: z.string().min(1, "Please select a schedule"),
  customCron: z.string().optional(),
  initialPrompt: z.string().min(1, "Initial prompt is required"),
  runnerType: z.string().min(1, "Please select a runner type"),
  model: z.string().min(1, "Please select a model"),
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

type CreateScheduledSessionDialogProps = {
  projectName: string;
  trigger: React.ReactNode;
  onSuccess?: () => void;
};

export function CreateScheduledSessionDialog({
  projectName,
  trigger,
  onSuccess,
}: CreateScheduledSessionDialogProps) {
  const [open, setOpen] = useState(false);
  const createMutation = useCreateScheduledSession();
  const { data: runnerTypes, isLoading: runnerTypesLoading, isError: runnerTypesError, refetch: refetchRunnerTypes } = useRunnerTypes(projectName);

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      displayName: "",
      schedulePreset: "0 * * * *",
      customCron: "",
      initialPrompt: "",
      runnerType: DEFAULT_RUNNER_TYPE_ID,
      model: DEFAULT_MODEL,
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
    projectName, open && !runnerTypesLoading && !runnerTypesError, selectedRunner?.provider
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

  const onSubmit = (values: FormValues) => {
    const schedule = values.schedulePreset === "custom"
      ? (values.customCron ?? "").trim()
      : values.schedulePreset;

    createMutation.mutate(
      {
        projectName,
        data: {
          displayName: values.displayName?.trim() || undefined,
          schedule,
          sessionTemplate: {
            initialPrompt: values.initialPrompt,
            runnerType: values.runnerType,
            llmSettings: { model: values.model, temperature: 0.7, maxTokens: 4000 },
            timeout: 300,
          },
        },
      },
      {
        onSuccess: () => {
          setOpen(false);
          form.reset();
          toast.success("Scheduled session created");
          onSuccess?.();
        },
        onError: (error) => {
          toast.error(error.message || "Failed to create scheduled session");
        },
      }
    );
  };

  const handleOpenChange = (newOpen: boolean) => {
    setOpen(newOpen);
    if (!newOpen) form.reset();
  };

  return (
    <>
      <div onClick={() => setOpen(true)}>{trigger}</div>
      <Dialog open={open} onOpenChange={handleOpenChange}>
        <DialogContent className="w-full max-w-2xl min-w-[550px]">
          <DialogHeader>
            <DialogTitle>Create Scheduled Session</DialogTitle>
          </DialogHeader>

          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
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

              <FormField
                control={form.control}
                name="initialPrompt"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Initial Prompt</FormLabel>
                    <FormControl>
                      <Textarea {...field} placeholder="Enter the prompt for each scheduled session..." rows={3} disabled={createMutation.isPending} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <RunnerTypeField
                form={form}
                runnerTypes={runnerTypes}
                runnerTypesLoading={runnerTypesLoading}
                runnerTypesError={runnerTypesError}
                refetchRunnerTypes={refetchRunnerTypes}
                onRunnerTypeChange={handleRunnerTypeChange}
              />

              <ModelField form={form} models={models} modelsLoading={modelsLoading} modelsError={modelsError} />

              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => setOpen(false)} disabled={createMutation.isPending}>
                  Cancel
                </Button>
                <Button
                  type="submit"
                  disabled={createMutation.isPending || runnerTypesLoading || runnerTypesError || modelsLoading || (modelsError && models.length === 0)}
                >
                  {createMutation.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  Create
                </Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>
    </>
  );
}

type RunnerTypeFieldProps = {
  form: ReturnType<typeof useForm<FormValues>>;
  runnerTypes: ReturnType<typeof useRunnerTypes>["data"];
  runnerTypesLoading: boolean;
  runnerTypesError: boolean;
  refetchRunnerTypes: () => void;
  onRunnerTypeChange: (value: string, onChange: (v: string) => void) => void;
};

function RunnerTypeField({ form, runnerTypes, runnerTypesLoading, runnerTypesError, refetchRunnerTypes, onRunnerTypeChange }: RunnerTypeFieldProps) {
  return (
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
                <span>Failed to load runner types.</span>
                <Button type="button" variant="outline" size="sm" onClick={() => refetchRunnerTypes()}>Retry</Button>
              </AlertDescription>
            </Alert>
          ) : (
            <Select onValueChange={(v) => onRunnerTypeChange(v, field.onChange)} defaultValue={field.value}>
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
  );
}

type ModelFieldProps = {
  form: ReturnType<typeof useForm<FormValues>>;
  models: { value: string; label: string }[];
  modelsLoading: boolean;
  modelsError: boolean;
};

function ModelField({ form, models, modelsLoading, modelsError }: ModelFieldProps) {
  return (
    <FormField
      control={form.control}
      name="model"
      render={({ field }) => (
        <FormItem>
          <FormLabel>Model</FormLabel>
          <Select onValueChange={field.onChange} value={field.value} disabled={modelsLoading}>
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
  );
}
