import { describe, it, expect } from 'vitest';
import { getCronDescription, getCronDescriptionWithLocal, getNextRuns } from '../cron';

describe('getCronDescription', () => {
  it('returns human-readable description for standard cron expressions', () => {
    expect(getCronDescription('0 9 * * *')).toMatch(/09:00/i);
    expect(getCronDescription('*/5 * * * *')).toMatch(/5 minutes/i);
    expect(getCronDescription('0 0 * * 0')).toMatch(/sunday/i);
    expect(getCronDescription('0 12 1 * *')).toMatch(/12:00/i);
    expect(getCronDescription('30 14 * * 1-5')).toMatch(/02:30 PM/i);
  });

  it('returns the raw expression for invalid cron expressions', () => {
    expect(getCronDescription('not-a-cron')).toBe('not-a-cron');
    expect(getCronDescription('')).toBe('');
    expect(getCronDescription('1 2 3 4 5 6 7 8')).toBe('1 2 3 4 5 6 7 8');
  });
});

describe('getNextRuns', () => {
  it('returns the requested number of future dates for valid cron', () => {
    const runs = getNextRuns('0 * * * *', 3);
    expect(runs).toHaveLength(3);
    for (const date of runs) {
      expect(date).toBeInstanceOf(Date);
      expect(date.getTime()).toBeGreaterThan(Date.now());
    }
    expect(runs[0].getTime()).toBeLessThan(runs[1].getTime());
  });

  it('returns an empty array for invalid cron', () => {
    expect(getNextRuns('invalid', 3)).toEqual([]);
  });
});

describe('getCronDescriptionWithLocal', () => {
  it('includes UTC label for daily schedule', () => {
    const result = getCronDescriptionWithLocal('0 9 * * *');
    expect(result).toContain('UTC');
  });

  it('appends parenthesized local time when browser is not in UTC', () => {
    const result = getCronDescriptionWithLocal('0 9 * * *');
    const isUtcEnv = new Date().getTimezoneOffset() === 0;
    if (isUtcEnv) {
      expect(result).not.toContain('(');
    } else {
      expect(result).toMatch(/UTC\s*\([^)]+\)$/);
    }
  });

  it('returns plain description without timezone for sub-daily schedules', () => {
    const everyMinute = getCronDescriptionWithLocal('* * * * *');
    expect(everyMinute).not.toContain('UTC');
    expect(everyMinute).toBe(getCronDescription('* * * * *'));

    const every15Min = getCronDescriptionWithLocal('*/15 * * * *');
    expect(every15Min).not.toContain('UTC');
    expect(every15Min).toBe(getCronDescription('*/15 * * * *'));

    const everyHour = getCronDescriptionWithLocal('0 * * * *');
    expect(everyHour).not.toContain('UTC');
    expect(everyHour).toBe(getCronDescription('0 * * * *'));

    const every2Hours = getCronDescriptionWithLocal('0 */2 * * *');
    expect(every2Hours).not.toContain('UTC');
    expect(every2Hours).toBe(getCronDescription('0 */2 * * *'));
  });

  it('falls back to plain description for invalid cron', () => {
    const result = getCronDescriptionWithLocal('not-valid');
    expect(result).toBe('not-valid');
  });

  it('includes UTC label for weekday schedule', () => {
    const result = getCronDescriptionWithLocal('30 14 * * 1-5');
    expect(result).toContain('UTC');
  });

  it('includes UTC label for weekly schedule', () => {
    const result = getCronDescriptionWithLocal('0 9 * * 1');
    expect(result).toContain('UTC');
  });

  it('includes UTC label for monthly schedule', () => {
    const result = getCronDescriptionWithLocal('0 12 1 * *');
    expect(result).toContain('UTC');
  });

  it('inserts UTC after time portion for both 12h and 24h formats', () => {
    const result = getCronDescriptionWithLocal('30 14 * * *');
    expect(result).toMatch(/14:30\s*(PM\s*)?UTC|02:30\s*PM\s*UTC/);
  });
});
