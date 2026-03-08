import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock useSessionQueue before importing the hook under test
const mockSetWorkflow = vi.fn();
const mockClearWorkflow = vi.fn();
const mockMarkWorkflowActivated = vi.fn();
vi.mock("@/hooks/use-session-queue", () => ({
  useSessionQueue: () => ({
    messages: [],
    addMessage: vi.fn(),
    markMessageSent: vi.fn(),
    cancelMessage: vi.fn(),
    updateMessage: vi.fn(),
    clearMessages: vi.fn(),
    pendingCount: 0,
    workflow: null,
    setWorkflow: mockSetWorkflow,
    markWorkflowActivated: mockMarkWorkflowActivated,
    clearWorkflow: mockClearWorkflow,
    metadata: {},
    updateMetadata: vi.fn(),
  }),
}));

vi.mock("@/hooks/use-toast", () => ({
  errorToast: vi.fn(),
}));

import { useWorkflowManagement } from "../use-workflow-management";

// Helper to build a workflow config
function makeWorkflow(overrides: Record<string, unknown> = {}) {
  return {
    id: "wf-test",
    name: "Test Workflow",
    description: "A test workflow",
    gitUrl: "https://github.com/org/repo",
    branch: "main",
    path: "workflows/test",
    enabled: true,
    ...overrides,
  };
}

describe("useWorkflowManagement", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
    // Mock successful fetch for workflow activation
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ message: "Workflow updated" }),
    });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe("workflowGreeting — Critical fix: stale greeting cleared on switch", () => {
    /**
     * BUG (pre-patch): activateWorkflow only set workflowGreeting inside
     *   `if (workflow.startupPrompt) { setWorkflowGreeting(...) }`
     * so switching from a workflow WITH a greeting to one WITHOUT left the
     * old greeting displayed.
     *
     * FIX: `setWorkflowGreeting(workflow.startupPrompt || null)` —
     * unconditional, always clears or updates.
     *
     * This test FAILS on the pre-patch code and PASSES on the fix.
     */
    it("clears workflowGreeting when activating a workflow without startupPrompt after one with", async () => {
      const { result } = renderHook(() =>
        useWorkflowManagement({
          projectName: "proj",
          sessionName: "sess",
          sessionPhase: "Running",
        })
      );

      // Step 1: Activate a workflow that HAS a startupPrompt
      const withGreeting = makeWorkflow({
        id: "greeter",
        startupPrompt: "Welcome! Let me help you fix a bug.",
      });

      await act(async () => {
        const promise = result.current.activateWorkflow(withGreeting, "Running");
        // Advance past the 3s post-activation wait
        await vi.advanceTimersByTimeAsync(3000);
        await promise;
      });

      expect(result.current.workflowGreeting).toBe(
        "Welcome! Let me help you fix a bug."
      );

      // Step 2: Activate a workflow that does NOT have a startupPrompt
      const withoutGreeting = makeWorkflow({ id: "no-greeting" });

      await act(async () => {
        const promise = result.current.activateWorkflow(
          withoutGreeting,
          "Running"
        );
        await vi.advanceTimersByTimeAsync(3000);
        await promise;
      });

      // CRITICAL ASSERTION: greeting must be null, not stale
      expect(result.current.workflowGreeting).toBeNull();
    });

    it("sets workflowGreeting when activating a workflow with startupPrompt", async () => {
      const { result } = renderHook(() =>
        useWorkflowManagement({
          projectName: "proj",
          sessionName: "sess",
          sessionPhase: "Running",
        })
      );

      const workflow = makeWorkflow({
        startupPrompt: "Hello from the workflow!",
      });

      await act(async () => {
        const promise = result.current.activateWorkflow(workflow, "Running");
        await vi.advanceTimersByTimeAsync(3000);
        await promise;
      });

      expect(result.current.workflowGreeting).toBe(
        "Hello from the workflow!"
      );
    });

    it("sets workflowGreeting to null when startupPrompt is empty string", async () => {
      const { result } = renderHook(() =>
        useWorkflowManagement({
          projectName: "proj",
          sessionName: "sess",
          sessionPhase: "Running",
        })
      );

      // First activate one with a greeting
      await act(async () => {
        const promise = result.current.activateWorkflow(
          makeWorkflow({ id: "a", startupPrompt: "Hi" }),
          "Running"
        );
        await vi.advanceTimersByTimeAsync(3000);
        await promise;
      });
      expect(result.current.workflowGreeting).toBe("Hi");

      // Then activate one with empty string startupPrompt
      await act(async () => {
        const promise = result.current.activateWorkflow(
          makeWorkflow({ id: "b", startupPrompt: "" }),
          "Running"
        );
        await vi.advanceTimersByTimeAsync(3000);
        await promise;
      });

      expect(result.current.workflowGreeting).toBeNull();
    });
  });

  describe("workflowGreeting — starts null", () => {
    it("workflowGreeting is null on initial render", () => {
      const { result } = renderHook(() =>
        useWorkflowManagement({
          projectName: "proj",
          sessionName: "sess",
        })
      );

      expect(result.current.workflowGreeting).toBeNull();
    });
  });

  describe("workflowGreeting — not set on fetch failure", () => {
    it("does not set workflowGreeting if the activation API call fails", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        json: () =>
          Promise.resolve({ error: "Runner not available", retryable: false }),
      });

      const { result } = renderHook(() =>
        useWorkflowManagement({
          projectName: "proj",
          sessionName: "sess",
          sessionPhase: "Running",
        })
      );

      const workflow = makeWorkflow({
        startupPrompt: "Should not appear",
      });

      await act(async () => {
        await result.current.activateWorkflow(workflow, "Running");
      });

      expect(result.current.workflowGreeting).toBeNull();
    });
  });
});
