/** Strip the `mcp__` prefix, replace underscores, and title-case each word. */
export function formatToolName(name: string): string {
  return name
    .replace(/^mcp__/, "")
    .replace(/_/g, " ")
    .split(" ")
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(" ");
}

/** Return the first short string argument value as a summary, or fall back to the formatted name. */
export function toolSummary(name: string, args: Record<string, unknown>): string {
  if (!args || Object.keys(args).length === 0) return formatToolName(name);
  const first = Object.values(args).find(
    (v) => typeof v === "string" && v.length > 0,
  ) as string | undefined;
  if (first) return first.length > 60 ? first.substring(0, 60) + "…" : first;
  return formatToolName(name);
}
