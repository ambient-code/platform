"use client";

import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import Link from "next/link";
import { AlertTriangle, CheckCircle2, Loader2 } from "lucide-react";
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
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import type { CreateAgenticSessionRequest } from "@/types/agentic-session";
import { useCreateSession } from "@/services/queries/use-sessions";
import { useProjectIntegrationStatus } from "@/services/queries/use-projects";
import { useIntegrationSecrets } from "@/services/queries/use-secrets";
import { useIntegrationsStatus } from "@/services/queries/use-integrations";
import { errorToast } from "@/hooks/use-toast";

const models = [
  { value: "claude-sonnet-4-5", label: "Claude Sonnet 4.5" },
  { value: "claude-opus-4-6", label: "Claude Opus 4.6" },
  { value: "claude-opus-4-5", label: "Claude Opus 4.5" },
  { value: "claude-opus-4-1", label: "Claude Opus 4.1" },
  { value: "claude-haiku-4-5", label: "Claude Haiku 4.5" },
];

const formSchema = z.object({
  displayName: z.string().max(50).optional(),
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

export function CreateSessionDialog({
  projectName,
  trigger,
  onSuccess,
}: CreateSessionDialogProps) {
  const [open, setOpen] = useState(false);
  const router = useRouter();
  const createSessionMutation = useCreateSession();

  const { data: integrationStatus } = useProjectIntegrationStatus(projectName);
  const { data: integrationSecrets } = useIntegrationSecrets(projectName);
  const { data: integrationsStatus } = useIntegrationsStatus();

  const githubConfigured = integrationStatus?.github ?? false;
  const gitlabConfigured = integrationsStatus?.gitlab?.connected ?? false;
  const byKey = integrationSecrets
    ? Object.fromEntries(integrationSecrets.map((s) => [s.key, s.value]))
    : {};
  const atlassianConfigured =
    !!(byKey.JIRA_URL?.trim() && byKey.JIRA_PROJECT?.trim() && byKey.JIRA_EMAIL?.trim() && byKey.JIRA_API_TOKEN?.trim());

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      displayName: "",
      model: "claude-sonnet-4-5",
      temperature: 0.7,
      maxTokens: 4000,
      timeout: 300,
    },
  });

  const onSubmit = async (values: FormValues) => {
    if (!projectName) return;

    const request: CreateAgenticSessionRequest = {
      interactive: true,
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

              {/* Integration status (same visual style as integrations accordion), alphabetical: Jira, GitHub, Google Workspace */}
              <div className="w-full space-y-2">
                <FormLabel>Integrations</FormLabel>
                {/* Jira card */}
                {atlassianConfigured ? (
                  <div className="flex items-start justify-between gap-3 p-3 border rounded-lg bg-background/50">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <div className="flex-shrink-0">
                          <CheckCircle2 className="h-4 w-4 text-green-600" />
                        </div>
                        <h4 className="font-medium text-sm">Jira</h4>
                        <Badge variant="secondary" className="text-xs font-normal">
                          read only
                        </Badge>
                      </div>
                      <p className="text-xs text-muted-foreground mt-0.5">
                        MCP access to Jira issues and projects.
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
                        Configure{" "}
                        <Link
                          href="/integrations"
                          className="text-primary hover:underline"
                        >
                          Integrations
                        </Link>{" "}
                        to access Jira MCP in this session.
                      </p>
                    </div>
                  </div>
                )}
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
                        MCP access to GitHub repositories.
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
                        Configure{" "}
                        <Link href="/integrations" className="text-primary hover:underline">
                          Integrations
                        </Link>{" "}
                        to access GitHub MCP in this session.
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
                        MCP access to GitLab repositories.
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
                        Configure{" "}
                        <Link href="/integrations" className="text-primary hover:underline">
                          Integrations
                        </Link>{" "}
                        to access GitLab MCP in this session.
                      </p>
                    </div>
                  </div>
                )}
                {/* Google Workspace card */}
                <div className="flex items-start gap-3 p-3 border rounded-lg bg-background/50">
                  <div className="flex-shrink-0">
                    <AlertTriangle className="h-4 w-4 text-amber-500" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <h4 className="font-medium text-sm">Google Workspace</h4>
                    <p className="text-xs text-muted-foreground mt-0.5">
                      Configure{" "}
                      <Link href="/integrations" className="text-primary hover:underline">
                        Integrations
                      </Link>{" "}
                      to access Google Workspace MCP in this session.
                    </p>
                  </div>
                </div>
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
