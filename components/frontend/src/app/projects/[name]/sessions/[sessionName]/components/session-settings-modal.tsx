"use client";

import { useState } from "react";
import {
  Info,
  Plug,
  Link2,
} from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useMcpStatus } from "@/services/queries/use-mcp";
import { useIntegrationsStatus } from "@/services/queries/use-integrations";
import type { AgenticSession } from "@/types/agentic-session";

import { SessionDetails } from "./settings/session-details";
import { McpServersPanel } from "./settings/mcp-servers-panel";
import { IntegrationsPanel } from "./settings/integrations-panel";

type SettingsTab = "session" | "mcp" | "integrations";

type SessionSettingsModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  session: AgenticSession;
  projectName: string;
  onEditName?: () => void;
};

export function SessionSettingsModal({
  open,
  onOpenChange,
  session,
  projectName,
  onEditName,
}: SessionSettingsModalProps) {
  const [activeTab, setActiveTab] = useState<SettingsTab>("session");

  const phase = session.status?.phase || "Pending";
  const isRunning = phase === "Running";

  const { data: mcpStatus } = useMcpStatus(
    projectName,
    session.metadata.name,
    isRunning
  );
  const mcpCount = mcpStatus?.servers?.length ?? 0;

  const { data: integrationsStatus } = useIntegrationsStatus();
  const integrationsTotal = 4;
  const integrationsConnected = [
    integrationsStatus?.github?.active != null,
    integrationsStatus?.gitlab?.connected,
    integrationsStatus?.jira?.connected,
    integrationsStatus?.google?.connected,
  ].filter(Boolean).length;

  const tabs: {
    id: SettingsTab;
    label: string;
    icon: typeof Info;
    badge?: string;
  }[] = [
    { id: "session", label: "Session", icon: Info },
    {
      id: "mcp",
      label: "MCP Servers",
      icon: Plug,
      badge: mcpCount > 0 ? String(mcpCount) : undefined,
    },
    {
      id: "integrations",
      label: "Integrations",
      icon: Link2,
      badge: `${integrationsConnected}/${integrationsTotal}`,
    },
  ];

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-5xl p-0 gap-0">
        <DialogHeader className="px-6 py-4 border-b">
          <DialogTitle>Session Settings</DialogTitle>
        </DialogHeader>

        <div className="flex flex-col md:flex-row h-auto md:h-[600px] max-h-[80vh]">
          {/* Sidebar nav */}
          <nav className="flex md:flex-col md:w-48 border-b md:border-b-0 md:border-r p-2 gap-1 flex-shrink-0 overflow-x-auto">
            {tabs.map((tab) => {
              const Icon = tab.icon;
              return (
                <Button
                  key={tab.id}
                  variant="ghost"
                  className={cn(
                    "w-auto md:w-full justify-center md:justify-start gap-2 font-normal",
                    activeTab === tab.id && "bg-accent font-medium"
                  )}
                  onClick={() => setActiveTab(tab.id)}
                >
                  <Icon className="h-4 w-4 flex-shrink-0" />
                  <span className="truncate">{tab.label}</span>
                  {tab.badge && (
                    <Badge
                      variant="outline"
                      className="ml-auto text-[10px] px-1.5 py-0"
                    >
                      {tab.badge}
                    </Badge>
                  )}
                </Button>
              );
            })}
          </nav>

          {/* Tab content */}
          <div className="flex-1 p-6 overflow-y-auto">
            {activeTab === "session" && (
              <SessionDetails session={session} onEditName={onEditName} />
            )}
            {activeTab === "mcp" && (
              <McpServersPanel
                projectName={projectName}
                sessionName={session.metadata.name}
                sessionPhase={phase}
                session={session}
              />
            )}
            {activeTab === "integrations" && <IntegrationsPanel />}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
