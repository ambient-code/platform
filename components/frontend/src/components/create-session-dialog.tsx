"use client";

import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { Loader2 } from "lucide-react";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
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

