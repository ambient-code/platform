/* eslint-disable @typescript-eslint/no-unused-vars */
import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { StreamMessage } from "../stream-message";
import type {
  ToolUseBlock,
  ToolResultBlock,
  ToolUseMessages,
  HierarchicalToolMessage,
  MessageObject,
} from "@/types/agentic-session";

// Mock child components to avoid deep rendering
vi.mock("@/components/ui/message", () => ({
  Message: ({ content, role, actions, feedbackButtons }: { content: string; role: string; actions?: React.ReactNode; feedbackButtons?: React.ReactNode }) => (
    <div data-testid="message" data-role={role}>
      {content}
      {actions}
      {feedbackButtons}
    </div>
  ),
  LoadingDots: () => <div data-testid="loading-dots" />,
}));

vi.mock("@/components/ui/tool-message", () => ({
  ToolMessage: ({ toolUseBlock, resultBlock }: { toolUseBlock?: ToolUseBlock; resultBlock?: ToolResultBlock }) => (
    <div data-testid="tool-message">
      {toolUseBlock?.name || "tool"}
      {resultBlock?.content ? ` result:${resultBlock.content}` : ""}
    </div>
  ),
}));

vi.mock("@/components/ui/thinking-message", () => ({
  ThinkingMessage: ({ block }: { block: { thinking: string } }) => (
    <div data-testid="thinking-message">{block.thinking}</div>
  ),
}));

vi.mock("@/components/ui/system-message", () => ({
  SystemMessage: ({ subtype }: { subtype?: string }) => (
    <div data-testid="system-message">{subtype}</div>
  ),
}));

vi.mock("@/components/feedback", () => ({
  FeedbackButtons: () => <div data-testid="feedback-buttons" />,
}));

describe("StreamMessage", () => {
  describe("tool use pairs", () => {
    it("renders ToolMessage for tool_use_messages", () => {
      const msg: ToolUseMessages = {
        type: "tool_use_messages",
        toolUseBlock: {
          type: "tool_use_block",
          id: "tu-1",
          name: "Read",
          input: {},
        },
        resultBlock: {
          type: "tool_result_block",
          tool_use_id: "tu-1",
          content: "file contents",
        },
        timestamp: "2025-01-01T00:00:00Z",
      };
      const { container } = render(<StreamMessage message={msg} />);
      expect(screen.getByTestId("tool-message")).toBeTruthy();
      expect(screen.getByText(/Read/)).toBeTruthy();
    });

    it("renders hierarchical tool message with children", () => {
      const msg: HierarchicalToolMessage = {
        type: "tool_use_messages",
        toolUseBlock: {
          type: "tool_use_block",
          id: "tu-1",
          name: "Agent",
          input: { subagent_type: "Helper" },
        },
        resultBlock: {
          type: "tool_result_block",
          tool_use_id: "tu-1",
          content: "done",
        },
        timestamp: "2025-01-01T00:00:00Z",
        children: [],
      };
      const { container } = render(<StreamMessage message={msg} />);
      expect(screen.getByTestId("tool-message")).toBeTruthy();
    });
  });

  describe("agent_running", () => {
    it("renders LoadingDots when isNewest", () => {
      const msg = { type: "agent_running" as const, timestamp: "2025-01-01T00:00:00Z" };
      const { container } = render(<StreamMessage message={msg} isNewest={true} />);
      expect(screen.getByTestId("loading-dots")).toBeTruthy();
    });

    it("renders nothing when not newest", () => {
      const msg = { type: "agent_running" as const, timestamp: "2025-01-01T00:00:00Z" };
      const { container } = render(<StreamMessage message={msg} isNewest={false} />);
      expect(container.innerHTML).toBe("");
    });
  });

  describe("agent_waiting", () => {
    it("renders a random agent message when isNewest", () => {
      const msg = { type: "agent_waiting" as const, timestamp: "2025-01-01T00:00:00Z" };
      const { container } = render(<StreamMessage message={msg} isNewest={true} />);
      // Should render one of the funny messages
      const span = document.querySelector(".text-muted-foreground");
      expect(span).toBeTruthy();
    });

    it("renders nothing when not newest", () => {
      const msg = { type: "agent_waiting" as const, timestamp: "2025-01-01T00:00:00Z" };
      const { container } = render(<StreamMessage message={msg} isNewest={false} />);
      expect(container.innerHTML).toBe("");
    });
  });

  describe("user_message", () => {
    it("renders string content as user message", () => {
      const msg: MessageObject = {
        type: "user_message",
        content: "Hello world",
        timestamp: "2025-01-01T00:00:00Z",
      };
      const { container } = render(<StreamMessage message={msg} />);
      const el = screen.getByTestId("message");
      expect(el.getAttribute("data-role")).toBe("user");
      expect(el.textContent).toBe("Hello world");
    });
  });

  describe("agent_message", () => {
    it("renders string content as bot message", () => {
      const msg = {
        type: "agent_message",
        content: "I can help with that",
        timestamp: "2025-01-01T00:00:00Z",
      } as unknown as MessageObject;
      const { container } = render(<StreamMessage message={msg} />);
      const el = screen.getByTestId("message");
      expect(el.getAttribute("data-role")).toBe("bot");
    });

    it("renders text_block content", () => {
      const msg: MessageObject = {
        type: "agent_message",
        content: { type: "text_block", text: "block text" },
        model: "test",
        timestamp: "2025-01-01T00:00:00Z",
      };
      const { container } = render(<StreamMessage message={msg} />);
      expect(screen.getByText("block text")).toBeTruthy();
    });

    it("renders reasoning_block as ThinkingMessage", () => {
      const msg: MessageObject = {
        type: "agent_message",
        content: { type: "reasoning_block", thinking: "deep thought", signature: "sig" },
        model: "test",
        timestamp: "2025-01-01T00:00:00Z",
      };
      const { container } = render(<StreamMessage message={msg} />);
      expect(screen.getByTestId("thinking-message")).toBeTruthy();
      expect(screen.getByText("deep thought")).toBeTruthy();
    });

    it("renders tool_use_block as ToolMessage", () => {
      const msg: MessageObject = {
        type: "agent_message",
        content: { type: "tool_use_block", id: "tu-1", name: "Write", input: {} },
        model: "test",
        timestamp: "2025-01-01T00:00:00Z",
      };
      const { container } = render(<StreamMessage message={msg} />);
      expect(screen.getByTestId("tool-message")).toBeTruthy();
    });

    it("renders tool_result_block as ToolMessage", () => {
      const msg: MessageObject = {
        type: "agent_message",
        content: { type: "tool_result_block", tool_use_id: "tu-1", content: "result" },
        model: "test",
        timestamp: "2025-01-01T00:00:00Z",
      };
      const { container } = render(<StreamMessage message={msg} />);
      expect(screen.getByTestId("tool-message")).toBeTruthy();
    });

    it("shows feedback buttons for non-streaming agent messages", () => {
      const msg = {
        type: "agent_message",
        id: "msg-1",
        content: "Complete response",
        timestamp: "2025-01-01T00:00:00Z",
      } as unknown as MessageObject;
      const { container } = render(<StreamMessage message={msg} />);
      expect(screen.getByTestId("feedback-buttons")).toBeTruthy();
    });
  });

  describe("system_message", () => {
    it("renders SystemMessage component", () => {
      const msg: MessageObject = {
        type: "system_message",
        subtype: "info",
        data: {},
        timestamp: "2025-01-01T00:00:00Z",
      };
      const { container } = render(<StreamMessage message={msg} />);
      expect(screen.getByTestId("system-message")).toBeTruthy();
    });
  });

  describe("result_message", () => {
    it("renders success result with Go to Results button", () => {
      const onGoToResults = vi.fn();
      const msg: MessageObject = {
        type: "result_message",
        is_error: false,
        duration_ms: 1000,
        duration_api_ms: 500,
        num_turns: 3,
        subtype: "success",
        session_id: "sess-1",
        timestamp: "2025-01-01T00:00:00Z",
      };
      const { container } = render(<StreamMessage message={msg} onGoToResults={onGoToResults} />);
      expect(screen.getByText("Agent completed successfully.")).toBeTruthy();
      expect(screen.getByText("Go to Results")).toBeTruthy();
    });

    it("renders error result message", () => {
      const msg: MessageObject = {
        type: "result_message",
        is_error: true,
        duration_ms: 500,
        duration_api_ms: 200,
        num_turns: 1,
        subtype: "error",
        session_id: "sess-1",
        timestamp: "2025-01-01T00:00:00Z",
      };
      const { container } = render(<StreamMessage message={msg} />);
      expect(screen.getByText("Agent completed with errors.")).toBeTruthy();
    });
  });

  describe("unknown message type", () => {
    it("returns null for unknown types", () => {
      const msg = { type: "unknown_type", timestamp: "2025-01-01T00:00:00Z" } as unknown as MessageObject;
      const { container } = render(<StreamMessage message={msg} />);
      expect(container.innerHTML).toBe("");
    });
  });
});
