import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import {
  MemoryCitationBadge,
  MemoryCitationSummary,
} from "../memory-citation-badge";

// Mock Radix UI Popover for testability
vi.mock("radix-ui", () => ({
  Popover: {
    Root: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
    Trigger: ({
      children,
      ...props
    }: { children: React.ReactNode } & Record<string, unknown>) => (
      <button {...props}>{children}</button>
    ),
    Portal: ({ children }: { children: React.ReactNode }) => (
      <div>{children}</div>
    ),
    Content: ({
      children,
      ...props
    }: { children: React.ReactNode } & Record<string, unknown>) => (
      <div data-testid="popover-content" {...props}>
        {children}
      </div>
    ),
  },
  Tooltip: {
    Provider: ({ children }: { children: React.ReactNode }) => (
      <div>{children}</div>
    ),
    Root: ({ children }: { children: React.ReactNode }) => (
      <div>{children}</div>
    ),
    Trigger: ({ children }: { children: React.ReactNode }) => (
      <span>{children}</span>
    ),
    Portal: ({ children }: { children: React.ReactNode }) => (
      <div>{children}</div>
    ),
    Content: ({ children }: { children: React.ReactNode }) => (
      <div data-testid="tooltip-content">{children}</div>
    ),
    Arrow: () => null,
  },
}));

describe("MemoryCitationBadge", () => {
  it("renders memory ID in badge", () => {
    render(<MemoryCitationBadge memoryId="PM-042" />);
    // Badge text appears in both the trigger and the popover header
    const matches = screen.getAllByText("PM-042");
    expect(matches.length).toBeGreaterThanOrEqual(1);
  });

  it("renders with summary when provided", () => {
    render(
      <MemoryCitationBadge
        memoryId="PM-042"
        summary="Always use gofmt before committing Go code to ensure consistent formatting"
      />
    );
    const matches = screen.getAllByText("PM-042");
    expect(matches.length).toBeGreaterThanOrEqual(1);
  });

  it("renders warning state for non-existent memory (FR-015)", () => {
    render(<MemoryCitationBadge memoryId="PM-999" notFound />);
    const badge = screen.getByText("PM-999");
    expect(badge).toBeTruthy();
    // Should have warning indicator via tooltip
    expect(screen.getByTestId("tooltip-content")).toBeTruthy();
  });

  it("renders deprecated state with data attribute (FR-014)", () => {
    render(<MemoryCitationBadge memoryId="PM-042" deprecated />);
    const matches = screen.getAllByText("PM-042");
    // Find the one inside a button with data-deprecated
    const deprecatedButton = matches
      .map((el) => el.closest("[data-deprecated]"))
      .find(Boolean);
    expect(deprecatedButton).toBeTruthy();
  });

  it("shows popover content with full details", () => {
    render(
      <MemoryCitationBadge
        memoryId="PM-042"
        summary="Test summary"
        fullContent="Full memory content here"
        author="jeder"
        createdAt="2026-04-10"
      />
    );
    expect(screen.getByTestId("popover-content")).toBeTruthy();
    expect(screen.getByText(/Full memory content here/)).toBeTruthy();
  });
});

describe("MemoryCitationSummary", () => {
  it("renders citation count when over 10 (FR-016)", () => {
    render(<MemoryCitationSummary count={12} />);
    expect(screen.getByText("12 memories cited")).toBeTruthy();
  });

  it("does not render when count is 10 or fewer", () => {
    const { container } = render(<MemoryCitationSummary count={10} />);
    expect(container.textContent).toBe("");
  });
});
