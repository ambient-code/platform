import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { AdvancedSdkOptions } from "../advanced-sdk-options";
import {
  claudeAgentOptionsSchema,
  claudeAgentOptionsDefaults,
  type ClaudeAgentOptionsForm,
} from "../claude-agent-options";

// Mock AgentOptionsFields to avoid rendering the full form tree
vi.mock("../claude-agent-options", async () => {
  const actual = await vi.importActual("../claude-agent-options");
  return {
    ...actual,
    AgentOptionsFields: ({ disabled }: { disabled?: boolean }) => (
      <div data-testid="agent-options-fields" data-disabled={disabled}>
        Agent Options Fields
      </div>
    ),
  };
});

function renderWithForm(props?: { disabled?: boolean; onSave?: () => void }) {
  const onSave = props?.onSave ?? vi.fn();
  function TestHarness() {
    const form = useForm<ClaudeAgentOptionsForm>({
      resolver: zodResolver(claudeAgentOptionsSchema),
      defaultValues: claudeAgentOptionsDefaults,
    });
    return (
      <AdvancedSdkOptions
        projectName="test-project"
        form={form}
        disabled={props?.disabled}
        onSave={onSave}
      />
    );
  }
  return { ...render(<TestHarness />), onSave };
}

describe("AdvancedSdkOptions", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders form fields and save/cancel buttons", () => {
    renderWithForm();
    expect(screen.getByTestId("agent-options-fields")).toBeDefined();
    expect(screen.getByText("Save Options")).toBeDefined();
    expect(screen.getByText("Cancel")).toBeDefined();
  });

  it("calls onSave when Save Options is clicked", () => {
    const { onSave } = renderWithForm();
    fireEvent.click(screen.getByText("Save Options"));
    expect(onSave).toHaveBeenCalled();
  });

  it("calls onSave when Cancel is clicked with no changes", () => {
    const { onSave } = renderWithForm();
    fireEvent.click(screen.getByText("Cancel"));
    expect(onSave).toHaveBeenCalled();
  });

  it("disables buttons when disabled prop is true", () => {
    renderWithForm({ disabled: true });
    const saveBtn = screen.getByText("Save Options").closest("button");
    expect(saveBtn?.hasAttribute("disabled")).toBe(true);
  });
});
