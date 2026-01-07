"use client";

import { useState } from "react";
import { useForm, useFieldArray } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { Loader2, Plus, Trash2, GitBranch } from "lucide-react";
import { useRouter } from "next/navigation";

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
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import { Separator } from "@/components/ui/separator";
import type { CreateAgenticSessionRequest } from "@/types/agentic-session";
import { useCreateSession } from "@/services/queries/use-sessions";
import { errorToast } from "@/hooks/use-toast";

const models = [
  { value: "claude-sonnet-4-5", label: "Claude Sonnet 4.5" },
  { value: "claude-opus-4-5", label: "Claude Opus 4.5" },
  { value: "claude-opus-4-1", label: "Claude Opus 4.1" },
  { value: "claude-haiku-4-5", label: "Claude Haiku 4.5" },
];

const formSchema = z.object({
  model: z.string().min(1, "Please select a model"),
  temperature: z.number().min(0).max(2),
  maxTokens: z.number().min(100).max(8000),
  timeout: z.number().min(60).max(1800),
  repos: z.array(z.object({
    input: z.object({
      url: z.string().min(1, "URL is required").url("Must be a valid URL"),
      branch: z.string().optional(),
    }),
    output: z.object({
      url: z.string().min(1, "URL is required").url("Must be a valid URL"),
      branch: z.string().optional(),
    }).optional(),
    autoPush: z.boolean().optional(),
  })).optional(),
});

type FormValues = z.infer<typeof formSchema>;

type CreateSessionDialogProps = {
  projectName: string;
  trigger: React.ReactNode;
  onSuccess?: () => void;
};

export function CreateSessionDialog({
  projectName,
  trigger,
  onSuccess,
}: CreateSessionDialogProps) {
  const [open, setOpen] = useState(false);
  const router = useRouter();
  const createSessionMutation = useCreateSession();

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      model: "claude-sonnet-4-5",
      temperature: 0.7,
      maxTokens: 4000,
      timeout: 300,
      repos: [],
    },
  });

  const { fields, append, remove } = useFieldArray({
    control: form.control,
    name: "repos",
  });

  const onSubmit = async (values: FormValues) => {
    if (!projectName) return;

    // Clean up repos: filter out empty repos and clean empty output objects
    const cleanedRepos = values.repos
      ?.filter(repo => repo.input.url && repo.input.url.trim() !== "")
      .map(repo => ({
        input: repo.input,
        output: repo.output?.url && repo.output.url.trim() !== ""
          ? repo.output
          : undefined,
        autoPush: repo.autoPush,
      }));

    const request: CreateAgenticSessionRequest = {
      interactive: true,
      llmSettings: {
        model: values.model,
        temperature: values.temperature,
        maxTokens: values.maxTokens,
      },
      timeout: values.timeout,
      repos: cleanedRepos && cleanedRepos.length > 0 ? cleanedRepos : undefined,
    };

    createSessionMutation.mutate(
      { projectName, data: request },
      {
        onSuccess: (session) => {
          const sessionName = session.metadata.name;
          setOpen(false);
          form.reset();
          router.push(`/projects/${encodeURIComponent(projectName)}/sessions/${sessionName}`);
          onSuccess?.();
        },
        onError: (error) => {
          errorToast(error.message || "Failed to create session");
        },
      }
    );
  };

  const handleOpenChange = (newOpen: boolean) => {
    setOpen(newOpen);
    if (!newOpen) {
      form.reset();
    }
  };

  const handleTriggerClick = () => {
    setOpen(true);
  };

  return (
    <>
      <div onClick={handleTriggerClick}>{trigger}</div>
      <Dialog open={open} onOpenChange={handleOpenChange}>
        <DialogContent className="w-full max-w-3xl min-w-[650px]">
          <DialogHeader>
            <DialogTitle>Create Session</DialogTitle>
          </DialogHeader>

          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
              {/* Model Selection */}
              <FormField
                control={form.control}
                name="model"
                render={({ field }) => (
                  <FormItem className="w-full">
                    <FormLabel>Model</FormLabel>
                    <Select onValueChange={field.onChange} defaultValue={field.value}>
                      <FormControl>
                        <SelectTrigger className="w-full">
                          <SelectValue placeholder="Select a model" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        {models.map((m) => (
                          <SelectItem key={m.value} value={m.value}>
                            {m.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {/* Repository Configuration */}
              <Separator />

              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <h4 className="text-sm font-medium">Repositories</h4>
                    <p className="text-sm text-muted-foreground">Configure repositories for this session (optional)</p>
                  </div>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => append({ input: { url: "" }, autoPush: false })}
                  >
                    <Plus className="mr-2 h-4 w-4" />
                    Add Repository
                  </Button>
                </div>

                {fields.length === 0 ? (
                  <div className="text-center py-6 text-sm text-muted-foreground border rounded-lg border-dashed">
                    <GitBranch className="h-8 w-8 mx-auto mb-2 opacity-30" />
                    <p>No repositories configured</p>
                    <p className="text-xs mt-1">
                      Add repositories to clone code and work on specific projects
                    </p>
                  </div>
                ) : (
                  fields.map((field, index) => (
                  <div key={field.id} className="space-y-4 rounded-lg border p-4">
                    <div className="flex items-center justify-between">
                      <h5 className="text-sm font-medium">Repository {index + 1}</h5>
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => remove(index)}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>

                    {/* Input Repository */}
                    <div className="space-y-2">
                      <FormLabel>Input Repository</FormLabel>
                      <FormField
                        control={form.control}
                        name={`repos.${index}.input.url`}
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>Repository URL <span className="text-red-500">*</span></FormLabel>
                            <FormControl>
                              <Input
                                placeholder="https://github.com/owner/repo"
                                required
                                {...field}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name={`repos.${index}.input.branch`}
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>Branch</FormLabel>
                            <FormControl>
                              <Input placeholder="main" {...field} />
                            </FormControl>
                            <FormDescription>Branch to clone (optional)</FormDescription>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                    </div>

                    {/* Output Repository */}
                    <div className="space-y-2">
                      <FormLabel>Output Repository (optional)</FormLabel>
                      <FormField
                        control={form.control}
                        name={`repos.${index}.output.url`}
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>Output URL</FormLabel>
                            <FormControl>
                              <Input placeholder="https://github.com/owner/fork" {...field} />
                            </FormControl>
                            <FormDescription>Where to push changes (if different from input)</FormDescription>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      {form.watch(`repos.${index}.output.url`) && (
                        <FormField
                          control={form.control}
                          name={`repos.${index}.output.branch`}
                          render={({ field }) => (
                            <FormItem>
                              <FormLabel>Output Branch</FormLabel>
                              <FormControl>
                                <Input placeholder="feature-branch" {...field} />
                              </FormControl>
                              <FormDescription>Target branch for push</FormDescription>
                              <FormMessage />
                            </FormItem>
                          )}
                        />
                      )}
                    </div>

                    {/* AutoPush - only show if output URL is configured */}
                    {form.watch(`repos.${index}.output.url`) && (
                      <FormField
                        control={form.control}
                        name={`repos.${index}.autoPush`}
                        render={({ field }) => (
                          <FormItem className="flex flex-row items-start space-x-3 space-y-0">
                            <FormControl>
                              <Checkbox
                                checked={field.value}
                                onCheckedChange={field.onChange}
                              />
                            </FormControl>
                            <div className="space-y-1 leading-none">
                              <FormLabel>
                                Enable Auto-Push
                              </FormLabel>
                              <FormDescription>
                                Automatically push changes to the output repository
                              </FormDescription>
                            </div>
                          </FormItem>
                        )}
                      />
                    )}
                  </div>
                  ))
                )}
              </div>

              <DialogFooter>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setOpen(false)}
                  disabled={createSessionMutation.isPending}
                >
                  Cancel
                </Button>
                <Button type="submit" disabled={createSessionMutation.isPending}>
                  {createSessionMutation.isPending && (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  )}
                  Create Session
                </Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>
    </>
  );
}

