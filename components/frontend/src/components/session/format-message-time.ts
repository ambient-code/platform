/**
 * Formats a Unix-epoch millisecond timestamp into a short time string
 * suitable for chat message badges (e.g. "14:32" or "2:32 PM"
 * depending on the user's locale).
 */
export function formatMessageTime(epochMs: number): string {
  return new Date(epochMs).toLocaleTimeString(undefined, {
    hour: "numeric",
    minute: "2-digit",
  });
}
