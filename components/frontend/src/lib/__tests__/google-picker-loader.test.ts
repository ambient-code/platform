import { describe, it, expect, vi, beforeEach } from 'vitest';

// We need a fresh module for each test because the loader caches state
// in module-level variables (pickerApiLoaded, gisLoaded).
let loadPickerApi: () => Promise<void>;
let loadGis: () => Promise<void>;
let loadGoogleApis: () => Promise<void>;

beforeEach(async () => {
  vi.resetModules();
  // Reset DOM — remove any previously injected scripts
  document.querySelectorAll('script').forEach((s) => s.remove());

  const mod = await import('../google-picker-loader');
  loadPickerApi = mod.loadPickerApi;
  loadGis = mod.loadGis;
  loadGoogleApis = mod.loadGoogleApis;
});

describe('loadPickerApi', () => {
  it('creates a script element with correct src and calls gapi.load', async () => {
    // Set up gapi mock that will be available after script loads
    const gapiLoadMock = vi.fn(
      (_api: string, opts: { callback: () => void }) => {
        opts.callback();
      },
    );

    const appendChildSpy = vi.spyOn(document.head, 'appendChild').mockImplementation((node) => {
      // Simulate script load
      const script = node as HTMLScriptElement;
      expect(script.src).toContain('https://apis.google.com/js/api.js');
      expect(script.async).toBe(true);

      // Make gapi available before calling onload
      vi.stubGlobal('gapi', { load: gapiLoadMock });
      window.gapi = { load: gapiLoadMock };

      // Trigger onload
      if (script.onload) {
        (script.onload as EventListener)(new Event('load'));
      }
      return node;
    });

    await loadPickerApi();

    expect(appendChildSpy).toHaveBeenCalledTimes(1);
    expect(gapiLoadMock).toHaveBeenCalledWith('picker', expect.objectContaining({
      callback: expect.any(Function),
      onerror: expect.any(Function),
    }));

    appendChildSpy.mockRestore();
  });

  it('returns immediately on second call (caching)', async () => {
    // First call: set up script loading
    const gapiLoadMock = vi.fn(
      (_api: string, opts: { callback: () => void }) => {
        opts.callback();
      },
    );

    const appendChildSpy = vi.spyOn(document.head, 'appendChild').mockImplementation((node) => {
      const script = node as HTMLScriptElement;
      window.gapi = { load: gapiLoadMock };
      if (script.onload) {
        (script.onload as EventListener)(new Event('load'));
      }
      return node;
    });

    await loadPickerApi();
    expect(appendChildSpy).toHaveBeenCalledTimes(1);

    // Second call should skip script creation and gapi.load
    await loadPickerApi();
    // appendChild should not have been called again
    expect(appendChildSpy).toHaveBeenCalledTimes(1);
    // gapi.load should only have been called once
    expect(gapiLoadMock).toHaveBeenCalledTimes(1);

    appendChildSpy.mockRestore();
  });
});

describe('loadGis', () => {
  it('creates script element with correct src', async () => {
    const appendChildSpy = vi.spyOn(document.head, 'appendChild').mockImplementation((node) => {
      const script = node as HTMLScriptElement;
      expect(script.src).toContain('https://accounts.google.com/gsi/client');

      if (script.onload) {
        (script.onload as EventListener)(new Event('load'));
      }
      return node;
    });

    await loadGis();

    expect(appendChildSpy).toHaveBeenCalledTimes(1);

    appendChildSpy.mockRestore();
  });

  it('returns immediately on second call (caching)', async () => {
    const appendChildSpy = vi.spyOn(document.head, 'appendChild').mockImplementation((node) => {
      const script = node as HTMLScriptElement;
      if (script.onload) {
        (script.onload as EventListener)(new Event('load'));
      }
      return node;
    });

    await loadGis();
    await loadGis();

    // Script should only be appended once
    expect(appendChildSpy).toHaveBeenCalledTimes(1);

    appendChildSpy.mockRestore();
  });
});

describe('loadGoogleApis', () => {
  it('calls both loadPickerApi and loadGis', async () => {
    const gapiLoadMock = vi.fn(
      (_api: string, opts: { callback: () => void }) => {
        opts.callback();
      },
    );

    const appendChildSpy = vi.spyOn(document.head, 'appendChild').mockImplementation((node) => {
      const script = node as HTMLScriptElement;
      if (script.src.includes('api.js')) {
        window.gapi = { load: gapiLoadMock };
      }
      if (script.onload) {
        (script.onload as EventListener)(new Event('load'));
      }
      return node;
    });

    await loadGoogleApis();

    // Should have appended two scripts (api.js and gsi/client)
    expect(appendChildSpy).toHaveBeenCalledTimes(2);
    expect(gapiLoadMock).toHaveBeenCalledWith('picker', expect.any(Object));

    appendChildSpy.mockRestore();
  });
});
