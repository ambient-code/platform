"use client";

import Link from "next/link";
import { CheckCircle2, AlertTriangle } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { CardSkeleton } from "./card-skeleton";
import { useIntegrationsStatus } from "@/services/queries/use-integrations";

export function IntegrationsPanel() {
  const { data: integrationsStatus, isPending } = useIntegrationsStatus();

  const githubConfigured = integrationsStatus?.github?.active != null;
  const gitlabConfigured = integrationsStatus?.gitlab?.connected ?? false;
  const jiraConfigured = integrationsStatus?.jira?.connected ?? false;
  const googleConfigured = integrationsStatus?.google?.connected ?? false;

  const integrations = [
    {
      key: "github",
      name: "GitHub",
      configured: githubConfigured,
      configuredMessage:
        "Authenticated. Git push and repository access enabled.",
    },
    {
      key: "gitlab",
      name: "GitLab",
      configured: gitlabConfigured,
      configuredMessage:
        "Authenticated. Git push and repository access enabled.",
    },
    {
      key: "google",
      name: "Google Workspace",
      configured: googleConfigured,
      configuredMessage:
        "Authenticated. Drive, Calendar, and Gmail access enabled.",
    },
    {
      key: "jira",
      name: "Jira",
      configured: jiraConfigured,
      configuredMessage: "Authenticated. Issue and project access enabled.",
    },
  ].sort((a, b) => a.name.localeCompare(b.name));

  const configuredCount = integrations.filter((i) => i.configured).length;

  return (
    <div>
      <div className="flex items-center justify-between mb-1">
        <h3 className="text-lg font-semibold">Integrations</h3>
        <Badge variant="outline" className="text-xs">
          {isPending ? "—" : `${configuredCount}/${integrations.length}`}
        </Badge>
      </div>
      <p className="text-sm text-muted-foreground mb-4">
        External services connected to the platform.
      </p>

      <div className="space-y-2">
        {isPending ? (
          <>
            <CardSkeleton />
            <CardSkeleton />
            <CardSkeleton />
          </>
        ) : (
          integrations.map((integration) => (
            <IntegrationCard key={integration.key} integration={integration} />
          ))
        )}
      </div>
    </div>
  );
}

function IntegrationCard({
  integration,
}: {
  integration: {
    key: string;
    name: string;
    configured: boolean;
    configuredMessage: string;
  };
}) {
  return (
    <div className="flex items-start justify-between gap-3 p-3 border rounded-lg bg-background/50">
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <div className="flex-shrink-0">
            {integration.configured ? (
              <CheckCircle2 className="h-4 w-4 text-green-600" />
            ) : (
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="inline-flex">
                      <AlertTriangle className="h-4 w-4 text-amber-500" />
                    </span>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>Not configured</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            )}
          </div>
          <h4 className="font-medium text-sm">{integration.name}</h4>
        </div>
        <p className="text-xs text-muted-foreground mt-0.5">
          {integration.configured ? (
            integration.configuredMessage
          ) : (
            <>
              Not connected.{" "}
              <Link href="/integrations" className="text-primary hover:underline">
                Set up
              </Link>{" "}
              to enable {integration.name} access.
            </>
          )}
        </p>
      </div>
    </div>
  );
}

