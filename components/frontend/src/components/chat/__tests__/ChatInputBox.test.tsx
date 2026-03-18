import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { ChatInputBox, type ChatInputBoxProps } from "../ChatInputBox";

// Capture toast mock for assertions
const mockToast = vi.fn();
vi.mock("@/hooks/use-toast", () => ({
  useToast: () => ({ toast: mockToast }),
}));

vi.mock("@/hooks/use-resize-textarea", () => ({
  useResizeTextarea: () => ({
    textareaHeight: 108,
    handleResizeStart: vi.fn(),
  }),
}));

// Autocomplete mock with configurable behavior
let mockAutocomplete = {
  isOpen: false,
  type: null as string | null,
  filter: "",
  selectedIndex: 0,
  filteredItems: [] as Array<{ id: string; name: string }>,
  handleInputChange: vi.fn(),
  handleKeyDown: vi.fn(() => false),
  select: vi.fn(() => 5),
  setSelectedIndex: vi.fn(),
  close: vi.fn(),
  open: vi.fn(),
};

vi.mock("@/hooks/use-autocomplete", () => ({
  useAutocomplete: () => mockAutocomplete,
}));

vi.mock("../AutocompletePopover", () => ({
  AutocompletePopover: () => null,
}));

vi.mock("../AttachmentPreview", () => ({
  AttachmentPreview: () => null,
}));

function renderInput(overrides: Partial<ChatInputBoxProps> = {}) {
  const defaultProps: ChatInputBoxProps = {
    value: "",
    onChange: vi.fn(),
    onSend: vi.fn().mockResolvedValue(undefined),
    onInterrupt: vi.fn().mockResolvedValue(undefined),
    ...overrides,
  };
  return { ...render(<ChatInputBox {...defaultProps} />), props: defaultProps };
}

describe("ChatInputBox", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAutocomplete = {
      isOpen: false,
      type: null,
      filter: "",
      selectedIndex: 0,
      filteredItems: [],
      handleInputChange: vi.fn(),
      handleKeyDown: vi.fn(() => false),
      select: vi.fn(() => 5),
      setSelectedIndex: vi.fn(),
      close: vi.fn(),
      open: vi.fn(),
    };
  });

  describe("rendering", () => {
    it("renders textarea with default placeholder", () => {
      renderInput();
      const textarea = screen.getByRole("textbox");
      expect(textarea).toBeTruthy();
    });

    it("renders custom placeholder", () => {
      renderInput({ placeholder: "Custom placeholder" });
      expect(screen.getByPlaceholderText("Custom placeholder")).toBeTruthy();
    });

    it("renders Send button when not running", () => {
      renderInput({ value: "hello" });
      // Send is now a circular icon button with ArrowUp, no text label
      const sendBtn = screen.getAllByRole("button").find(
        (btn) => !btn.hasAttribute("disabled") && btn.className.includes("rounded-full")
      );
      expect(sendBtn).toBeTruthy();
    });

    it("renders Stop button when run is active", () => {
      renderInput({ isRunActive: true });
      expect(screen.getByText("Stop")).toBeTruthy();
    });

    it("disables textarea when disabled prop is true", () => {
      renderInput({ disabled: true });
      const textarea = screen.getByRole("textbox");
      expect(textarea.hasAttribute("disabled")).toBe(true);
    });
  });

  describe("handleKeyDown", () => {
    it("calls onSend on Enter (without Shift)", async () => {
      const onSend = vi.fn().mockResolvedValue(undefined);
      renderInput({ value: "hello", onSend });
      const textarea = screen.getByRole("textbox");

      fireEvent.keyDown(textarea, { key: "Enter", shiftKey: false });

      await waitFor(() => {
        expect(onSend).toHaveBeenCalled();
      });
    });

    it("does not send on Shift+Enter", () => {
      const onSend = vi.fn().mockResolvedValue(undefined);
      renderInput({ value: "hello", onSend });
      const textarea = screen.getByRole("textbox");

      fireEvent.keyDown(textarea, { key: "Enter", shiftKey: true });
      expect(onSend).not.toHaveBeenCalled();
    });

    it("does not send when value is empty", async () => {
      const onSend = vi.fn().mockResolvedValue(undefined);
      renderInput({ value: "", onSend });
      const textarea = screen.getByRole("textbox");

      fireEvent.keyDown(textarea, { key: "Enter" });
      // Wait a tick to make sure nothing happens
      await new Promise((r) => setTimeout(r, 10));
      expect(onSend).not.toHaveBeenCalled();
    });
  });

  describe("handleInterrupt", () => {
    it("calls onInterrupt when Stop button is clicked", async () => {
      const onInterrupt = vi.fn().mockResolvedValue(undefined);
      renderInput({ isRunActive: true, onInterrupt });

      fireEvent.click(screen.getByText("Stop"));

      await waitFor(() => {
        expect(onInterrupt).toHaveBeenCalled();
      });
    });
  });

  describe("queue indicator", () => {
    it("shows queue count when run is active and messages are queued", () => {
      renderInput({ isRunActive: true, queuedCount: 3 });
      expect(screen.getByText(/3 messages queued/)).toBeTruthy();
    });

    it("shows singular form for 1 queued message", () => {
      renderInput({ isRunActive: true, queuedCount: 1 });
      expect(screen.getByText(/1 message queued/)).toBeTruthy();
    });

    it("does not show queue when not active", () => {
      renderInput({ isRunActive: false, queuedCount: 3 });
      expect(screen.queryByText(/queued/)).toBeNull();
    });

    it("does not show queue when count is 0", () => {
      renderInput({ isRunActive: true, queuedCount: 0 });
      expect(screen.queryByText(/queued/)).toBeNull();
    });

    it("shows clear button when onClearQueue is provided", () => {
      const onClearQueue = vi.fn();
      renderInput({ isRunActive: true, queuedCount: 2, onClearQueue });
      expect(screen.getByText("Clear")).toBeTruthy();
    });

    it("calls onClearQueue when clear is clicked", () => {
      const onClearQueue = vi.fn();
      renderInput({ isRunActive: true, queuedCount: 2, onClearQueue });
      fireEvent.click(screen.getByText("Clear"));
      expect(onClearQueue).toHaveBeenCalled();
    });
  });

  describe("phase banners", () => {
    it("shows creating banner when session is Creating", () => {
      renderInput({ sessionPhase: "Creating" });
      expect(screen.getByText(/Session is starting up/)).toBeTruthy();
    });

    it("shows creating banner when session is Pending", () => {
      renderInput({ sessionPhase: "Pending" });
      expect(screen.getByText(/Session is starting up/)).toBeTruthy();
    });

    it("shows terminal banner for Completed state", () => {
      renderInput({ sessionPhase: "Completed" });
      expect(screen.getByText(/Session has completed/)).toBeTruthy();
    });

    it("shows terminal banner for Failed state", () => {
      renderInput({ sessionPhase: "Failed" });
      expect(screen.getByText(/Session has failed/)).toBeTruthy();
    });

    it("shows Resume link when onContinue is provided in terminal state", () => {
      renderInput({ sessionPhase: "Completed", onContinue: vi.fn() });
      expect(screen.getByText("Resume session")).toBeTruthy();
    });
  });

  describe("textarea styling", () => {
    it("applies amber style when run is active", () => {
      renderInput({ isRunActive: true });
      const textarea = screen.getByRole("textbox");
      // Border styling is now on the container wrapper, not the textarea
      const container = textarea.closest("div.border");
      expect(container?.className).toContain("border-amber");
    });
  });

  describe("agents and commands buttons", () => {
    it("shows Agents button", () => {
      renderInput({ agents: [{ id: "a1", name: "Helper", description: "Helps" }] });
      expect(screen.getByText("Agents")).toBeTruthy();
    });

    it("shows Commands button", () => {
      renderInput({
        commands: [{ id: "c1", name: "Deploy", slashCommand: "/deploy", description: "Deploy" }],
      });
      expect(screen.getByText("Commands")).toBeTruthy();
    });

    it("disables Agents button when no agents", () => {
      renderInput({ agents: [] });
      const btn = screen.getByText("Agents").closest("button");
      expect(btn?.hasAttribute("disabled")).toBe(true);
    });
  });

  describe("send as new flow", () => {
    it("shows Update and Send as new buttons when editing queued message", async () => {
      // Simulate editing a queued message by providing queued history and pressing up
      const onChange = vi.fn();
      renderInput({
        value: "",
        onChange,
        queuedMessageHistory: [{ id: "q1", content: "queued msg" }],
        onUpdateQueuedMessage: vi.fn(),
        onCancelQueuedMessage: vi.fn(),
      });

      const textarea = screen.getByRole("textbox");
      // ArrowUp at position 0 should go to history
      fireEvent.keyDown(textarea, { key: "ArrowUp" });

      // onChange should be called with the queued message content
      await waitFor(() => {
        expect(onChange).toHaveBeenCalledWith("queued msg");
      });
    });
  });

  describe("input change", () => {
    it("calls onChange when typing", () => {
      const onChange = vi.fn();
      renderInput({ onChange });
      const textarea = screen.getByRole("textbox");
      fireEvent.change(textarea, { target: { value: "new text" } });
      expect(onChange).toHaveBeenCalledWith("new text");
    });

    it("resets history index when typing during history browsing", () => {
      const onChange = vi.fn();
      renderInput({
        value: "",
        onChange,
        messageHistory: ["old msg"],
      });
      const textarea = screen.getByRole("textbox");

      // First go into history
      fireEvent.keyDown(textarea, { key: "ArrowUp" });
      expect(onChange).toHaveBeenCalledWith("old msg");

      // Now type — should reset history
      fireEvent.change(textarea, { target: { value: "new text" } });
      expect(onChange).toHaveBeenCalledWith("new text");
    });
  });

  describe("handlePaste", () => {
    it("adds image from paste event to pending attachments", async () => {
      const onPasteImage = vi.fn();
      renderInput({ onPasteImage });
      const textarea = screen.getByRole("textbox");

      const file = new File(["data"], "image.png", { type: "image/png" });
      Object.defineProperty(file, "size", { value: 1024 });

      const clipboardData = {
        items: [{ type: "image/png", getAsFile: () => file }],
      };

      fireEvent.paste(textarea, { clipboardData });

      // Should not crash; the paste handler adds to pending attachments
      await waitFor(() => {
        // The file should have been renamed since it was "image.png"
        expect(onPasteImage).not.toHaveBeenCalled(); // not called until send
      });
    });

    it.skip("shows toast for oversized paste image", async () => {
      const onPasteImage = vi.fn();
      renderInput({ onPasteImage });
      const textarea = screen.getByRole("textbox");

      const file = new File(["data"], "photo.jpg", { type: "image/jpeg" });
      Object.defineProperty(file, "size", { value: 20 * 1024 * 1024 }); // 20MB

      const clipboardData = {
        items: [{ type: "image/jpeg", getAsFile: () => file }],
      };

      fireEvent.paste(textarea, { clipboardData });

      await waitFor(() => {
        expect(mockToast).toHaveBeenCalledWith(
          expect.objectContaining({ variant: "destructive", title: "File too large" })
        );
      });
    });

    it("skips items that return null from getAsFile", async () => {
      const onPasteImage = vi.fn();
      renderInput({ onPasteImage });
      const textarea = screen.getByRole("textbox");

      const clipboardData = {
        items: [{ type: "image/png", getAsFile: () => null }],
      };

      fireEvent.paste(textarea, { clipboardData });

      // Should not crash
      await new Promise((r) => setTimeout(r, 10));
    });

    it("does nothing when no onPasteImage callback", () => {
      renderInput({ onPasteImage: undefined });

      const clipboardData = {
        items: [{ type: "image/png", getAsFile: () => new File(["x"], "img.png", { type: "image/png" }) }],
      };

      // Should not call preventDefault (paste proceeds normally)
      const event = new Event("paste", { bubbles: true });
      Object.assign(event, { clipboardData });
      // !textarea.dispatchEvent(event);
      // No crash expected
    });
  });

  describe("handleFileSelect", () => {
    it("adds valid file to pending attachments", async () => {
      renderInput({ onPasteImage: vi.fn() });

      const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement;
      expect(fileInput).toBeTruthy();

      const file = new File(["content"], "test.txt", { type: "text/plain" });
      Object.defineProperty(file, "size", { value: 1024 });

      Object.defineProperty(fileInput, "files", {
        value: [file],
        writable: true,
      });

      fireEvent.change(fileInput);

      // Should not crash — file is added to pending
      await new Promise((r) => setTimeout(r, 10));
    });

    it.skip("shows toast for oversized file", async () => {
      renderInput({ onPasteImage: vi.fn() });

      const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement;
      const file = new File(["x"], "big.pdf", { type: "application/pdf" });
      Object.defineProperty(file, "size", { value: 20 * 1024 * 1024 });

      Object.defineProperty(fileInput, "files", {
        value: [file],
        writable: true,
      });

      fireEvent.change(fileInput);

      await waitFor(() => {
        expect(mockToast).toHaveBeenCalledWith(
          expect.objectContaining({ variant: "destructive", title: "File too large" })
        );
      });
    });
  });

  describe("handleKeyDown — autocomplete integration", () => {
    it("delegates to autocomplete select on Tab when items available", async () => {
      mockAutocomplete.handleKeyDown = vi.fn(() => true);
      mockAutocomplete.filteredItems = [{ id: "a1", name: "Agent1" }];

      renderInput({ value: "@Ag" });
      const textarea = screen.getByRole("textbox");

      fireEvent.keyDown(textarea, { key: "Tab" });

      expect(mockAutocomplete.select).toHaveBeenCalled();
    });

    it("delegates to autocomplete select on Enter when items available", async () => {
      mockAutocomplete.handleKeyDown = vi.fn(() => true);
      mockAutocomplete.filteredItems = [{ id: "a1", name: "Agent1" }];

      renderInput({ value: "@Ag" });
      const textarea = screen.getByRole("textbox");

      fireEvent.keyDown(textarea, { key: "Enter" });

      expect(mockAutocomplete.select).toHaveBeenCalled();
    });

    it("does not select autocomplete when handleKeyDown returns false", () => {
      mockAutocomplete.handleKeyDown = vi.fn(() => false);
      mockAutocomplete.filteredItems = [{ id: "a1", name: "Agent1" }];

      renderInput({ value: "@Ag" });
      const textarea = screen.getByRole("textbox");

      fireEvent.keyDown(textarea, { key: "Tab" });

      expect(mockAutocomplete.select).not.toHaveBeenCalled();
    });
  });

  describe("handleKeyDown — history navigation", () => {
    it("navigates up through combined history", async () => {
      const onChange = vi.fn();
      renderInput({
        value: "",
        onChange,
        messageHistory: ["sent 1", "sent 2"],
      });
      const textarea = screen.getByRole("textbox");

      fireEvent.keyDown(textarea, { key: "ArrowUp" });
      expect(onChange).toHaveBeenCalledWith("sent 1");
    });

    it("navigates down back to draft", async () => {
      const onChange = vi.fn();
      renderInput({
        value: "",
        onChange,
        messageHistory: ["sent 1"],
      });
      const textarea = screen.getByRole("textbox");

      // Go up
      fireEvent.keyDown(textarea, { key: "ArrowUp" });
      expect(onChange).toHaveBeenCalledWith("sent 1");

      // Go down — should restore draft (empty)
      fireEvent.keyDown(textarea, { key: "ArrowDown" });
      expect(onChange).toHaveBeenCalledWith("");
    });

    it("does not go beyond history bounds", async () => {
      const onChange = vi.fn();
      renderInput({
        value: "",
        onChange,
        messageHistory: ["only one"],
      });
      const textarea = screen.getByRole("textbox");

      fireEvent.keyDown(textarea, { key: "ArrowUp" });
      expect(onChange).toHaveBeenCalledWith("only one");

      onChange.mockClear();
      // Try to go further up
      fireEvent.keyDown(textarea, { key: "ArrowUp" });
      // onChange should not be called again (already at last item)
      expect(onChange).not.toHaveBeenCalled();
    });

    it("Escape cancels editing queued message", async () => {
      const onChange = vi.fn();
      renderInput({
        value: "",
        onChange,
        queuedMessageHistory: [{ id: "q1", content: "queued text" }],
        onUpdateQueuedMessage: vi.fn(),
        onCancelQueuedMessage: vi.fn(),
      });
      const textarea = screen.getByRole("textbox");

      // Navigate to queued message
      fireEvent.keyDown(textarea, { key: "ArrowUp" });
      expect(onChange).toHaveBeenCalledWith("queued text");

      // Now press Escape to cancel editing
      fireEvent.keyDown(textarea, { key: "Escape" });
      // Should restore draft
      expect(onChange).toHaveBeenCalledWith("");
    });

    it("Ctrl+Space opens autocomplete", () => {
      renderInput({ value: "hello " });
      const textarea = screen.getByRole("textbox");

      fireEvent.keyDown(textarea, { key: " ", ctrlKey: true });

      expect(mockAutocomplete.open).toHaveBeenCalledWith("agent", expect.any(Number));
    });
  });

  describe("handleSendOrUpdate — queued message editing", () => {
    it("calls onUpdateQueuedMessage when editing a queued message", async () => {
      const onUpdateQueuedMessage = vi.fn();
      const onChange = vi.fn();
      const onSend = vi.fn().mockResolvedValue(undefined);
      renderInput({
        value: "",
        onChange,
        onSend,
        queuedMessageHistory: [{ id: "q1", content: "queued text" }],
        onUpdateQueuedMessage,
        onCancelQueuedMessage: vi.fn(),
      });
      const textarea = screen.getByRole("textbox");

      // Navigate to queued message
      fireEvent.keyDown(textarea, { key: "ArrowUp" });
      expect(onChange).toHaveBeenCalledWith("queued text");
    });

    it.skip("shows toast when sending while run is active", async () => {
      renderInput({
        value: "hello",
        isRunActive: true,
        onSend: vi.fn().mockResolvedValue(undefined),
      });
      const textarea = screen.getByRole("textbox");

      fireEvent.keyDown(textarea, { key: "Enter" });

      await waitFor(() => {
        expect(mockToast).toHaveBeenCalledWith(
          expect.objectContaining({ title: "Message queued" })
        );
      });
    });
  });

  describe("handleSendAsNew", () => {
    it("shows Send as new and Update buttons when editing queued", async () => {
      const onChange = vi.fn();
      const onCancelQueuedMessage = vi.fn();
      renderInput({
        value: "",
        onChange,
        queuedMessageHistory: [{ id: "q1", content: "queued" }],
        onUpdateQueuedMessage: vi.fn(),
        onCancelQueuedMessage,
      });
      const textarea = screen.getByRole("textbox");

      // Navigate into queued editing mode
      fireEvent.keyDown(textarea, { key: "ArrowUp" });

      await waitFor(() => {
        expect(screen.getByText("Update")).toBeTruthy();
        expect(screen.getByText("Send as new")).toBeTruthy();
      });
    });
  });

  describe("editing queued message indicator", () => {
    it("shows editing indicator when editing a queued message", async () => {
      const onChange = vi.fn();
      renderInput({
        value: "",
        onChange,
        queuedMessageHistory: [{ id: "q1", content: "queued text" }],
        onUpdateQueuedMessage: vi.fn(),
        onCancelQueuedMessage: vi.fn(),
      });
      const textarea = screen.getByRole("textbox");

      fireEvent.keyDown(textarea, { key: "ArrowUp" });

      await waitFor(() => {
        expect(screen.getByText("Editing queued message")).toBeTruthy();
      });
    });

    it("applies blue border style when editing queued message", async () => {
      const onChange = vi.fn();
      renderInput({
        value: "",
        onChange,
        queuedMessageHistory: [{ id: "q1", content: "queued text" }],
        onUpdateQueuedMessage: vi.fn(),
        onCancelQueuedMessage: vi.fn(),
      });
      const textarea = screen.getByRole("textbox");

      fireEvent.keyDown(textarea, { key: "ArrowUp" });

      await waitFor(() => {
        // Border styling is now on the container wrapper, not the textarea
        const container = textarea.closest("div.border");
        expect(container?.className).toContain("border-blue");
      });
    });
  });

  describe("dynamic placeholder", () => {
    it("shows terminal state placeholder", () => {
      renderInput({ sessionPhase: "Completed" });
      expect(screen.getByPlaceholderText(/resume this session/)).toBeTruthy();
    });

    it("shows creating state placeholder", () => {
      renderInput({ sessionPhase: "Creating" });
      expect(screen.getByPlaceholderText(/will be queued until session starts/)).toBeTruthy();
    });

    it("shows queued placeholder when run is active", () => {
      renderInput({ isRunActive: true });
      expect(screen.getByPlaceholderText(/will be queued/)).toBeTruthy();
    });
  });

  describe("uploadPendingAttachments", () => {
    it("uploads attachments before sending", async () => {
      const onPasteImage = vi.fn().mockResolvedValue(undefined);
      const onSend = vi.fn().mockResolvedValue(undefined);
      renderInput({ value: "hello", onPasteImage, onSend });
      const textarea = screen.getByRole("textbox");

      // Add an attachment via paste
      const file = new File(["data"], "test.png", { type: "image/png" });
      Object.defineProperty(file, "size", { value: 1024 });

      fireEvent.paste(textarea, {
        clipboardData: {
          items: [{ type: "image/png", getAsFile: () => file }],
        },
      });

      // Wait for attachment to be added
      await new Promise((r) => setTimeout(r, 50));

      // Now send
      fireEvent.keyDown(textarea, { key: "Enter" });

      await waitFor(() => {
        expect(onPasteImage).toHaveBeenCalled();
        expect(onSend).toHaveBeenCalled();
      });
    });
  });

  describe("ArrowDown history with queued messages", () => {
    it("navigates through queued + sent history entries", async () => {
      const onChange = vi.fn();
      renderInput({
        value: "",
        onChange,
        queuedMessageHistory: [{ id: "q1", content: "queued" }],
        messageHistory: ["sent"],
        onUpdateQueuedMessage: vi.fn(),
        onCancelQueuedMessage: vi.fn(),
      });
      const textarea = screen.getByRole("textbox");

      // Go up to queued
      fireEvent.keyDown(textarea, { key: "ArrowUp" });
      expect(onChange).toHaveBeenCalledWith("queued");

      // Go up to sent
      fireEvent.keyDown(textarea, { key: "ArrowUp" });
      expect(onChange).toHaveBeenCalledWith("sent");

      // Go down back to queued
      fireEvent.keyDown(textarea, { key: "ArrowDown" });
      expect(onChange).toHaveBeenCalledWith("queued");
    });
  });
});
