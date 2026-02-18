'use client';

/**
 * Unleash feature flag provider.
 * Wraps the app so components can use useFlag() and useVariant() from @unleash/proxy-client-react.
 * Flags are fetched from our Next.js proxy /api/feature-flags (which forwards to Unleash when configured).
 * When Unleash is not configured, the proxy returns empty toggles and all flags are false.
 */

import { FlagProvider } from '@unleash/proxy-client-react';
import { useState, useEffect, type ReactNode } from 'react';

const UNLEASH_APP_NAME = 'ambient-code-platform';

// Get client key from environment or use placeholder
// The placeholder key is used when Unleash is not configured - the SDK requires a non-empty string
const UNLEASH_CLIENT_KEY = process.env.NEXT_PUBLIC_UNLEASH_CLIENT_KEY || 'placeholder-not-configured';

type FeatureFlagProviderProps = {
  children: ReactNode;
};

export function FeatureFlagProvider({ children }: FeatureFlagProviderProps) {
  const [mounted, setMounted] = useState(false);
  const [baseUrl, setBaseUrl] = useState<string | null>(null);

  // Only run on client side after mount to get the correct origin
  useEffect(() => {
    if (typeof window !== 'undefined') {
      // Use environment variable URL if set, otherwise construct from window.location
      const unleashUrl = process.env.NEXT_PUBLIC_UNLEASH_URL || `${window.location.origin}/api/feature-flags`;
      setBaseUrl(unleashUrl);
      setMounted(true);
    }
  }, []);

  // During SSR or before mount, just render children without the provider
  // This avoids the "Invalid URL" error during static generation
  if (!mounted || !baseUrl) {
    return <>{children}</>;
  }

  return (
    <FlagProvider
      config={{
        url: baseUrl,
        clientKey: UNLEASH_CLIENT_KEY,
        appName: UNLEASH_APP_NAME,
        environment: process.env.NEXT_PUBLIC_UNLEASH_ENV_CONTEXT_FIELD || 'development',
      }}
    >
      {children}
    </FlagProvider>
  );
}
