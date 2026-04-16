import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import LearningPage from "../page";
import type { LearningSummary, TimelineResponse } from "@/types/learning";

// Mock Next.js navigation
vi.mock("next/navigation", () => ({
  useParams: () => ({ name: "test-project" }),
  useRouter: () => ({ push: vi.fn() }),
  usePathname: () => "/projects/test-project/learning",
}));

// Mock feature flag hook - use a mutable object so tests can override
const flagState = { enabled: true, isLoading: false, source: "unleash" as const, error: null };
vi.mock("@/services/queries/use-feature-flags-admin", () => ({
  useWorkspaceFlag: () => flagState,
}));

// Mock learning hooks - use mutable state objects
const summaryState: {
  data: LearningSummary | undefined;
  isLoading: boolean;
  error: null;
} = {
  data: undefined,
  isLoading: false,
  error: null,
};
const timelineState: {
  data: TimelineResponse | undefined;
  isLoading: boolean;
  error: null;
} = {
  data: undefined,
  isLoading: false,
  error: null,
};

vi.mock("@/services/queries/use-learning", () => ({
  useLearningSummary: () => summaryState,
  useLearningTimeline: () => timelineState,
}));

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <LearningPage />
    </QueryClientProvider>
  );
}

describe("LearningPage", () => {
  beforeEach(() => {
    summaryState.data = undefined;
    summaryState.isLoading = false;
    timelineState.data = undefined;
    timelineState.isLoading = false;
    flagState.enabled = true;
    flagState.isLoading = false;
  });

  it("shows empty state when no data", () => {
    summaryState.data = {
      totalCorrections: 0,
      correctionsByType: {},
      improvementSessions: 0,
      memoriesCreated: 0,
      memoryCitations: 0,
    };
    timelineState.data = {
      items: [],
      totalCount: 0,
      page: 1,
      pageSize: 20,
    };

    renderPage();
    expect(screen.getByText("No learning data yet")).toBeTruthy();
  });

  it("shows disabled message when flag is off", () => {
    flagState.enabled = false;

    renderPage();
    expect(
      screen.getByText(/learning agent loop feature is not enabled/)
    ).toBeTruthy();
  });

  it("renders summary cards with data", () => {
    summaryState.data = {
      totalCorrections: 10,
      correctionsByType: { incomplete: 4, style: 6 },
      improvementSessions: 3,
      memoriesCreated: 5,
      memoryCitations: 12,
    };
    timelineState.data = {
      items: [],
      totalCount: 0,
      page: 1,
      pageSize: 20,
    };

    renderPage();
    expect(screen.getByText("10")).toBeTruthy();
    expect(screen.getByText("3")).toBeTruthy();
    expect(screen.getByText("5")).toBeTruthy();
    expect(screen.getByText("12")).toBeTruthy();
  });

  it("renders heading", () => {
    summaryState.data = {
      totalCorrections: 0,
      correctionsByType: {},
      improvementSessions: 0,
      memoriesCreated: 0,
      memoryCitations: 0,
    };
    timelineState.data = { items: [], totalCount: 0, page: 1, pageSize: 20 };

    renderPage();
    expect(screen.getByText("Learning")).toBeTruthy();
  });
});
