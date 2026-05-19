"use client";

import { useState, useCallback } from "react";
import { useProjectsPaginated } from "@/services/queries/use-projects";

const ONBOARDING_FLAG = "acp-onboarding-complete";

/**
 * Determines whether the onboarding wizard should be displayed.
 *
 * Shows the wizard when BOTH conditions are true:
 * - The user has zero workspaces (dedicated lightweight query, limit=1).
 * - The localStorage flag `acp-onboarding-complete` is not set.
 *
 * To change trigger logic (e.g. add feature flag, role check), modify only
 * this file. No other component references localStorage or project counts
 * for onboarding purposes.
 */
export function useShouldShowOnboarding(): {
  shouldShow: boolean;
  isLoading: boolean;
  dismiss: () => void;
} {
  const { data, isLoading } = useProjectsPaginated({ limit: 1 });

  const [dismissed, setDismissed] = useState(() => {
    if (typeof window === "undefined") return false;
    return localStorage.getItem(ONBOARDING_FLAG) === "true";
  });

  const hasProjects = (data?.totalCount ?? 0) > 0;
  const shouldShow = !isLoading && !hasProjects && !dismissed;

  const dismiss = useCallback(() => {
    setDismissed(true);
    try {
      localStorage.setItem(ONBOARDING_FLAG, "true");
    } catch {
      // localStorage may be unavailable in some environments
    }
  }, []);

  return { shouldShow, isLoading, dismiss };
}

export { ONBOARDING_FLAG };
