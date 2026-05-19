"use client";

import { useMemo } from "react";
import type { AppConfig } from "./integration-registry";

/**
 * Reads server-side configuration values from <meta> tags in the document head.
 * Uses the same pattern as the existing `backend-ws-base` meta tag.
 *
 * Values are set in `app/layout.tsx` (server component) and read here on the
 * client so that any client component can access them without prop drilling.
 */
export function useAppConfig(): AppConfig {
  return useMemo(() => {
    if (typeof document === "undefined") {
      return { githubAppSlug: "", githubCallbackUrl: "" };
    }

    const readMeta = (name: string): string =>
      document.querySelector(`meta[name="${name}"]`)?.getAttribute("content") ?? "";

    return {
      githubAppSlug: readMeta("github-app-slug"),
      githubCallbackUrl: readMeta("github-callback-url"),
    };
  }, []);
}
