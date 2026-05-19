import { renderHook, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { githubReleasesKeys, useGitHubReleases } from '../use-github-releases';
import { createWrapper } from './test-utils';
import { BACKEND_VERSION } from '../query-keys';

vi.mock('../../api/github-releases', () => ({
  getGitHubReleases: vi.fn().mockResolvedValue([
    {
      id: 1,
      tag_name: 'v1.0.0',
      name: 'Release 1.0.0',
      body: 'Initial release',
      html_url: 'https://github.com/ambient-code/platform/releases/tag/v1.0.0',
      published_at: '2026-01-01T00:00:00Z',
      prerelease: false,
      draft: false,
    },
  ]),
}));

describe('githubReleasesKeys', () => {
  it('includes BACKEND_VERSION prefix', () => {
    expect(githubReleasesKeys.all[0]).toBe(BACKEND_VERSION);
  });

  it('generates correct query keys', () => {
    expect(githubReleasesKeys.all).toEqual([BACKEND_VERSION, 'github-releases']);
    expect(githubReleasesKeys.list()).toEqual([BACKEND_VERSION, 'github-releases', 'list']);
  });
});

describe('useGitHubReleases', () => {
  it('fetches releases and returns data', async () => {
    const { result } = renderHook(() => useGitHubReleases(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toHaveLength(1);
    expect(result.current.data?.[0].tag_name).toBe('v1.0.0');
  });
});
