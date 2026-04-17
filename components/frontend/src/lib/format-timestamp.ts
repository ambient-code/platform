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

/**
 * Formats a Date as a UTC time string with "UTC" suffix (e.g. "5:00 PM UTC").
 */
export function formatTimeUTC(date: Date): string {
  return date.toLocaleString(undefined, {
    hour: "numeric",
    minute: "2-digit",
    timeZone: "UTC",
  }) + " UTC";
}

/**
 * Formats a Date as a local time string with timezone abbreviation (e.g. "1:00 PM EDT").
 */
export function formatTimeLocal(date: Date): string {
  return date.toLocaleString(undefined, {
    hour: "numeric",
    minute: "2-digit",
    timeZoneName: "short",
  });
}

/**
 * Formats a Date showing both UTC and local time (e.g. "5:00 PM UTC (1:00 PM EDT)").
 * If the user's timezone is UTC, only the UTC time is shown.
 */
export function formatScheduleTime(date: Date): string {
  const utc = formatTimeUTC(date);
  const local = formatTimeLocal(date);

  // If local already shows UTC, don't duplicate
  if (local.endsWith("UTC")) {
    return utc;
  }

  return `${utc} (${local})`;
}

/**
 * Formats a Date showing date + both UTC and local time for schedule displays.
 * (e.g. "Feb 27, 5:00 PM UTC (1:00 PM EDT)")
 */
export function formatScheduleDateTime(date: Date): string {
  const datePart = date.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
  });
  const utcTime = date.toLocaleString(undefined, {
    hour: "numeric",
    minute: "2-digit",
    timeZone: "UTC",
  });
  const localTime = formatTimeLocal(date);

  if (localTime.endsWith("UTC")) {
    return `${datePart}, ${utcTime} UTC`;
  }

  return `${datePart}, ${utcTime} UTC (${localTime})`;
}
