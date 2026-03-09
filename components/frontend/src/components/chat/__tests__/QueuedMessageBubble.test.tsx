import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { QueuedMessageBubble } from "../QueuedMessageBubble";
import type { QueuedMessageItem } from "@/hooks/use-session-queue";

function makeMessage(overrides: Partial<QueuedMessageItem> = {}): QueuedMessageItem {
  return {
    id: "msg-1",
    content: "Hello from queue",
    timestamp: Date.now() - 30_000, // 30 seconds ago
    ...overrides,
  };
}

describe("QueuedMessageBubble", () => {
  it("renders queued message content", () => {
    render(<QueuedMessageBubble message={makeMessage()} onCancel={vi.fn()} />);
    expect(screen.getByText("Hello from queue")).toBeTruthy();
  });

  it("shows Queued label", () => {
    render(<QueuedMessageBubble message={makeMessage()} onCancel={vi.fn()} />);
    expect(screen.getByText("Queued")).toBeTruthy();
  });

  it("shows user avatar", () => {
    render(<QueuedMessageBubble message={makeMessage()} onCancel={vi.fn()} />);
    expect(screen.getByText("You")).toBeTruthy();
  });

  it("shows 'just now' for recent messages", () => {
    render(
      <QueuedMessageBubble
        message={makeMessage({ timestamp: Date.now() - 5_000 })}
        onCancel={vi.fn()}
      />
    );
    expect(screen.getByText("just now")).toBeTruthy();
  });

  it("shows minutes ago for older messages", () => {
    render(
      <QueuedMessageBubble
        message={makeMessage({ timestamp: Date.now() - 120_000 })}
        onCancel={vi.fn()}
      />
    );
    expect(screen.getByText("2m ago")).toBeTruthy();
  });

  it("shows hours ago for much older messages", () => {
    render(
      <QueuedMessageBubble
        message={makeMessage({ timestamp: Date.now() - 7_200_000 })}
        onCancel={vi.fn()}
      />
    );
    expect(screen.getByText("2h ago")).toBeTruthy();
  });

  it("calls onCancel with message id when cancel button is clicked", () => {
    const onCancel = vi.fn();
    render(<QueuedMessageBubble message={makeMessage({ id: "msg-42" })} onCancel={onCancel} />);
    fireEvent.click(screen.getByText("Cancel"));
    expect(onCancel).toHaveBeenCalledWith("msg-42");
  });

  it("preserves whitespace in message content", () => {
    render(
      <QueuedMessageBubble
        message={makeMessage({ content: "line1\nline2" })}
        onCancel={vi.fn()}
      />
    );
    // The whitespace-pre-wrap class is set on the <p>, check both lines appear
    expect(screen.getByText(/line1/)).toBeTruthy();
    expect(screen.getByText(/line2/)).toBeTruthy();
  });
});
