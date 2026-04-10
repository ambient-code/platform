import { describe, it, expect } from "vitest";
import {
  formatTimestamp,
  formatTimeUTC,
  formatTimeLocal,
  formatScheduleTime,
  formatScheduleDateTime,
} from "../format-timestamp";

describe("formatTimestamp", () => {
  it("returns empty string for undefined", () => {
    expect(formatTimestamp(undefined)).toBe("");
  });

  it("returns empty string for empty string", () => {
    expect(formatTimestamp("")).toBe("");
  });

  it("returns empty string for invalid date", () => {
    expect(formatTimestamp("not-a-date")).toBe("");
  });

  it("returns formatted date for valid ISO timestamp", () => {
    const result = formatTimestamp("2025-02-27T17:34:00Z");
    expect(result).toBeTruthy();
    expect(result.length).toBeGreaterThan(0);
  });
});

describe("formatTimeUTC", () => {
  it("includes UTC suffix", () => {
    const date = new Date("2025-06-15T17:00:00Z");
    const result = formatTimeUTC(date);
    expect(result).toContain("UTC");
  });

  it("formats time in UTC timezone", () => {
    const date = new Date("2025-06-15T17:00:00Z");
    const result = formatTimeUTC(date);
    // Should contain 5:00 PM or 17:00 depending on locale
    expect(result).toMatch(/5:00\s*PM\s*UTC|17:00\s*UTC/);
  });
});

describe("formatTimeLocal", () => {
  it("returns a non-empty string", () => {
    const date = new Date("2025-06-15T17:00:00Z");
    const result = formatTimeLocal(date);
    expect(result).toBeTruthy();
    expect(result.length).toBeGreaterThan(0);
  });

  it("includes a timezone abbreviation", () => {
    const date = new Date("2025-06-15T17:00:00Z");
    const result = formatTimeLocal(date);
    // Should include some timezone indicator (UTC, EDT, PST, etc.)
    expect(result).toMatch(/[A-Z]{2,}/);
  });
});

describe("formatScheduleTime", () => {
  it("includes UTC time", () => {
    const date = new Date("2025-06-15T17:00:00Z");
    const result = formatScheduleTime(date);
    expect(result).toContain("UTC");
  });

  it("returns a non-empty string for valid date", () => {
    const date = new Date("2025-06-15T17:00:00Z");
    const result = formatScheduleTime(date);
    expect(result).toBeTruthy();
  });
});

describe("formatScheduleDateTime", () => {
  it("includes UTC time", () => {
    const date = new Date("2025-06-15T17:00:00Z");
    const result = formatScheduleDateTime(date);
    expect(result).toContain("UTC");
  });

  it("includes date portion", () => {
    const date = new Date("2025-06-15T17:00:00Z");
    const result = formatScheduleDateTime(date);
    // Should contain a month abbreviation and day
    expect(result).toMatch(/\w+\s+\d+/);
  });

  it("returns a non-empty string for valid date", () => {
    const date = new Date("2025-06-15T17:00:00Z");
    const result = formatScheduleDateTime(date);
    expect(result).toBeTruthy();
  });
});
