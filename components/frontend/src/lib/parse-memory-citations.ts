/**
 * Parse [memory:PM-XXX] citation patterns from agent message text.
 *
 * Citations inside fenced code blocks (``` ... ```) or inline code (` ... `)
 * are NOT parsed -- they are left as plain text (FR-005).
 */

const CITATION_PATTERN = /\[memory:(PM-\d+)\]/g;

export type CitationSegment =
  | { type: "text"; value: string }
  | { type: "citation"; value: string };

/**
 * Extract all unique citation IDs from a text string.
 * Does not account for code blocks -- use parseMemoryCitations for rendering.
 */
export function extractCitationIds(text: string): string[] {
  const ids = new Set<string>();
  const re = new RegExp(CITATION_PATTERN.source, "g");
  let match: RegExpExecArray | null;
  while ((match = re.exec(text)) !== null) {
    ids.add(match[1]);
  }
  return Array.from(ids);
}

/**
 * Mask code blocks and inline code so citations inside them are not parsed.
 *
 * Replaces fenced code blocks and inline code with placeholder strings,
 * returning the masked text and a map for restoring them later.
 */
function maskCodeBlocks(text: string): {
  masked: string;
  placeholders: Map<string, string>;
} {
  const placeholders = new Map<string, string>();
  let counter = 0;

  // Mask fenced code blocks first (triple-backtick blocks)
  let masked = text.replace(/```[\s\S]*?```/g, (match) => {
    const key = `\x00CODEBLOCK_${counter++}\x00`;
    placeholders.set(key, match);
    return key;
  });

  // Mask inline code (single-backtick spans)
  masked = masked.replace(/`[^`]+`/g, (match) => {
    const key = `\x00CODEBLOCK_${counter++}\x00`;
    placeholders.set(key, match);
    return key;
  });

  return { masked, placeholders };
}

/** Restore code block placeholders back to original text. */
function restorePlaceholders(
  text: string,
  placeholders: Map<string, string>
): string {
  let result = text;
  for (const [key, original] of placeholders) {
    result = result.replace(key, original);
  }
  return result;
}

/**
 * Parse message text into segments of plain text and citations.
 *
 * Citations inside code blocks are preserved as plain text.
 * Returns an array of segments that can be rendered as React elements.
 */
export function parseMemoryCitations(text: string): CitationSegment[] {
  if (!text) {
    return [{ type: "text", value: "" }];
  }

  const { masked, placeholders } = maskCodeBlocks(text);

  // Check if there are any citations in the unmasked text
  const re = new RegExp(CITATION_PATTERN.source, "g");
  if (!re.test(masked)) {
    return [{ type: "text", value: text }];
  }

  // Split the masked text around citations
  const segments: CitationSegment[] = [];
  re.lastIndex = 0;
  let lastIndex = 0;
  let match: RegExpExecArray | null;

  while ((match = re.exec(masked)) !== null) {
    // Add text before this citation
    if (match.index > lastIndex) {
      const rawText = masked.slice(lastIndex, match.index);
      segments.push({
        type: "text",
        value: restorePlaceholders(rawText, placeholders),
      });
    }
    segments.push({ type: "citation", value: match[1] });
    lastIndex = match.index + match[0].length;
  }

  // Add remaining text after last citation
  if (lastIndex < masked.length) {
    const rawText = masked.slice(lastIndex);
    segments.push({
      type: "text",
      value: restorePlaceholders(rawText, placeholders),
    });
  }

  return segments;
}
