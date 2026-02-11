"use client";

import { useContext } from "react";
import type { UserMessageProps } from "@copilotkit/react-ui";
import { MessageTimestampsCtx } from "./session-contexts";
import { formatMessageTime } from "./format-message-time";

/**
 * Custom UserMessage that renders the user's text inside a styled
 * bubble with a timestamp underneath, matching the assistant timestamp
 * style on the opposite side.
 */
export function SessionAwareUserMessage({ message }: UserMessageProps) {
  const timestamps = useContext(MessageTimestampsCtx);
  const ts = message?.id ? timestamps[message.id] : undefined;

  const content =
    typeof message?.content === "string"
      ? message.content
      : Array.isArray(message?.content)
        ? message.content
            .filter((c): c is { type: "text"; text: string } => c.type === "text")
            .map((c) => c.text)
            .join("\n")
        : "";

  if (!content) return null;

  return (
    <>
      <div className="copilotKitMessage copilotKitUserMessage">
        {content}
      </div>
      {ts != null && (
        <span className="mt-1 block text-[10px] leading-none text-muted-foreground/60 text-right select-none">
          {formatMessageTime(ts)}
        </span>
      )}
    </>
  );
}
