"use client";

import { useState, useCallback, useEffect } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Progress } from "@/components/ui/progress";
import { WelcomeStep } from "./steps/welcome-step";
import { CreateWorkspaceStep } from "./steps/create-workspace-step";
import { IntegrationsStep, SESSION_KEY } from "./steps/integrations-step";
import { CompletionStep } from "./steps/completion-step";

/** Shared state passed through every step. */
export type WizardState = {
  createdWorkspaceName: string | null;
};

/** Props that every step component receives. */
export type WizardStepProps = {
  onNext: (update?: Partial<WizardState>) => void;
  onSkip: () => void;
  wizardState: WizardState;
};

type WelcomeWizardProps = {
  open: boolean;
  onDismiss: () => void;
};

const STEPS = [WelcomeStep, CreateWorkspaceStep, IntegrationsStep, CompletionStep];

export function WelcomeWizard({ open, onDismiss }: WelcomeWizardProps) {
  const [stepIndex, setStepIndex] = useState(0);
  const [wizardState, setWizardState] = useState<WizardState>({
    createdWorkspaceName: null,
  });

  // Resume from sessionStorage after OAuth redirect (GitHub App install)
  useEffect(() => {
    try {
      const saved = sessionStorage.getItem(SESSION_KEY);
      if (saved) {
        const parsed = JSON.parse(saved) as { step?: number; createdWorkspaceName?: string };
        if (parsed.step !== undefined) setStepIndex(parsed.step);
        if (parsed.createdWorkspaceName)
          setWizardState((s) => ({ ...s, createdWorkspaceName: parsed.createdWorkspaceName ?? null }));
        sessionStorage.removeItem(SESSION_KEY);
      }
    } catch {
      // sessionStorage may be unavailable or contain invalid JSON
    }
  }, []);

  const handleNext = useCallback(
    (update?: Partial<WizardState>) => {
      if (update) setWizardState((prev) => ({ ...prev, ...update }));
      if (stepIndex < STEPS.length - 1) {
        setStepIndex((i) => i + 1);
      } else {
        onDismiss();
      }
    },
    [stepIndex, onDismiss]
  );

  const StepComponent = STEPS[stepIndex];
  const progress = ((stepIndex + 1) / STEPS.length) * 100;
  const isWideStep = stepIndex === 2;
  const isIntermediate = stepIndex > 0 && stepIndex < STEPS.length - 1;

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) onDismiss(); }}>
      <DialogContent
        showCloseButton={!isIntermediate}
        className={`w-[calc(100%-2rem)] max-h-[90vh] overflow-y-auto transition-[max-width] duration-200 ${isWideStep ? "sm:max-w-[95vw] lg:max-w-[1200px]" : "sm:max-w-[720px]"}`}
      >
        <DialogHeader>
          <DialogTitle className="sr-only">Setup Wizard</DialogTitle>
          <DialogDescription className="sr-only">
            Step {stepIndex + 1} of {STEPS.length}
          </DialogDescription>
          <Progress value={progress} className="h-1" />
        </DialogHeader>

        <StepComponent
          onNext={handleNext}
          onSkip={onDismiss}
          wizardState={wizardState}
        />

        {isIntermediate && (
          <button
            type="button"
            onClick={onDismiss}
            className="text-xs text-muted-foreground hover:text-foreground transition-colors self-center mt-2"
          >
            Skip setup
          </button>
        )}
      </DialogContent>
    </Dialog>
  );
}
