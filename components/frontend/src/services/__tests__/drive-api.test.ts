import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  initDriveSetup,
  listFileGrants,
  updateFileGrants,
  getDriveIntegration,
  disconnectDriveIntegration,
  driveQueryKeys,
} from '../drive-api';

function mockResponse(body: unknown, ok = true, status = 200): Response {
  return {
    ok,
    status,
    json: vi.fn().mockResolvedValue(body),
    headers: new Headers(),
    redirected: false,
    statusText: ok ? 'OK' : 'Error',
    type: 'basic',
    url: '',
    clone: vi.fn(),
    body: null,
    bodyUsed: false,
    arrayBuffer: vi.fn(),
    blob: vi.fn(),
    formData: vi.fn(),
    text: vi.fn(),
    bytes: vi.fn(),
  } as unknown as Response;
}

describe('drive-api', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  // ---------------------------------------------------------------------------
  // initDriveSetup
  // ---------------------------------------------------------------------------
  describe('initDriveSetup', () => {
    it('calls fetch with correct URL, method, and body', async () => {
      const payload = { authUrl: 'https://auth.example.com', state: 'abc123' };
      vi.mocked(fetch).mockResolvedValue(mockResponse(payload));

      const result = await initDriveSetup('my-project', 'granular', 'https://redirect.example.com');

      expect(fetch).toHaveBeenCalledWith(
        '/api/projects/my-project/integrations/google-drive/setup',
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ permissionScope: 'granular', redirectUri: 'https://redirect.example.com' }),
        },
      );
      expect(result).toEqual(payload);
    });

    it('encodes project name in URL', async () => {
      vi.mocked(fetch).mockResolvedValue(mockResponse({ authUrl: '', state: '' }));

      await initDriveSetup('project with spaces', 'full', 'https://example.com');

      expect(fetch).toHaveBeenCalledWith(
        '/api/projects/project%20with%20spaces/integrations/google-drive/setup',
        expect.any(Object),
      );
    });
  });

  // ---------------------------------------------------------------------------
  // listFileGrants
  // ---------------------------------------------------------------------------
  describe('listFileGrants', () => {
    it('calls GET with correct URL and returns parsed response', async () => {
      const payload = { files: [{ id: 'f1', fileName: 'doc.pdf' }] };
      vi.mocked(fetch).mockResolvedValue(mockResponse(payload));

      const result = await listFileGrants('my-project');

      expect(fetch).toHaveBeenCalledWith(
        '/api/projects/my-project/integrations/google-drive/files',
        { method: 'GET' },
      );
      expect(result).toEqual(payload);
    });
  });

  // ---------------------------------------------------------------------------
  // updateFileGrants
  // ---------------------------------------------------------------------------
  describe('updateFileGrants', () => {
    it('calls PUT with correct URL and body', async () => {
      const files = [
        { id: 'f1', name: 'doc.pdf', mimeType: 'application/pdf', url: 'https://drive.google.com/f1', sizeBytes: 1024, isFolder: false },
      ];
      const payload = { files: [{ id: 'f1', fileName: 'doc.pdf', status: 'active' }] };
      vi.mocked(fetch).mockResolvedValue(mockResponse(payload));

      const result = await updateFileGrants('my-project', files);

      expect(fetch).toHaveBeenCalledWith(
        '/api/projects/my-project/integrations/google-drive/files',
        {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ files }),
        },
      );
      expect(result).toEqual(payload);
    });
  });

  // ---------------------------------------------------------------------------
  // getDriveIntegration
  // ---------------------------------------------------------------------------
  describe('getDriveIntegration', () => {
    it('calls GET and returns integration', async () => {
      const payload = {
        id: 'int-1',
        projectName: 'my-project',
        permissionScope: 'granular',
        status: 'active',
        fileCount: 5,
        createdAt: '2025-01-01T00:00:00Z',
        updatedAt: '2025-01-01T00:00:00Z',
      };
      vi.mocked(fetch).mockResolvedValue(mockResponse(payload));

      const result = await getDriveIntegration('my-project');

      expect(fetch).toHaveBeenCalledWith(
        '/api/projects/my-project/integrations/google-drive',
        { method: 'GET' },
      );
      expect(result).toEqual(payload);
    });
  });

  // ---------------------------------------------------------------------------
  // disconnectDriveIntegration
  // ---------------------------------------------------------------------------
  describe('disconnectDriveIntegration', () => {
    it('calls DELETE and returns response', async () => {
      const payload = { success: true };
      vi.mocked(fetch).mockResolvedValue(mockResponse(payload));

      const result = await disconnectDriveIntegration('my-project');

      expect(fetch).toHaveBeenCalledWith(
        '/api/projects/my-project/integrations/google-drive',
        { method: 'DELETE' },
      );
      expect(result).toEqual(payload);
    });
  });

  // ---------------------------------------------------------------------------
  // Error handling
  // ---------------------------------------------------------------------------
  describe('error handling', () => {
    it('throws ApiError when response is not ok', async () => {
      vi.mocked(fetch).mockResolvedValue(
        mockResponse({ message: 'Not found' }, false, 404),
      );

      await expect(getDriveIntegration('missing')).rejects.toThrow('Not found');
    });

    it('throws ApiError with default message when body has no message field', async () => {
      vi.mocked(fetch).mockResolvedValue(
        mockResponse({ error: 'something' }, false, 500),
      );

      await expect(getDriveIntegration('bad')).rejects.toThrow(
        'Request failed with status 500',
      );
    });

    it('throws ApiError when body is not valid JSON', async () => {
      const resp = {
        ok: false,
        status: 502,
        json: vi.fn().mockRejectedValue(new SyntaxError('bad json')),
      } as unknown as Response;
      vi.mocked(fetch).mockResolvedValue(resp);

      await expect(getDriveIntegration('bad')).rejects.toThrow(
        'Request failed with status 502',
      );
    });
  });

  // ---------------------------------------------------------------------------
  // Query key factory
  // ---------------------------------------------------------------------------
  describe('driveQueryKeys', () => {
    it('generates correct query keys', () => {
      expect(driveQueryKeys.all).toEqual(['drive-integration']);
      expect(driveQueryKeys.integration('proj')).toEqual([
        'drive-integration',
        'integration',
        'proj',
      ]);
      expect(driveQueryKeys.fileGrants('proj')).toEqual([
        'drive-integration',
        'file-grants',
        'proj',
      ]);
      expect(driveQueryKeys.pickerToken('proj')).toEqual([
        'drive-integration',
        'picker-token',
        'proj',
      ]);
    });
  });
});
