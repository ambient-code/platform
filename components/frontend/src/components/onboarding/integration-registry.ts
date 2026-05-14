"use client";

import React from "react";
import type { IntegrationsStatus } from "@/services/api/integrations";
import { GitHubConnectionCard } from "@/components/github-connection-card";
import { GitLabConnectionCard } from "@/components/gitlab-connection-card";
import { GoogleDriveConnectionCard } from "@/components/google-drive-connection-card";
import { JiraConnectionCard } from "@/components/jira-connection-card";
import { CodeRabbitConnectionCard } from "@/components/coderabbit-connection-card";
import { GerritConnectionCard } from "@/components/gerrit-connection-card";
import { useAppConfig } from "./use-app-config";

/**
 * Describes a single platform integration for dynamic rendering.
 *
 * To add a new integration:
 * 1. Add its status field to `IntegrationsStatus` in `services/api/integrations.ts`.
 * 2. Add an `IntegrationEntry` here with a matching `id`.
 * 3. The onboarding wizard and any other registry consumer picks it up automatically.
 */
export type IntegrationEntry = {
  /** Must match a key of IntegrationsStatus (excluding mcpServers). */
  id: IntegrationStatusKey;
  name: string;
  description: string;
  /** Derive a boolean "connected" signal from the heterogeneous status shape. */
  isConnected: (status: IntegrationsStatus) => boolean;
  /** Render the integration's connection card. */
  renderCard: (props: {
    status: IntegrationsStatus;
    onRefresh: () => void;
    appConfig: AppConfig;
  }) => React.ReactNode;
};

/** All IntegrationsStatus keys that represent real integrations (not mcpServers). */
type IntegrationStatusKey = Exclude<keyof IntegrationsStatus, "mcpServers">;

/** Server-side config values exposed via meta tags. */
export type AppConfig = {
  githubAppSlug: string;
  githubCallbackUrl: string;
};

/**
 * Compile-time completeness guard: ensures every integration status key has a
 * registry entry. If a new key is added to IntegrationsStatus without a
 * corresponding entry here, TypeScript will report an error.
 */
type AssertComplete<T extends readonly IntegrationEntry[]> =
  IntegrationStatusKey extends T[number]["id"] ? T : never;

const _REGISTRY = [
  {
    id: "github" as const,
    name: "GitHub",
    description: "Connect repositories and pull requests",
    isConnected: (status: IntegrationsStatus) =>
      status.github?.installed || status.github?.pat?.configured || false,
    renderCard: ({ status, onRefresh, appConfig }) =>
      React.createElement(GitHubConnectionCard, {
        appSlug: appConfig.githubAppSlug || undefined,
        githubCallbackUrl: appConfig.githubCallbackUrl || undefined,
        showManageButton: true,
        status: status.github,
        onRefresh,
      }),
  },
  {
    id: "gitlab" as const,
    name: "GitLab",
    description: "Connect GitLab repositories and merge requests",
    isConnected: (status: IntegrationsStatus) =>
      status.gitlab?.connected || false,
    renderCard: ({ status, onRefresh }) =>
      React.createElement(GitLabConnectionCard, {
        status: status.gitlab,
        onRefresh,
      }),
  },
  {
    id: "google" as const,
    name: "Google Drive",
    description: "Access documents and files from Google Drive",
    isConnected: (status: IntegrationsStatus) =>
      status.google?.connected || false,
    renderCard: ({ status, onRefresh }) =>
      React.createElement(GoogleDriveConnectionCard, {
        showManageButton: true,
        status: status.google,
        onRefresh,
      }),
  },
  {
    id: "jira" as const,
    name: "Jira",
    description: "Link issues and track work from Jira",
    isConnected: (status: IntegrationsStatus) =>
      status.jira?.connected || false,
    renderCard: ({ status, onRefresh }) =>
      React.createElement(JiraConnectionCard, {
        status: status.jira,
        onRefresh,
      }),
  },
  {
    id: "coderabbit" as const,
    name: "CodeRabbit",
    description: "AI-powered code review integration",
    isConnected: (status: IntegrationsStatus) =>
      status.coderabbit?.connected || false,
    renderCard: ({ status, onRefresh }) =>
      React.createElement(CodeRabbitConnectionCard, {
        status: status.coderabbit,
        onRefresh,
      }),
  },
  {
    id: "gerrit" as const,
    name: "Gerrit",
    description: "Connect Gerrit instances for code review",
    isConnected: (status: IntegrationsStatus) =>
      (status.gerrit?.instances ?? []).some((i) => i.connected),
    renderCard: ({ onRefresh }) =>
      React.createElement(GerritConnectionCard, { onRefresh }),
  },
] as const satisfies readonly IntegrationEntry[];

// This line triggers a type error if _REGISTRY is missing any IntegrationStatusKey.
export const INTEGRATION_REGISTRY: AssertComplete<typeof _REGISTRY> = _REGISTRY;

export { useAppConfig };
