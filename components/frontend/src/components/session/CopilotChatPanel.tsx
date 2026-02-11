"use client";

import React from "react";
import "@copilotkit/react-ui/styles.css";
import { CopilotKit } from "@copilotkit/react-core";
import { CopilotChat } from "@copilotkit/react-ui";

type CopilotChatPanelProps = {
  projectName: string;
  sessionName: string;
  className?: string;
};

/**
 * CopilotKit-powered chat panel for session interaction.
 *
 * Replaces the custom useAGUIStream + MessagesTab combo with
 * CopilotKit's built-in chat component. The CopilotRuntime on the
 * server (Next.js API route) proxies to the backend's AG-UI runner.
 */
export function CopilotChatPanel({
  projectName,
  sessionName,
  className = "",
}: CopilotChatPanelProps) {
  const runtimeUrl = `/api/copilotkit/${projectName}/${sessionName}`;

  return (
    <CopilotKit
      runtimeUrl={runtimeUrl}
      showDevConsole={false}
      agent="session"
    >
      <ChatContent className={className} />
    </CopilotKit>
  );
}

function ChatContent({ className }: { className: string }) {
  return (
    <div className={`flex flex-col h-full ${className}`}>
      <CopilotChat
        className="h-full flex-1"
        labels={{
          initial: "Session ready. How can I help you?",
          placeholder: "Send a message...",
        }}
      />
    </div>
  );
}
