'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { Loader2 } from 'lucide-react';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { errorToast, successToast } from '@/hooks/use-toast';
import { useV1CreateSession, useV1StartSession } from '@/services/queries/v1';
import type { SessionCreateRequest } from '@ambient-platform/sdk';

const models = [
  { value: 'claude-sonnet-4-5', label: 'Claude Sonnet 4.5' },
  { value: 'claude-opus-4-6', label: 'Claude Opus 4.6' },
  { value: 'claude-opus-4-5', label: 'Claude Opus 4.5' },
  { value: 'claude-opus-4-1', label: 'Claude Opus 4.1' },
  { value: 'claude-haiku-4-5', label: 'Claude Haiku 4.5' },
];

const formSchema = z.object({
  name: z.string().min(1, 'Session name is required').max(100),
  prompt: z.string().max(5000).optional(),
  model: z.string().min(1, 'Please select a model'),
  temperature: z.number().min(0).max(2),
  maxTokens: z.number().min(100).max(8000),
  timeout: z.number().min(60).max(1800),
});

type FormValues = z.infer<typeof formSchema>;

type V1CreateSessionDialogProps = {
  projectName: string;
  trigger: React.ReactNode;
  onSuccess?: () => void;
};

export function V1CreateSessionDialog({
  projectName,
  trigger,
  onSuccess,
}: V1CreateSessionDialogProps) {
  const [open, setOpen] = useState(false);
  const createMutation = useV1CreateSession(projectName);
  const startMutation = useV1StartSession(projectName);

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: '',
      prompt: '',
      model: 'claude-sonnet-4-5',
      temperature: 0.7,
      maxTokens: 4000,
      timeout: 300,
    },
  });

  const onSubmit = async (values: FormValues) => {
    if (!projectName) return;

    const request: SessionCreateRequest = {
      name: values.name.trim(),
      llm_model: values.model,
      llm_temperature: values.temperature,
      llm_max_tokens: values.maxTokens,
      timeout: values.timeout,
    };

    if (values.prompt?.trim()) {
      request.prompt = values.prompt.trim();
    }

    createMutation.mutate(request, {
      onSuccess: (session) => {
        startMutation.mutate(session.id, {
          onSuccess: () => {
            successToast(`Session "${session.name}" created and started`);
          },
          onError: () => {
            successToast(`Session "${session.name}" created (start failed — try manually)`);
          },
        });
        setOpen(false);
        form.reset();
        onSuccess?.();
      },
      onError: (err) => {
        errorToast(err instanceof Error ? err.message : 'Failed to create session');
      },
    });
  };

  const handleOpenChange = (newOpen: boolean) => {
    setOpen(newOpen);
    if (!newOpen) {
      form.reset();
    }
  };

  const isPending = createMutation.isPending || startMutation.isPending;

  return (
    <>
      <div onClick={() => setOpen(true)}>{trigger}</div>
      <Dialog open={open} onOpenChange={handleOpenChange}>
        <DialogContent className="w-full max-w-3xl min-w-[650px]">
          <DialogHeader>
            <DialogTitle>Create Session (API Server)</DialogTitle>
          </DialogHeader>

          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem className="w-full">
                    <FormLabel>Session Name *</FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        placeholder="e.g. code-review-sprint-42"
                        maxLength={100}
                        disabled={isPending}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="prompt"
                render={({ field }) => (
                  <FormItem className="w-full">
                    <FormLabel>Initial Prompt</FormLabel>
                    <FormControl>
                      <Textarea
                        {...field}
                        placeholder="Describe what this session should work on..."
                        maxLength={5000}
                        rows={3}
                        disabled={isPending}
                      />
                    </FormControl>
                    <p className="text-xs text-muted-foreground">
                      {(field.value ?? '').length}/5000 characters. Optional.
                    </p>
                    <FormMessage />
                  </FormItem>
                )}
              />

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
                  disabled={isPending}
                >
                  Cancel
                </Button>
                <Button type="submit" disabled={isPending}>
                  {isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
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
