/**
 * Feature flags via Unleash (optional).
 * Re-export SDK hooks so the app uses a single import path.
 *
 * Usage:
 *   import { useFlag, useVariant, useFlagsStatus } from '@/lib/feature-flags';
 *
 *   const enabled = useFlag('my-feature-name');
 *   const variant = useVariant('experiment-name');
 *   const { flagsReady, flagsError } = useFlagsStatus();
 */

export {
  useFlag,
  useVariant,
  useFlagsStatus,
  useUnleashContext,
} from '@unleash/proxy-client-react';
