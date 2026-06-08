export function safeReturnTo(value: string | null | undefined): string {
  if (!value) return "/";
  try {
    const parsed = new URL(value, "http://localhost");
    if (parsed.origin !== "http://localhost") return "/";
    return parsed.pathname + parsed.search;
  } catch {
    return "/";
  }
}
