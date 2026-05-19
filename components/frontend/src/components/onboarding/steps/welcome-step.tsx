"use client";

import { Button } from "@/components/ui/button";
import { Sparkles } from "lucide-react";
import type { WizardStepProps } from "../welcome-wizard";

export function WelcomeStep({ onNext }: WizardStepProps) {
  return (
    <div className="flex flex-col items-center text-center space-y-6 py-4">
      <div className="rounded-full bg-primary/10 p-4">
        <Sparkles className="h-10 w-10 text-primary" />
      </div>

      <div className="space-y-2">
        <h2 className="text-xl font-semibold tracking-tight">
          Welcome to Ambient Code Platform
        </h2>
        <p className="text-sm text-muted-foreground max-w-md mx-auto">
          An AI-native platform for intelligent agentic sessions. We&apos;ll
          help you set up your first workspace, connect your tools, and start
          your first session.
        </p>
      </div>

      <div className="space-y-3 text-left w-full max-w-sm">
        <StepPreview number={1} label="Create a workspace" />
        <StepPreview number={2} label="Connect integrations" />
        <StepPreview number={3} label="Start your first session" />
      </div>

      <Button size="lg" onClick={() => onNext()} className="mt-4">
        Get Started
      </Button>
    </div>
  );
}

function StepPreview({ number, label }: { number: number; label: string }) {
  return (
    <div className="flex items-center gap-3">
      <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full border bg-background text-sm font-medium">
        {number}
      </div>
      <span className="text-sm">{label}</span>
    </div>
  );
}
