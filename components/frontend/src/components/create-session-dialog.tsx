"use client";

import { useState, useCallback } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import Link from "next/link";
import { AlertTriangle, CheckCircle2, Loader2, X, Plus, ChevronDown } from "lucide-react";
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
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import type { CreateAgenticSessionRequest } from "@/types/agentic-session";
import { useCreateSession } from "@/services/queries/use-sessions";
import { useIntegrationsStatus } from "@/services/queries/use-integrations";
import { errorToast } from "@/hooks/use-toast";

const models = [
  { value: "claude-sonnet-4-5", label: "Claude Sonnet 4.5" },
  { value: "claude-opus-4-6", label: "Claude Opus 4.6" },
  { value: "claude-opus-4-5", label: "Claude Opus 4.5" },
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
  const [labels, setLabels] = useState<Record<string, string>>({});
  const router = useRouter();
  const createSessionMutation = useCreateSession();

  const { data: integrationsStatus } = useIntegrationsStatus();

  const githubConfigured = integrationsStatus?.github?.active != null;
  const gitlabConfigured = integrationsStatus?.gitlab?.connected ?? false;
  const atlassianConfigured = integrationsStatus?.jira?.connected ?? false;
  const googleConfigured = integrationsStatus?.google?.connected ?? false;

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
    if (Object.keys(labels).length > 0) {
      request.labels = labels;
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
      setLabels({});
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

              {/* Labels */}
              <div className="w-full space-y-2">
                <FormLabel>Labels</FormLabel>
                <LabelEditor
                  labels={labels}
                  onChange={setLabels}
                  disabled={createSessionMutation.isPending}
                  suggestions={["issue", "research", "team", "type", "other"]}
                />
              </div>

              {/* Integration auth status */}
              <div className="w-full space-y-2">
                <FormLabel>Integrations</FormLabel>
                <IntegrationCard name="GitHub" connected={githubConfigured} connectedText="Git push and repository access enabled." disconnectedText="to enable repository access." />
                <IntegrationCard name="GitLab" connected={gitlabConfigured} connectedText="Git push and repository access enabled." disconnectedText="to enable repository access." />
                <IntegrationCard name="Google Workspace" connected={googleConfigured} connectedText="Drive, Calendar, and Gmail access enabled." disconnectedText="to enable Drive, Calendar, and Gmail access." />
                <IntegrationCard name="Jira" connected={atlassianConfigured} connectedText="Issue and project access enabled." disconnectedText="to enable issue and project access." />
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

type IntegrationCardProps = {
  name: string;
  connected: boolean;
  connectedText: string;
  disconnectedText: string;
};

function IntegrationCard({ name, connected, connectedText, disconnectedText }: IntegrationCardProps) {
  return (
    <div className="flex items-start gap-3 p-3 border rounded-lg bg-background/50">
      <div className="flex-shrink-0">
        {connected ? (
          <CheckCircle2 className="h-4 w-4 text-green-600" />
        ) : (
          <AlertTriangle className="h-4 w-4 text-amber-500" />
        )}
      </div>
      <div className="flex-1 min-w-0">
        <h4 className="font-medium text-sm">{name}</h4>
        <p className="text-xs text-muted-foreground mt-0.5">
          {connected ? (
            <>Authenticated. {connectedText}</>
          ) : (
            <>
              Not connected.{" "}
              <Link href="/integrations" className="text-primary hover:underline">
                Set up
              </Link>{" "}
              {disconnectedText}
            </>
          )}
        </p>
      </div>
    </div>
  );
}

// K8s label segment: 1-63 chars, alphanumeric start/end, dashes/dots/underscores allowed
const K8S_LABEL_REGEX = /^[a-zA-Z0-9]([a-zA-Z0-9._-]{0,61}[a-zA-Z0-9])?$/;

function isValidLabelSegment(s: string): boolean {
  return s.length > 0 && s.length <= 63 && K8S_LABEL_REGEX.test(s);
}

type LabelEditorProps = {
  labels: Record<string, string>;
  onChange: (labels: Record<string, string>) => void;
  disabled?: boolean;
  suggestions?: string[];
};

const DEFAULT_SUGGESTIONS = ["issue", "research", "team", "type", "other"];

function LabelEditor({
  labels,
  onChange,
  disabled = false,
  suggestions = DEFAULT_SUGGESTIONS,
}: LabelEditorProps) {
  const [inputValue, setInputValue] = useState("");
  const [suggestionsOpen, setSuggestionsOpen] = useState(false);
  const [validationError, setValidationError] = useState<string | null>(null);

  const handleRemove = useCallback(
    (key: string) => {
      const next = { ...labels };
      delete next[key];
      onChange(next);
    },
    [labels, onChange]
  );

  const handleAdd = useCallback(() => {
    const trimmed = inputValue.trim();
    if (!trimmed) return;

    const colonIdx = trimmed.indexOf(":");
    if (colonIdx <= 0 || colonIdx === trimmed.length - 1) return;

    const key = trimmed.slice(0, colonIdx).trim();
    const value = trimmed.slice(colonIdx + 1).trim();
    if (!key || !value) return;

    if (!isValidLabelSegment(key)) {
      setValidationError(`Key "${key}" must be 1-63 alphanumeric chars (dashes, dots, underscores allowed)`);
      return;
    }
    if (!isValidLabelSegment(value)) {
      setValidationError(`Value "${value}" must be 1-63 alphanumeric chars (dashes, dots, underscores allowed)`);
      return;
    }

    setValidationError(null);
    onChange({ ...labels, [key]: value });
    setInputValue("");
  }, [inputValue, labels, onChange]);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault();
      handleAdd();
    }
  };

  const handleSuggestionClick = (suggestion: string) => {
    setInputValue(`${suggestion}:`);
    setSuggestionsOpen(false);
    setValidationError(null);
  };

  const entries = Object.entries(labels);

  return (
    <div className="space-y-2">
      {entries.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {entries.map(([key, value]) => (
            <Badge key={key} variant="secondary" className="gap-1 pr-1">
              <span className="font-semibold">{key}</span>
              <span className="text-muted-foreground">=</span>
              <span>{value}</span>
              {!disabled && (
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="h-4 w-4 ml-0.5"
                  onClick={() => handleRemove(key)}
                  aria-label={`Remove label ${key}`}
                >
                  <X className="h-3 w-3" />
                </Button>
              )}
            </Badge>
          ))}
        </div>
      )}

      {!disabled && (
        <div className="flex gap-2">
          <div className="flex-1">
            <Input
              value={inputValue}
              onChange={(e) => { setInputValue(e.target.value); setValidationError(null); }}
              onKeyDown={handleKeyDown}
              placeholder="key:value"
              disabled={disabled}
            />
          </div>
          <Popover open={suggestionsOpen} onOpenChange={setSuggestionsOpen}>
            <PopoverTrigger asChild>
              <Button type="button" variant="outline" size="sm" className="h-9 px-2" disabled={disabled}>
                <ChevronDown className="h-4 w-4" />
              </Button>
            </PopoverTrigger>
            <PopoverContent align="end" className="w-40 p-1">
              {suggestions.map((s) => (
                <button
                  key={s}
                  type="button"
                  onClick={() => handleSuggestionClick(s)}
                  className="w-full text-left text-sm px-2 py-1.5 rounded hover:bg-accent"
                >
                  {s}
                </button>
              ))}
            </PopoverContent>
          </Popover>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="h-9"
            onClick={handleAdd}
            disabled={disabled || !inputValue.includes(":")}
          >
            <Plus className="h-4 w-4 mr-1" />
            Add
          </Button>
        </div>
      )}

      {validationError && (
        <p className="text-xs text-destructive">{validationError}</p>
      )}

      {!disabled && !validationError && (
        <p className="text-xs text-muted-foreground">
          Add labels as key:value pairs. Use the dropdown for common keys.
        </p>
      )}
    </div>
  );
}
