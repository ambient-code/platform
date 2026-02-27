/**
 * Formats an ISO timestamp using the browser's locale (e.g. "2/27, 5:34 PM").
 */
export function formatTimestamp(timestamp: string | undefined): string {
  if (!timestamp) return "";

  try {
    const date = new Date(timestamp);
    if (isNaN(date.getTime())) return "";

    return date.toLocaleString(undefined, {
      month: "numeric",
      day: "numeric",
      hour: "numeric",
      minute: "2-digit",
    });
  } catch {
    return "";
  }
}
