/**
 * Types for the Learning dashboard (spec 006).
 * Matches backend response shapes from handlers/learning.go.
 */

export type LearningSummary = {
  totalCorrections: number;
  correctionsByType: Record<string, number>;
  improvementSessions: number;
  memoriesCreated: number;
  memoryCitations: number;
};

export type TimelineEntry = {
  id: string;
  timestamp: string;
  eventType: string;
  summary: string;
  correctionType?: string;
  improvementSession?: string;
  memoryId?: string;
};

export type TimelineResponse = {
  items: TimelineEntry[];
  totalCount: number;
  page: number;
  pageSize: number;
};
