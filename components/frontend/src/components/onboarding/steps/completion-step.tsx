"use client";

import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { CheckCircle2, MessageSquarePlus, Settings } from "lucide-react";
import type { WizardStepProps } from "../welcome-wizard";

export function CompletionStep({ onNext, wizardState }: WizardStepProps) {
  const router = useRouter();
  const workspaceName = wizardState.createdWorkspaceName;

  const handleStartSession = () => {
    onNext();
    if (workspaceName) {
      router.push(`/projects/${encodeURIComponent(workspaceName)}/new`);
    }
  };

  const handleGoToSettings = () => {
    onNext();
    if (workspaceName) {
      router.push(
        `/projects/${encodeURIComponent(workspaceName)}/settings`
      );
    }
  };

  return (
    <div className="flex flex-col items-center text-center space-y-6 py-4">
      <div className="rounded-full bg-green-100 dark:bg-green-900/30 p-4">
        <CheckCircle2 className="h-10 w-10 text-green-600 dark:text-green-400" />
      </div>

      <div className="space-y-2">
        <h2 className="text-xl font-semibold tracking-tight">
          You&apos;re all set!
        </h2>
        <p className="text-sm text-muted-foreground max-w-md mx-auto">
          Your workspace is ready. Start a session to begin working with AI, or
          configure workspace settings like API keys and storage.
        </p>
      </div>

      <div className="flex flex-col sm:flex-row gap-3 w-full max-w-sm">
        <Button
          size="lg"
          className="flex-1"
          onClick={handleStartSession}
        >
          <MessageSquarePlus className="h-4 w-4 mr-2" />
          Start a session
        </Button>
        <Button
          size="lg"
          variant="outline"
          className="flex-1"
          onClick={handleGoToSettings}
        >
          <Settings className="h-4 w-4 mr-2" />
          Workspace settings
        </Button>
      </div>
    </div>
  );
}
