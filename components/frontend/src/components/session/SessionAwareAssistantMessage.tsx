"use client";

import { useContext, useState } from "react";
import { Markdown, useChatContext } from "@copilotkit/react-ui";
import type { AssistantMessageProps } from "@copilotkit/react-ui";
import { SessionActiveCtx, PersistedFeedbackCtx, MessageTimestampsCtx } from "./session-contexts";
import { formatMessageTime } from "./format-message-time";

/**
 * Custom AssistantMessage that:
 *   1. Omits the regenerate / rerun button entirely
 *   2. Disables thumbs up/down when the session is not running
 *      (buttons remain visible so users can see prior feedback state)
 *   3. Restores thumbs highlight from ambient:feedback RAW events
 *      in the AG-UI stream (CopilotKit's internal state resets on mount)
 */
export function SessionAwareAssistantMessage(props: AssistantMessageProps) {
  const isSessionActive = useContext(SessionActiveCtx);
  const feedbackMap = useContext(PersistedFeedbackCtx);
  const timestamps = useContext(MessageTimestampsCtx);
  const { icons, labels } = useChatContext();
  const {
    message,
    isLoading,
    onCopy,
    onThumbsUp,
    onThumbsDown,
    isCurrentMessage,
    feedback: copilotFeedback,
    markdownTagRenderers,
  } = props;
  const [copied, setCopied] = useState(false);

  // Use CopilotKit's live feedback if set, otherwise fall back to
  // the map derived from ambient:feedback RAW events in the stream.
  const feedback =
    copilotFeedback ?? (message?.id ? feedbackMap[message.id] : undefined);

  const handleCopy = () => {
    const content = message?.content || "";
    if (content) {
      navigator.clipboard.writeText(content);
      setCopied(true);
      if (onCopy) onCopy(content);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const handleThumbsUp = () => {
    if (isSessionActive && onThumbsUp && message) onThumbsUp(message);
  };

  const handleThumbsDown = () => {
    if (isSessionActive && onThumbsDown && message) onThumbsDown(message);
  };

  const content = message?.content || "";
  const ts = message?.id ? timestamps[message.id] : undefined;
  const subComponent = message?.generativeUI?.() ?? props.subComponent;
  const subComponentPosition = message?.generativeUIPosition ?? "after";
  const renderBefore = subComponent && subComponentPosition === "before";
  const renderAfter = subComponent && subComponentPosition !== "before";

  const disabledStyle = !isSessionActive
    ? { opacity: 0.5, cursor: "default" as const, pointerEvents: "none" as const }
    : undefined;

  return (
    <>
      {renderBefore ? <div style={{ marginBottom: "0.5rem" }}>{subComponent}</div> : null}
      {content && (
        <div className="copilotKitMessage copilotKitAssistantMessage">
          <Markdown content={content} components={markdownTagRenderers} />

          {!isLoading && (
            <div
              className={`copilotKitMessageControls ${isCurrentMessage ? "currentMessage" : ""}`}
            >
              {ts != null && (
                <span className="text-[10px] leading-none text-muted-foreground/60 select-none mr-auto self-center">
                  {formatMessageTime(ts)}
                </span>
              )}
              {/* Regenerate / rerun button intentionally omitted */}
              <button
                className="copilotKitMessageControlButton"
                onClick={handleCopy}
                aria-label={labels.copyToClipboard}
                title={labels.copyToClipboard}
              >
                {copied ? (
                  <span style={{ fontSize: "10px", fontWeight: "bold" }}>✓</span>
                ) : (
                  icons.copyIcon
                )}
              </button>
              {onThumbsUp && (
                <button
                  className={`copilotKitMessageControlButton ${
                    feedback === "thumbsUp" ? "active" : ""
                  }`}
                  onClick={handleThumbsUp}
                  aria-label={labels.thumbsUp}
                  title={labels.thumbsUp}
                  disabled={!isSessionActive}
                  style={disabledStyle}
                >
                  {icons.thumbsUpIcon}
                </button>
              )}
              {onThumbsDown && (
                <button
                  className={`copilotKitMessageControlButton ${
                    feedback === "thumbsDown" ? "active" : ""
                  }`}
                  onClick={handleThumbsDown}
                  aria-label={labels.thumbsDown}
                  title={labels.thumbsDown}
                  disabled={!isSessionActive}
                  style={disabledStyle}
                >
                  {icons.thumbsDownIcon}
                </button>
              )}
            </div>
          )}
        </div>
      )}
      {renderAfter ? (
        <div style={{ marginTop: "1.75rem", marginBottom: "0.5rem" }}>{subComponent}</div>
      ) : null}
      {isLoading && <span>{icons.activityIcon}</span>}
    </>
  );
}
