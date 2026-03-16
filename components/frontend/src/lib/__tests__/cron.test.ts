import { describe, it, expect } from 'vitest';
import { getCronDescription, getNextRuns } from '../cron';

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
