import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { FeedbackModal } from "../FeedbackModal";

// Mock the FeedbackContext
const mockFeedbackContext = {
  projectName: "test-project",
  sessionName: "test-session",
  username: "testuser",
  initialPrompt: "initial prompt",
  activeWorkflow: undefined,
  messages: [],
  traceId: "trace-123",
};

vi.mock("@/contexts/FeedbackContext", () => ({
  useFeedbackContextOptional: () => mockFeedbackContext,
}));

// Mock fetch
const mockFetch = vi.fn();
global.fetch = mockFetch;

describe("FeedbackModal", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockFetch.mockResolvedValue({ ok: true });
  });

  it("renders positive feedback dialog", () => {
    render(
      <FeedbackModal
        open={true}
        onOpenChange={vi.fn()}
        feedbackType="positive"
      />
    );
    expect(screen.getByText("Share feedback")).toBeTruthy();
    expect(screen.getByText("Help us improve by sharing what went well.")).toBeTruthy();
  });

  it("renders negative feedback dialog", () => {
    render(
      <FeedbackModal
        open={true}
        onOpenChange={vi.fn()}
        feedbackType="negative"
      />
    );
    expect(screen.getByText("Help us improve by sharing what went wrong.")).toBeTruthy();
  });

  it("does not render when open is false", () => {
    render(
      <FeedbackModal
        open={false}
        onOpenChange={vi.fn()}
        feedbackType="positive"
      />
    );
    expect(screen.queryByText("Share feedback")).toBeNull();
  });

  it("shows comment textarea", () => {
    render(
      <FeedbackModal
        open={true}
        onOpenChange={vi.fn()}
        feedbackType="positive"
      />
    );
    expect(screen.getByLabelText("Additional comments (optional)")).toBeTruthy();
  });

  it("allows typing in comment textarea", () => {
    render(
      <FeedbackModal
        open={true}
        onOpenChange={vi.fn()}
        feedbackType="positive"
      />
    );
    const textarea = screen.getByPlaceholderText("What was good about this response?");
    fireEvent.change(textarea, { target: { value: "Great response!" } });
    expect((textarea as HTMLTextAreaElement).value).toBe("Great response!");
  });

  it("shows negative placeholder for negative feedback", () => {
    render(
      <FeedbackModal
        open={true}
        onOpenChange={vi.fn()}
        feedbackType="negative"
      />
    );
    expect(screen.getByPlaceholderText("What could be improved about this response?")).toBeTruthy();
  });

  it("calls onOpenChange(false) when cancel is clicked", () => {
    const onOpenChange = vi.fn();
    render(
      <FeedbackModal
        open={true}
        onOpenChange={onOpenChange}
        feedbackType="positive"
      />
    );
    fireEvent.click(screen.getByText("Cancel"));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("submits feedback and closes on success", async () => {
    const onOpenChange = vi.fn();
    const onSubmitSuccess = vi.fn();
    render(
      <FeedbackModal
        open={true}
        onOpenChange={onOpenChange}
        feedbackType="positive"
        messageId="msg-1"
        onSubmitSuccess={onSubmitSuccess}
      />
    );

    fireEvent.click(screen.getByText("Send feedback"));

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    // Check fetch was called with correct URL
    const [url, opts] = mockFetch.mock.calls[0];
    expect(url).toContain("/test-project/");
    expect(url).toContain("/test-session/");
    expect(url).toContain("/agui/feedback");

    const body = JSON.parse(opts.body);
    expect(body.metaType).toBe("thumbs_up");
    expect(body.payload.messageId).toBe("msg-1");
    expect(body.payload.userId).toBe("testuser");

    await waitFor(() => {
      expect(onOpenChange).toHaveBeenCalledWith(false);
      expect(onSubmitSuccess).toHaveBeenCalled();
    });
  });

  it("submits thumbs_down for negative feedback", async () => {
    render(
      <FeedbackModal
        open={true}
        onOpenChange={vi.fn()}
        feedbackType="negative"
      />
    );

    fireEvent.click(screen.getByText("Send feedback"));

    await waitFor(() => {
      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(body.metaType).toBe("thumbs_down");
    });
  });

  it("shows error message on fetch failure", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      json: async () => ({ error: "Server error" }),
    });

    render(
      <FeedbackModal
        open={true}
        onOpenChange={vi.fn()}
        feedbackType="positive"
      />
    );

    fireEvent.click(screen.getByText("Send feedback"));

    await waitFor(() => {
      expect(screen.getByText("Server error")).toBeTruthy();
    });
  });

  it("shows error when fetch throws", async () => {
    mockFetch.mockRejectedValueOnce(new Error("Network error"));

    render(
      <FeedbackModal
        open={true}
        onOpenChange={vi.fn()}
        feedbackType="positive"
      />
    );

    fireEvent.click(screen.getByText("Send feedback"));

    await waitFor(() => {
      expect(screen.getByText("Network error")).toBeTruthy();
    });
  });

  it("includes comment in payload when provided", async () => {
    render(
      <FeedbackModal
        open={true}
        onOpenChange={vi.fn()}
        feedbackType="positive"
      />
    );

    const textarea = screen.getByPlaceholderText("What was good about this response?");
    fireEvent.change(textarea, { target: { value: "Very helpful!" } });
    fireEvent.click(screen.getByText("Send feedback"));

    await waitFor(() => {
      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(body.payload.comment).toBe("Very helpful!");
    });
  });

  it("shows privacy disclaimer", () => {
    render(
      <FeedbackModal
        open={true}
        onOpenChange={vi.fn()}
        feedbackType="positive"
      />
    );
    expect(screen.getByText("Privacy")).toBeTruthy();
    expect(screen.getByText(/will be stored to help improve/)).toBeTruthy();
  });
});
