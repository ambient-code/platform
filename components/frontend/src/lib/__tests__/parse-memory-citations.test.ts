import { describe, it, expect } from "vitest";
import {
  parseMemoryCitations,
  extractCitationIds,
  type CitationSegment,
} from "../parse-memory-citations";

describe("extractCitationIds", () => {
  it("extracts single citation ID", () => {
    const ids = extractCitationIds("Used knowledge from [memory:PM-042] here.");
    expect(ids).toEqual(["PM-042"]);
  });

  it("extracts multiple citation IDs", () => {
    const ids = extractCitationIds(
      "Based on [memory:PM-001] and [memory:PM-002] we should..."
    );
    expect(ids).toEqual(["PM-001", "PM-002"]);
  });

  it("returns empty array for no citations", () => {
    const ids = extractCitationIds("No citations here.");
    expect(ids).toEqual([]);
  });

  it("deduplicates repeated IDs", () => {
    const ids = extractCitationIds(
      "[memory:PM-001] and again [memory:PM-001]"
    );
    expect(ids).toEqual(["PM-001"]);
  });
});

describe("parseMemoryCitations", () => {
  it("returns single text segment when no citations present", () => {
    const result = parseMemoryCitations("Hello world");
    expect(result).toEqual([{ type: "text", value: "Hello world" }]);
  });

  it("splits text around a single citation", () => {
    const result = parseMemoryCitations("Before [memory:PM-042] after");
    expect(result).toEqual([
      { type: "text", value: "Before " },
      { type: "citation", value: "PM-042" },
      { type: "text", value: " after" },
    ]);
  });

  it("handles multiple citations", () => {
    const result = parseMemoryCitations(
      "[memory:PM-001] start and [memory:PM-002] end"
    );
    expect(result).toEqual([
      { type: "citation", value: "PM-001" },
      { type: "text", value: " start and " },
      { type: "citation", value: "PM-002" },
      { type: "text", value: " end" },
    ]);
  });

  it("handles citation at end of string", () => {
    const result = parseMemoryCitations("See [memory:PM-099]");
    expect(result).toEqual([
      { type: "text", value: "See " },
      { type: "citation", value: "PM-099" },
    ]);
  });

  it("does NOT parse citations inside fenced code blocks", () => {
    const input =
      "Before\n```\n[memory:PM-001]\n```\nAfter [memory:PM-002]";
    const result = parseMemoryCitations(input);
    // PM-001 is inside code block, should remain as text
    // PM-002 is outside, should be parsed
    const citations = result.filter(
      (s): s is CitationSegment & { type: "citation" } =>
        s.type === "citation"
    );
    expect(citations).toHaveLength(1);
    expect(citations[0].value).toBe("PM-002");
  });

  it("does NOT parse citations inside inline code", () => {
    const input = "See `[memory:PM-001]` and [memory:PM-002]";
    const result = parseMemoryCitations(input);
    const citations = result.filter(
      (s): s is CitationSegment & { type: "citation" } =>
        s.type === "citation"
    );
    expect(citations).toHaveLength(1);
    expect(citations[0].value).toBe("PM-002");
  });

  it("handles empty string", () => {
    const result = parseMemoryCitations("");
    expect(result).toEqual([{ type: "text", value: "" }]);
  });

  it("handles string that is only a citation", () => {
    const result = parseMemoryCitations("[memory:PM-001]");
    expect(result).toEqual([{ type: "citation", value: "PM-001" }]);
  });
});
