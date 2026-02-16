'use client';

import { createContext, useContext, useState, useCallback, type ReactNode } from 'react';

type ApiSource = 'k8s' | 'api-server';

type ApiSourceContextValue = {
  source: ApiSource;
  setSource: (source: ApiSource) => void;
  isApiServer: boolean;
  isK8s: boolean;
  toggle: () => void;
};

const ApiSourceContext = createContext<ApiSourceContextValue | null>(null);

const STORAGE_KEY = 'ambient-api-source';

function getInitialSource(): ApiSource {
  if (typeof window === 'undefined') return 'api-server';
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === 'k8s') return 'k8s';
    if (stored === 'api-server') return 'api-server';
  } catch {}
  return 'api-server';
}

type ApiSourceProviderProps = {
  children: ReactNode;
  defaultSource?: ApiSource;
};

export function ApiSourceProvider({ children, defaultSource }: ApiSourceProviderProps) {
  const [source, setSourceState] = useState<ApiSource>(defaultSource ?? getInitialSource);

  const setSource = useCallback((next: ApiSource) => {
    setSourceState(next);
    try {
      localStorage.setItem(STORAGE_KEY, next);
    } catch {}
  }, []);

  const toggle = useCallback(() => {
    setSource(source === 'k8s' ? 'api-server' : 'k8s');
  }, [source, setSource]);

  return (
    <ApiSourceContext.Provider
      value={{
        source,
        setSource,
        isApiServer: source === 'api-server',
        isK8s: source === 'k8s',
        toggle,
      }}
    >
      {children}
    </ApiSourceContext.Provider>
  );
}

export function useApiSource(): ApiSourceContextValue {
  const ctx = useContext(ApiSourceContext);
  if (!ctx) {
    throw new Error('useApiSource must be used within an ApiSourceProvider');
  }
  return ctx;
}

export function useApiSourceOptional(): ApiSourceContextValue | null {
  return useContext(ApiSourceContext);
}
