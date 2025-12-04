/**
 * Formats an ISO timestamp into a human-readable format
 * @param timestamp ISO 8601 timestamp string
 * @returns Formatted time string (HH:MM:SS)
 */
export function formatTimestamp(timestamp: string | undefined): string {
  if (!timestamp) return "";

  try {
    const date = new Date(timestamp);

    // Check if date is valid
    if (isNaN(date.getTime())) {
      return "";
    }

    // Format as HH:MM:SS in local timezone
    const hours = date.getHours().toString().padStart(2, "0");
    const minutes = date.getMinutes().toString().padStart(2, "0");
    const seconds = date.getSeconds().toString().padStart(2, "0");

    return `${hours}:${minutes}:${seconds}`;
  } catch {
    return "";
  }
}

/**
 * Formats an ISO timestamp into a human-readable format with date
 * @param timestamp ISO 8601 timestamp string
 * @returns Formatted datetime string (MM/DD HH:MM:SS)
 */
export function formatTimestampWithDate(timestamp: string | undefined): string {
  if (!timestamp) return "";

  try {
    const date = new Date(timestamp);

    // Check if date is valid
    if (isNaN(date.getTime())) {
      return "";
    }

    const month = (date.getMonth() + 1).toString().padStart(2, "0");
    const day = date.getDate().toString().padStart(2, "0");
    const hours = date.getHours().toString().padStart(2, "0");
    const minutes = date.getMinutes().toString().padStart(2, "0");
    const seconds = date.getSeconds().toString().padStart(2, "0");

    return `${month}/${day} ${hours}:${minutes}:${seconds}`;
  } catch {
    return "";
  }
}
