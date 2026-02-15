"use client";

import React, { useCallback, useContext, useMemo, useRef, useState, useEffect } from "react";
import { useChatContext } from "@copilotkit/react-ui";
import type { InputProps } from "@copilotkit/react-ui";
import { Loader2, Play } from "lucide-react";
import { SessionActiveCtx, ResumeCtx, WorkflowMetadataCtx } from "./session-contexts";
import { AutocompletePopup, type AutocompleteItem } from "./AutocompletePopup";

/**
 * Custom Input — disabled when session is not running.
 *
 * When the session is active: standard textarea + send/stop button.
 * When inactive: disabled textarea with explanatory placeholder.
 * Supports `/` slash-command and `@` agent autocomplete when workflow
 * metadata is available via WorkflowMetadataCtx.
 */
export function SessionAwareInput({
  inProgress,
  onSend,
  onStop,
  hideStopButton = false,
  chatReady = false,
}: InputProps) {
  const isSessionActive = useContext(SessionActiveCtx);
  const { onResume, isResuming } = useContext(ResumeCtx);
  const context = useChatContext();
  const { commands, agents } = useContext(WorkflowMetadataCtx);
  const [text, setText] = useState("");

  // ── Resume banner flag for hibernated sessions ─────────────────────
  const showResumeBanner = !isSessionActive && !!onResume;
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const autocompleteRef = useRef<HTMLDivElement>(null);

  // Autocomplete state
  const [acOpen, setAcOpen] = useState(false);
  const [acType, setAcType] = useState<"command" | "agent" | null>(null);
  const [acFilter, setAcFilter] = useState("");
  const [acTriggerPos, setAcTriggerPos] = useState(0);
  const [acSelected, setAcSelected] = useState(0);

  const effectivelyDisabled = !isSessionActive || !chatReady;

  const hasAutocompleteData = commands.length > 0 || agents.length > 0;

  // ── Filtered autocomplete items ────────────────────────────────────
  const acItems = useMemo<AutocompleteItem[]>(() => {
    if (!acOpen || !acType) return [];
    const filter = acFilter.toLowerCase();

    if (acType === "command") {
      return commands
        .filter(
          (cmd) =>
            cmd.name.toLowerCase().includes(filter) ||
            cmd.slashCommand.toLowerCase().includes(filter),
        )
        .map((cmd) => ({
          id: cmd.id,
          label: cmd.slashCommand,
          detail: cmd.name,
          insertText: `${cmd.slashCommand} `,
        }));
    }

    return agents
      .filter((a) => a.name.toLowerCase().includes(filter))
      .map((a) => {
        const short = a.name.split(" - ")[0];
        return {
          id: a.id,
          label: `@${short}`,
          detail: a.name,
          insertText: `@${short} `,
        };
      });
  }, [acOpen, acType, acFilter, commands, agents]);

  // ── Close autocomplete ─────────────────────────────────────────────
  const closeAutocomplete = useCallback(() => {
    setAcOpen(false);
    setAcType(null);
    setAcFilter("");
    setAcSelected(0);
  }, []);

  // ── Click-outside dismissal ────────────────────────────────────────
  useEffect(() => {
    if (!acOpen) return;
    const handler = (e: MouseEvent) => {
      if (
        autocompleteRef.current &&
        !autocompleteRef.current.contains(e.target as Node) &&
        textareaRef.current &&
        !textareaRef.current.contains(e.target as Node)
      ) {
        closeAutocomplete();
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [acOpen, closeAutocomplete]);

  // ── Handle selection ───────────────────────────────────────────────
  const handleAcSelect = useCallback(
    (item: AutocompleteItem) => {
      if (!textareaRef.current) return;
      const cursorPos = textareaRef.current.selectionStart;
      const before = text.substring(0, acTriggerPos);
      const after = text.substring(cursorPos);
      const newText = before + item.insertText + after;
      setText(newText);
      closeAutocomplete();

      // Restore cursor after the inserted text
      requestAnimationFrame(() => {
        if (textareaRef.current) {
          const pos = before.length + item.insertText.length;
          textareaRef.current.selectionStart = pos;
          textareaRef.current.selectionEnd = pos;
          textareaRef.current.focus();
        }
      });
    },
    [text, acTriggerPos, closeAutocomplete],
  );

  // ── Input change handler with trigger detection ────────────────────
  const handleInputChange = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      const newValue = e.target.value;
      const cursorPos = e.target.selectionStart;
      setText(newValue);

      if (!hasAutocompleteData) return;

      if (cursorPos > 0) {
        const charBefore = newValue[cursorPos - 1];
        const textBefore = newValue.substring(0, cursorPos);

        // Check for trigger characters
        if (charBefore === "/" || charBefore === "@") {
          const isAtStart = cursorPos === 1;
          const afterWhitespace =
            cursorPos >= 2 && /\s/.test(newValue[cursorPos - 2]);
          if (isAtStart || afterWhitespace) {
            setAcTriggerPos(cursorPos - 1);
            setAcType(charBefore === "@" ? "agent" : "command");
            setAcFilter("");
            setAcSelected(0);
            setAcOpen(true);
            return;
          }
        }

        // Update filter while autocomplete is open
        if (acOpen) {
          const filterText = textBefore.substring(acTriggerPos + 1);
          if (cursorPos <= acTriggerPos || /\s/.test(filterText)) {
            closeAutocomplete();
          } else {
            setAcFilter(filterText);
            setAcSelected(0);
          }
        }
      } else if (acOpen) {
        closeAutocomplete();
      }
    },
    [hasAutocompleteData, acOpen, acTriggerPos, closeAutocomplete],
  );

  // ── Send message ───────────────────────────────────────────────────
  const send = useCallback(() => {
    if (inProgress || effectivelyDisabled) return;
    if (text.trim().length === 0) return;
    onSend(text);
    setText("");
    closeAutocomplete();
    textareaRef.current?.focus();
  }, [inProgress, effectivelyDisabled, text, onSend, closeAutocomplete]);

  const canSend = !effectivelyDisabled && !inProgress && text.trim().length > 0;
  const canStop = !effectivelyDisabled && inProgress && !hideStopButton;
  const sendDisabled = !canSend && !canStop;

  const { buttonIcon, buttonAlt } = useMemo(() => {
    if (!chatReady)
      return { buttonIcon: context.icons.spinnerIcon, buttonAlt: "Loading" };
    if (!isSessionActive)
      return { buttonIcon: context.icons.sendIcon, buttonAlt: "Session not running" };
    return inProgress && !hideStopButton
      ? { buttonIcon: context.icons.stopIcon, buttonAlt: "Stop" }
      : { buttonIcon: context.icons.sendIcon, buttonAlt: "Send" };
  }, [inProgress, chatReady, isSessionActive, hideStopButton, context.icons]);

  const placeholder = !chatReady
    ? "Loading..."
    : !isSessionActive
      ? "Session is not running"
      : hasAutocompleteData
        ? "Send a message... (type / or @ for suggestions)"
        : context.labels.placeholder;

  // ── Keyboard handler ───────────────────────────────────────────────
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (acOpen && acItems.length > 0) {
        if (e.key === "ArrowDown") {
          e.preventDefault();
          setAcSelected((prev) => (prev + 1) % acItems.length);
          return;
        }
        if (e.key === "ArrowUp") {
          e.preventDefault();
          setAcSelected((prev) => (prev - 1 + acItems.length) % acItems.length);
          return;
        }
        if (e.key === "Enter" || e.key === "Tab") {
          e.preventDefault();
          handleAcSelect(acItems[acSelected]);
          return;
        }
        if (e.key === "Escape") {
          e.preventDefault();
          closeAutocomplete();
          return;
        }
      }

      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        if (canSend) send();
      }
    },
    [acOpen, acItems, acSelected, handleAcSelect, closeAutocomplete, canSend, send],
  );

  if (showResumeBanner) {
    return (
      <div className="copilotKitInputContainer">
        <button
          type="button"
          onClick={onResume}
          disabled={isResuming}
          className="w-full flex items-center justify-center gap-2 py-3 px-4 rounded-lg border border-border bg-muted/50 hover:bg-muted text-sm font-medium transition-colors disabled:opacity-60"
        >
          {isResuming ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" />
              Resuming session...
            </>
          ) : (
            <>
              <Play className="h-4 w-4" />
              Resume session to continue chatting
            </>
          )}
        </button>
      </div>
    );
  }

  return (
    <div className="copilotKitInputContainer">
      <div className="copilotKitInput" style={{ position: "relative" }}>
        {/* Autocomplete popup */}
        {acOpen && (
          <AutocompletePopup
            items={acItems}
            selectedIndex={acSelected}
            onSelect={handleAcSelect}
            onHover={setAcSelected}
            popupRef={autocompleteRef}
          />
        )}

        <textarea
          ref={textareaRef}
          placeholder={placeholder}
          disabled={effectivelyDisabled}
          autoFocus={false}
          value={text}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          rows={1}
          style={{ resize: "none" }}
        />
        <div className="copilotKitInputControls">
          <div style={{ flexGrow: 1 }} />
          <button
            disabled={sendDisabled}
            onClick={canStop ? onStop : send}
            data-copilotkit-in-progress={inProgress}
            className="copilotKitInputControlButton"
            aria-label={buttonAlt}
          >
            {buttonIcon}
          </button>
        </div>
      </div>
    </div>
  );
}
