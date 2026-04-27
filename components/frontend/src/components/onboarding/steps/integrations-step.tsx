"use client";

import { useCallback, useRef, useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Loader2, CheckCircle2, ChevronDown } from "lucide-react";
import { useIntegrationsStatus } from "@/services/queries/use-integrations";
import {
  INTEGRATION_REGISTRY,
  useAppConfig,
} from "../integration-registry";
import type { WizardStepProps } from "../welcome-wizard";

const SESSION_KEY = "acp-onboarding-wizard-state";

export function IntegrationsStep({ onNext, wizardState }: WizardStepProps) {
  const { data: integrations, isLoading, refetch } = useIntegrationsStatus();
  const appConfig = useAppConfig();

  const connectedCount = integrations
    ? INTEGRATION_REGISTRY.filter((entry) =>
        entry.isConnected(integrations)
      ).length
    : 0;

  const persistStateBeforeRedirect = useCallback(() => {
    try {
      sessionStorage.setItem(
        SESSION_KEY,
        JSON.stringify({
          step: 2,
          createdWorkspaceName: wizardState.createdWorkspaceName,
        })
      );
    } catch {
      // sessionStorage may be unavailable
    }
  }, [wizardState.createdWorkspaceName]);

  // GitHub App install navigates away; persist wizard state first
  const handleRefresh = useCallback(() => {
    persistStateBeforeRedirect();
    refetch();
  }, [persistStateBeforeRedirect, refetch]);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="space-y-1">
          <h2 className="text-lg font-semibold">Connect integrations</h2>
          <p className="text-sm text-muted-foreground">
            Connect your tools so sessions can access repositories, issues, and
            documents. You can always do this later from the Integrations page.
          </p>
        </div>
        {!isLoading && integrations && (
          <Badge variant="secondary" className="shrink-0">
            {connectedCount} / {INTEGRATION_REGISTRY.length} connected
          </Badge>
        )}
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-8">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : integrations ? (
        <ScrollableCardGrid>
          {INTEGRATION_REGISTRY.map((entry) => {
            const connected = entry.isConnected(integrations);
            return (
              <div key={entry.id} className="relative">
                {connected && (
                  <div className="absolute top-2 right-2 z-10">
                    <CheckCircle2 className="h-5 w-5 text-green-500" />
                  </div>
                )}
                {entry.renderCard({
                  status: integrations,
                  onRefresh: handleRefresh,
                  appConfig,
                })}
              </div>
            );
          })}
        </ScrollableCardGrid>
      ) : null}

      <div className="flex justify-end gap-2 pt-2">
        <Button variant="ghost" onClick={() => onNext()}>
          Skip for now
        </Button>
        <Button onClick={() => onNext()}>Continue</Button>
      </div>
    </div>
  );
}

function ScrollableCardGrid({ children }: { children: React.ReactNode }) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const [canScrollDown, setCanScrollDown] = useState(false);

  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    const check = () => {
      setCanScrollDown(el.scrollHeight - el.scrollTop - el.clientHeight > 8);
    };
    check();
    el.addEventListener("scroll", check, { passive: true });
    const observer = new ResizeObserver(check);
    observer.observe(el);
    return () => { el.removeEventListener("scroll", check); observer.disconnect(); };
  }, []);

  return (
    <div className="relative">
      <div
        ref={scrollRef}
        className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4 max-h-[55vh] overflow-y-auto pr-1"
      >
        {children}
      </div>
      {canScrollDown && (
        <div className="absolute bottom-0 left-0 right-0 flex flex-col items-center pointer-events-none">
          <div className="w-full h-12 bg-gradient-to-t from-background to-transparent" />
          <ChevronDown className="h-4 w-4 text-muted-foreground animate-bounce -mt-6" />
        </div>
      )}
    </div>
  );
}

export { SESSION_KEY };
