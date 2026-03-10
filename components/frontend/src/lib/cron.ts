import cronstrue from "cronstrue";
import { CronExpressionParser } from "cron-parser";

/**
 * Returns a human-readable description of a cron expression.
 * Falls back to the raw expression on parse error.
 */
export function getCronDescription(cronExpr: string): string {
  try {
    return cronstrue.toString(cronExpr);
  } catch {
    return cronExpr;
  }
}

/**
 * Returns the next N run dates for a cron expression.
 * Returns an empty array on parse error.
 */
export function getNextRuns(cronExpr: string, count: number): Date[] {
  try {
    const interval = CronExpressionParser.parse(cronExpr);
    const dates: Date[] = [];
    for (let i = 0; i < count; i++) {
      dates.push(interval.next().toDate());
    }
    return dates;
  } catch {
    return [];
  }
}
