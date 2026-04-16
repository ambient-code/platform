import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { CorrectionPopover } from "../CorrectionPopover";

// Radix Select/Popover calls scrollIntoView and ResizeObserver which are not available in jsdom
Element.prototype.scrollIntoView = vi.fn();

if (typeof globalThis.ResizeObserver === "undefined") {
  globalThis.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
}

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
  useFeedbackContextOptional: vi.fn(() => mockFeedbackContext),
}));

// Mock the corrections API
const mockSubmitCorrection = vi.fn();
vi.mock("@/services/api/corrections", () => ({
  submitCorrection: (...args: unknown[]) => mockSubmitCorrection(...args),
}));

describe("CorrectionPopover", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockSubmitCorrection.mockResolvedValue({
      message: "ok",
      status: "forwarded",
    });
  });

  it("renders the correct-this button", () => {
    render(
      <CorrectionPopover messageId="msg-1" messageContent="Hello world" />
    );
    expect(screen.getByLabelText("Correct this response")).toBeTruthy();
  });

  it("opens popover on click and shows form elements", async () => {
    render(
      <CorrectionPopover messageId="msg-1" messageContent="Hello world" />
    );
    fireEvent.click(screen.getByLabelText("Correct this response"));
    await waitFor(() => {
      expect(screen.getByText("Correct this response")).toBeTruthy();
      expect(screen.getByText("Correction type")).toBeTruthy();
      expect(
        screen.getByText("What should have happened instead?")
      ).toBeTruthy();
      expect(
        screen.getByText("Include message content as context")
      ).toBeTruthy();
    });
  });

  it("shows character counter starting at 0", async () => {
    render(
      <CorrectionPopover messageId="msg-1" messageContent="Hello world" />
    );
    fireEvent.click(screen.getByLabelText("Correct this response"));
    await waitFor(() => {
      expect(screen.getByText("0/2000")).toBeTruthy();
    });
  });

  it("updates character counter on typing", async () => {
    render(
      <CorrectionPopover messageId="msg-1" messageContent="Hello world" />
    );
    fireEvent.click(screen.getByLabelText("Correct this response"));
    await waitFor(() => {
      expect(
        screen.getByPlaceholderText("Describe the correct behavior...")
      ).toBeTruthy();
    });
    const textarea = screen.getByPlaceholderText(
      "Describe the correct behavior..."
    );
    fireEvent.change(textarea, { target: { value: "Short text" } });
    expect(screen.getByText("10/2000")).toBeTruthy();
  });

  it("closes popover on cancel without submitting", async () => {
    render(
      <CorrectionPopover messageId="msg-1" messageContent="Hello world" />
    );
    fireEvent.click(screen.getByLabelText("Correct this response"));
    await waitFor(() => {
      expect(screen.getByText("Cancel")).toBeTruthy();
    });
    fireEvent.click(screen.getByText("Cancel"));
    await waitFor(() => {
      expect(screen.queryByText("Correction type")).toBeNull();
    });
    expect(mockSubmitCorrection).not.toHaveBeenCalled();
  });

  it("has submit button disabled when form is incomplete", async () => {
    render(
      <CorrectionPopover messageId="msg-1" messageContent="Hello world" />
    );
    fireEvent.click(screen.getByLabelText("Correct this response"));
    await waitFor(() => {
      const submitBtn = screen.getByText("Submit");
      expect(submitBtn.closest("button")?.disabled).toBe(true);
    });
  });

  it("submit button remains disabled with text under 10 chars", async () => {
    render(
      <CorrectionPopover messageId="msg-1" messageContent="Hello world" />
    );
    fireEvent.click(screen.getByLabelText("Correct this response"));
    await waitFor(() => {
      expect(
        screen.getByPlaceholderText("Describe the correct behavior...")
      ).toBeTruthy();
    });
    const textarea = screen.getByPlaceholderText(
      "Describe the correct behavior..."
    );
    fireEvent.change(textarea, { target: { value: "too short" } });
    const submitBtn = screen.getByText("Submit");
    expect(submitBtn.closest("button")?.disabled).toBe(true);
  });

  it("shows error on submission failure without closing popover", async () => {
    mockSubmitCorrection.mockRejectedValueOnce(new Error("Network error"));

    render(
      <CorrectionPopover messageId="msg-1" messageContent="Hello world" />
    );
    fireEvent.click(screen.getByLabelText("Correct this response"));

    await waitFor(() => {
      expect(screen.getByText("Correction type")).toBeTruthy();
    });

    // Select correction type via the select trigger
    const trigger = screen.getByText("Select type...");
    fireEvent.click(trigger);

    await waitFor(() => {
      expect(screen.getByText("Incorrect")).toBeTruthy();
    });
    fireEvent.click(screen.getByText("Incorrect"));

    // Type correction text (>= 10 chars)
    const textarea = screen.getByPlaceholderText(
      "Describe the correct behavior..."
    );
    fireEvent.change(textarea, {
      target: {
        value: "Should use more concise language throughout the response",
      },
    });

    // Submit
    fireEvent.click(screen.getByText("Submit"));

    await waitFor(() => {
      expect(screen.getByText("Network error")).toBeTruthy();
    });

    // Popover should still be open with form intact
    expect(screen.getByText("Correction type")).toBeTruthy();
  });

  it("shows success indicator after submission", async () => {
    render(
      <CorrectionPopover messageId="msg-1" messageContent="Hello world" />
    );
    fireEvent.click(screen.getByLabelText("Correct this response"));

    await waitFor(() => {
      expect(screen.getByText("Correction type")).toBeTruthy();
    });

    // Select type
    fireEvent.click(screen.getByText("Select type..."));
    await waitFor(() => {
      expect(screen.getByText("Incorrect")).toBeTruthy();
    });
    fireEvent.click(screen.getByText("Incorrect"));

    // Type correction
    const textarea = screen.getByPlaceholderText(
      "Describe the correct behavior..."
    );
    fireEvent.change(textarea, {
      target: {
        value:
          "The response should have used a completely different approach here",
      },
    });

    // Submit
    fireEvent.click(screen.getByText("Submit"));

    await waitFor(() => {
      expect(mockSubmitCorrection).toHaveBeenCalledTimes(1);
    });

    // After successful submit, button should show submitted state
    await waitFor(() => {
      expect(screen.getByLabelText(/Correction submitted/)).toBeTruthy();
    });
  });

  it("submits correction with correct payload", async () => {
    render(
      <CorrectionPopover messageId="msg-1" messageContent="Hello world" />
    );
    fireEvent.click(screen.getByLabelText("Correct this response"));

    await waitFor(() => {
      expect(screen.getByText("Correction type")).toBeTruthy();
    });

    // Select type
    fireEvent.click(screen.getByText("Select type..."));
    await waitFor(() => {
      expect(screen.getByText("Incomplete")).toBeTruthy();
    });
    fireEvent.click(screen.getByText("Incomplete"));

    // Type correction
    const textarea = screen.getByPlaceholderText(
      "Describe the correct behavior..."
    );
    fireEvent.change(textarea, {
      target: {
        value: "Should also include the configuration steps for the setup",
      },
    });

    // Submit
    fireEvent.click(screen.getByText("Submit"));

    await waitFor(() => {
      expect(mockSubmitCorrection).toHaveBeenCalledWith("test-project", {
        correction_type: "incomplete",
        user_correction:
          "Should also include the configuration steps for the setup",
        session_name: "test-session",
        message_id: "msg-1",
        message_content: undefined,
        source: "user",
      });
    });
  });

  it("includes message content when checkbox is checked", async () => {
    render(
      <CorrectionPopover messageId="msg-1" messageContent="Hello world" />
    );
    fireEvent.click(screen.getByLabelText("Correct this response"));

    await waitFor(() => {
      expect(screen.getByText("Correction type")).toBeTruthy();
    });

    // Select type
    fireEvent.click(screen.getByText("Select type..."));
    await waitFor(() => {
      expect(screen.getByText("Style")).toBeTruthy();
    });
    fireEvent.click(screen.getByText("Style"));

    // Type correction
    const textarea = screen.getByPlaceholderText(
      "Describe the correct behavior..."
    );
    fireEvent.change(textarea, {
      target: {
        value: "The response should be more formal and professional in tone",
      },
    });

    // Check the include content checkbox
    const checkbox = screen.getByLabelText(
      "Include message content as context"
    );
    fireEvent.click(checkbox);

    // Submit
    fireEvent.click(screen.getByText("Submit"));

    await waitFor(() => {
      expect(mockSubmitCorrection).toHaveBeenCalledWith("test-project", {
        correction_type: "style",
        user_correction:
          "The response should be more formal and professional in tone",
        session_name: "test-session",
        message_id: "msg-1",
        message_content: "Hello world",
        source: "user",
      });
    });
  });

  it("does not render without feedback context", async () => {
    const { useFeedbackContextOptional } = await import(
      "@/contexts/FeedbackContext"
    );
    vi.mocked(useFeedbackContextOptional).mockReturnValueOnce(null);

    const { container } = render(
      <CorrectionPopover messageId="msg-1" messageContent="Hello world" />
    );
    expect(container.innerHTML).toBe("");
  });
});
